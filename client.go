/*
 * Copyright (c) 2024 Lynn <lynnplus90@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mqtt

import (
	"context"
	"errors"
	"fmt"
	"github.com/lynnplus/go-mqtt/packets"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ClientListener struct {
	OnConnected      func(client *Client, reconnection bool, ack *packets.Connack)
	OnConnectionLost func(client *Client, err error)
	// OnConnectFailed is called only when dialing fails or the server refuses the connection
	OnConnectFailed func(client *Client, err error)
	// OnServerDisconnect is called only when a packets.DISCONNECT is received from server
	OnServerDisconnect func(client *Client, pkt *packets.Disconnect)
	OnClientError      func(client *Client, err error)
}

type ClientConfig struct {
	ManualACK bool
	Logger    Logger
	Pinger    Pinger
	Router    Router
	Auther    Auther
	ClientListener
	Session SessionState
	//packet send timeout
	PacketTimeout  time.Duration
	ReConnector    ReConnector
	ConnectTimeout time.Duration
}

type Client struct {
	dialer     Dialer
	conn       *Conn
	bufferSize int
	version    packets.ProtocolVersion
	config     ClientConfig
	properties *ServerProperties
	connState  ConnState

	clientId          string
	sendQueue         chan *packetInfo
	ctxCancelFunc     context.CancelFunc
	workers           sync.WaitGroup
	publishHandleChan chan *packets.Publish
	closeSign         chan int
	closing           atomic.Bool
}

var (
	ErrInvalidArguments = errors.New("invalid argument")
	ErrNotConnected     = errors.New("client not connected")
)

func NewClient(dialer Dialer, config ClientConfig) *Client {
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 5 * time.Second
	}
	if config.Pinger == nil {
		config.Pinger = NewDefaultPinger()
	}
	if config.Session == nil {
		config.Session = NewDefaultSession()
	}
	if config.Logger == nil {
		config.Logger = &EmptyLogger{}
	}
	if config.Router == nil {
		config.Router = &DefaultRouter{}
	}
	if config.PacketTimeout == 0 {
		config.PacketTimeout = time.Second * 5
	}
	//if config.ReConnector == nil {
	//	config.ReConnector = NewAutoReConnector()
	//}
	return &Client{dialer: dialer, version: packets.ProtocolVersion5,
		bufferSize: 4096,
		config:     config,
		closeSign:  make(chan int, 1),
		properties: NewServerProperties(),
		sendQueue:  make(chan *packetInfo, 30)}
}

type connectionInfo struct {
	ctx    context.Context //conn context
	dialer Dialer
	pkt    *packets.Connect
	err    chan error
	resp   chan *packets.Connack
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connState.IsConnected()
}

// StartConnect
func (c *Client) StartConnect(ctx context.Context, pkt *packets.Connect) error {
	if !pkt.ProtocolVersion.IsValid() {
		return fmt.Errorf("protocol version is invalid")
	}
	if !c.connState.CompareAndSwap(StatusNone, StatusConnecting) {
		return fmt.Errorf("connection already in %s", c.connState)
	}
	c.closeSign = make(chan int, 1)
	cp := pkt.Clone()
	info := &connectionInfo{
		ctx:    ctx,
		dialer: c.dialer,
		pkt:    cp,
	}
	if c.config.ReConnector != nil {
		c.config.ReConnector.Reset()
	}
	go c.internalConnect(info)
	return nil
}

func (c *Client) Connect(ctx context.Context, pkt *packets.Connect) (*packets.Connack, error) {
	if !pkt.ProtocolVersion.IsValid() || pkt.ProtocolName == "" {
		return nil, fmt.Errorf("protocol version or name is invalid")
	}
	if !c.connState.CompareAndSwap(StatusNone, StatusConnecting) {
		return nil, fmt.Errorf("connection already in %s", c.connState)
	}
	c.closeSign = make(chan int, 1)
	cp := pkt.Clone()
	info := &connectionInfo{
		ctx:    ctx,
		dialer: c.dialer,
		pkt:    cp,
		err:    make(chan error, 1),
		resp:   make(chan *packets.Connack, 1),
	}
	if c.config.ReConnector != nil {
		c.config.ReConnector.Reset()
	}
	go c.internalConnect(info)

	select {
	case <-c.closeSign:
		return nil, errors.New("client disconnected")
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-info.err:
		return nil, err
	case ack := <-info.resp:
		return ack, nil
	}
}

func (c *Client) internalConnect(info *connectionInfo) {
	var err error
	defer func() {
		if err != nil {
			go c.connectFailed(info, err)
		}
	}()
	if c.connState.Load() != StatusConnecting {
		err = errors.New("client disconnecting")
		return
	}

	var conn *Conn
	//StatusConnecting
	conn, err = attemptConnection(info.ctx, info.dialer, c.bufferSize, c)
	if err != nil {
		return
	}
	_ = conn.conn.SetDeadline(time.Now().Add(c.config.ConnectTimeout))
	if err = conn.flushWrite(info.pkt); err != nil {
		_ = conn.conn.Close()
		return
	}
	var packet packets.Packet
	for {
		if c.connState.Load() != StatusConnecting {
			_ = conn.conn.Close()
			err = errors.New("client disconnecting")
			return
		}
		packet, err = conn.readPacket()
		if err != nil {
			_ = conn.conn.Close()
			return
		}
		switch temp := packet.(type) {
		case *packets.Connack:
			if temp.ReasonCode != packets.Success {
				msg := ""
				if temp.Properties != nil {
					msg = temp.Properties.ReasonString
				}
				err = packets.NewReasonCodeError(temp.ReasonCode, msg)
				_ = conn.conn.Close()
				return
			}
			if temp.Properties != nil && temp.Properties.AuthMethod != "" {
				if c.config.Auther != nil {
					err = c.config.Auther.Authenticated(info.pkt.ClientID, temp)
				}
			}
			if err != nil {
				_ = conn.conn.Close()
				return
			}

			if !c.connState.CompareAndSwap(StatusConnecting, StatusConnected) {
				_ = conn.conn.Close()
				err = fmt.Errorf("connection already in %s", c.connState)
				return
			}
			c.connectComplete(conn, info.pkt, temp)
			if info.resp != nil {
				info.resp <- temp
			}
			return
		case *packets.Auth:
			if c.config.Auther == nil {
				err = errors.New("auther not found when receiving server authentication request")
				_ = conn.conn.Close()
				return
			}
			packet, err = c.config.Auther.Authenticate(info.pkt.ClientID, temp)
			if err != nil {
				err = fmt.Errorf("authenticate handle err: %w", err)
				_ = conn.conn.Close()
				return
			}
			if err = conn.flushWrite(packet); err != nil {
				err = fmt.Errorf("authenticate write err: %w", err)
				_ = conn.conn.Close()
				return
			}
		default:
			err = fmt.Errorf("unknown packet type(%v) on connecting", packet.Type())
			_ = conn.conn.Close()
			return
		}
	}
}

func (c *Client) connectFailed(info *connectionInfo, err error) {
	c.config.Logger.Debug("connect failed: %v", err)
	if c.config.ReConnector == nil || c.connState.Load() != StatusConnecting {
		if info.err != nil {
			info.err <- err
			close(info.err)
		}
		return
	}
	d, t := c.config.ReConnector.ConnectionFailure(info.dialer, err)
	if t == nil {
		if info.err != nil {
			info.err <- err
			close(info.err)
		}
		if !c.connState.CompareAndSwap(StatusConnecting, StatusNone) {
			c.config.Logger.Error("Connection status switching failed, currently: %s", c.connState)
		}
		return
	}
	defer t.Stop()
	info.dialer = d
	select {
	case <-c.closeSign:
		if info.err != nil {
			info.err <- errors.New("client disconnected")
			close(info.err)
		}
	case <-info.ctx.Done():
		if info.err != nil {
			info.err <- info.ctx.Err()
			close(info.err)
		}
		if !c.connState.CompareAndSwap(StatusConnecting, StatusNone) {
			c.config.Logger.Error("Connection status switching failed, currently: %s", c.connState)
		}
	case <-t.C:
		c.config.Logger.Debug("start re-connect")
		go c.internalConnect(info)
	}
}

func (c *Client) connectComplete(conn *Conn, pkt *packets.Connect, ack *packets.Connack) {
	if c.config.ReConnector != nil {
		c.config.ReConnector.Reset()
	}

	clientId := pkt.ClientID
	keepAlive := time.Duration(pkt.KeepAlive) * time.Second
	if ack.Properties != nil {
		if ack.Properties.ServerKeepAlive != nil {
			keepAlive = time.Duration(*ack.Properties.ServerKeepAlive) * time.Second
		}
		if ack.Properties.AssignedClientID != "" {
			clientId = ack.Properties.AssignedClientID
		}
	}
	c.properties.ReconfigureFromResponse(ack)
	//callConnectComplete(c, connack)

	c.publishHandleChan = make(chan *packets.Publish, c.properties.ReceiveMaximum)

	clientCtx, cancelFunc := context.WithCancel(context.Background())
	c.ctxCancelFunc = cancelFunc
	c.conn = conn
	c.clientId = clientId
	c.conn.run(clientCtx, c.sendQueue)

	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		err := c.config.Pinger.Run(clientCtx, keepAlive, c)
		c.config.Logger.Debug("client pinger exit,err(%v)", err)
		if err != nil {
			go c.occurredError(fmt.Errorf("pinger run error: %w", err))
		}
	}()
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		c.handleReceivedPublish(clientCtx)
		c.config.Logger.Debug("client publish handler exit")
	}()
}

func (c *Client) SendPing(ctx context.Context) error {
	return c.awaitSendComplete(ctx, &packets.Pingreq{})
}

func (c *Client) verifyPublish(pkt *packets.Publish) error {
	//TODO mqtt v3 impl
	if pkt.QoS > c.properties.MaximumQoS {
		return fmt.Errorf("%w: cannot send Publish with QoS %d, server maximum QoS is %d", ErrInvalidArguments, pkt.QoS, c.properties.MaximumQoS)
	}
	if pkt.QoS == 0 && pkt.Duplicate {
		return fmt.Errorf("%w: cannot send Publish with qos 0 and Duplicate set to true", ErrInvalidArguments)
	}
	if pkt.Retain && !c.properties.RetainAvailable {
		return fmt.Errorf("%w: cannot send Publish with Retain flag,server does not support retained messages", ErrInvalidArguments)
	}
	if pkt.Properties != nil {
		if pkt.Topic == "" && pkt.Properties.TopicAlias == nil {
			return fmt.Errorf("%w: cannot send Publish without topic and without topic alias set", ErrInvalidArguments)
		}
		if pkt.Properties.TopicAlias != nil && c.properties.TopicAliasMaximum < *(pkt.Properties.TopicAlias) {
			return fmt.Errorf("%w: topic alias exceeds server limit,topic alias maximum is %v", ErrInvalidArguments, c.properties.TopicAliasMaximum)
		}
	}
	return nil
}

// PublishNR is used to send a packet without ack response to the server, and its qos will be forced to 0
func (c *Client) PublishNR(ctx context.Context, pkt *packets.Publish) error {
	pkt.QoS = 0

	err := c.awaitSendComplete(ctx, pkt)
	if err != nil {
		return err
	}
	c.config.Pinger.Ping()
	return err
}

// Subscribe sends a subscription message to the server,
// blocking and waiting for the server to respond to Suback or a timeout occurs.
// The function will return the server's response(packets.Suback) and any errors.
//
// Note: that the function does not check the error code inside the Suback.
func (c *Client) Subscribe(ctx context.Context, pkt *packets.Subscribe) (*packets.Suback, error) {
	if !c.connState.IsConnected() {
		return nil, ErrNotConnected
	}

	if !c.properties.SubIDAvailable && pkt.Properties != nil && pkt.Properties.SubscriptionID != nil {
		return nil, fmt.Errorf("%w: cannot send subscribe with SubscriptionID set, server does not support SubscriptionID", ErrInvalidArguments)
	}

	if !c.properties.WildcardSubAvailable || !c.properties.SharedSubAvailable {
		for _, sub := range pkt.Subscriptions {
			if !c.properties.WildcardSubAvailable && strings.ContainsAny(sub.Topic, "+#") {
				return nil, fmt.Errorf("%w: cannot subscribe to %s, server does not support wildcards", ErrInvalidArguments, sub.Topic)
			}
			if !c.properties.SharedSubAvailable && strings.HasPrefix(sub.Topic, "$share") {
				return nil, fmt.Errorf("%w: cannont subscribe to %s, server does not support shared subscriptions", ErrInvalidArguments, sub.Topic)
			}
		}
	}

	resp, err := c.config.Session.SubmitPacket(pkt)
	if err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithTimeout(ctx, c.config.PacketTimeout)
	defer cancel()

	if err = c.awaitSendComplete(subCtx, pkt); err != nil {
		if e := c.config.Session.RevokePacket(pkt); e != nil {
			c.config.Logger.Error("session revoke packet(Subscribe) error:%v", e)
		}
		return nil, err
	}
	c.config.Pinger.Ping()

	var respPkt packets.Packet
	select {
	case <-subCtx.Done():
		return nil, subCtx.Err()
	case respPkt = <-resp:
		if respPkt.Type() != packets.SUBACK {
			return nil, errors.New("invalid packet type received, expected SUBACK")
		}
	}
	ack := respPkt.(*packets.Suback)
	return ack, nil
}

func (c *Client) Unsubscribe(ctx context.Context, pkt *packets.Unsubscribe) (*packets.Unsuback, error) {
	resp, err := c.config.Session.SubmitPacket(pkt)
	if err != nil {
		return nil, err
	}
	if err = c.awaitSendComplete(ctx, pkt); err != nil {
		return nil, err
	}
	c.config.Pinger.Ping()
	var respPkt packets.Packet
	select {
	//TODO timeout ctx
	case <-ctx.Done():
	case respPkt = <-resp:
		if respPkt.Type() != packets.UNSUBACK {
			return nil, errors.New("invalid packet type received, expected UNSUBACK")
		}
	}
	ack := respPkt.(*packets.Unsuback)
	return ack, nil
}

// Disconnect is used to send Disconnect data packets to the MQTT server.
// The data packets use reason code 0. Regardless of whether it is sent successfully or not,
// the connection will be disconnected and no reconnection attempt will be made.
func (c *Client) Disconnect() error {
	return c.DisconnectWith(&packets.Disconnect{})
}

// DisconnectWith is used to send Disconnect data packets to the MQTT server.
// Regardless of whether it is sent successfully or not,
// the connection will be disconnected and no reconnection attempt will be made.
func (c *Client) DisconnectWith(pkt *packets.Disconnect) error {
	if c.connState.Load() == StatusNone {
		return nil
	}
	if c.connState.CompareAndSwap(StatusConnecting, StatusDisconnecting) {
		close(c.closeSign)
		return nil
	}

	if !c.connState.CompareAndSwap(StatusConnected, StatusDisconnecting) {
		old := c.connState.Swap(StatusDisconnecting)
		if old == StatusConnecting {
			close(c.closeSign)
			c.connState.Store(StatusNone)
			return nil
		}
	}

	if len(c.sendQueue) < cap(c.sendQueue) {
		ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
		defer cf()
		info := &packetInfo{packet: pkt, err: make(chan error, 1)}
		c.sendQueue <- info
		select {
		case <-ctx.Done():
		case e := <-info.err:
			c.config.Logger.Debug("write disconnect packet, err(%v)", e)
		}
	}
	c.shutdown()
	return nil
}

// awaitSendComplete sends the packet to the write queue and waits for the io write to complete.
func (c *Client) awaitSendComplete(ctx context.Context, pkt packets.Packet) error {
	info := &packetInfo{packet: pkt, err: make(chan error, 1)}
	if !c.connState.IsConnected() {
		return ErrNotConnected
	}
	//TODO Whether the send Queue will be full under certain circumstances
	if len(c.sendQueue) == cap(c.sendQueue) {
		return errors.New("send queue is full")
	}
	c.sendQueue <- info

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-info.err:
		return e
	}
}

func (c *Client) incoming(ctx context.Context, pkt packets.Packet) {
	c.config.Logger.Debug("incoming packet %s %v", pkt.Type(), pkt.ID())
	switch pkt.Type() {
	case packets.PUBLISH:
		pub := pkt.(*packets.Publish)
		c.config.Logger.Debug("incoming publish packet %v %v %v \n", pkt.ID(), pub.QoS, pub.Topic)
		if pub.QoS == 0 {
			select {
			case <-ctx.Done():
				return
			case c.publishHandleChan <- pub:
			}
			return
		}

		c.config.Logger.Error("qos 1 or 2 is not implemented")
	case packets.PUBACK:
	case packets.PUBREC:
	case packets.PUBREL:
	case packets.PUBCOMP:
	case packets.SUBACK, packets.UNSUBACK:
		err := c.config.Session.ResponsePacket(pkt)
		if err != nil {
			c.config.Logger.Error("incoming response error: %v", err)
		}
	case packets.PINGRESP:
		c.config.Pinger.Pong()
	case packets.DISCONNECT:
		c.serverDisconnect(pkt.(*packets.Disconnect))
	case packets.AUTH:

	default:
		go c.occurredError(fmt.Errorf("received unexpected %s for Packet", pkt.Type()))
	}
}

func (c *Client) handleReceivedPublish(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pkt, ok := <-c.publishHandleChan:
			if !ok {
				return
			}
			msgCtx := &pubMsgContext{
				client: c,
				packet: pkt,
			}
			c.config.Router.Route(msgCtx)
		}
	}
}

func (c *Client) serverDisconnect(pkt *packets.Disconnect) {
	c.config.Logger.Debug("server initiates disconnection: %s", pkt.ReasonCode.String())
	if pkt.Properties != nil && pkt.Properties.ReasonString != "" {
		c.config.Logger.Debug("disconnection reason: %s", pkt.Properties.ReasonString)
	}
	c.config.Logger.Debug("start shutdown,state:%v", c.connState)
	if c.connState.CompareAndSwap(StatusConnected, StatusDisconnecting) {
		c.shutdown()
	}
}

func (c *Client) shutdown() {
	if !c.closing.CompareAndSwap(false, true) {
		return
	}
	close(c.closeSign)
	if c.ctxCancelFunc != nil {
		c.ctxCancelFunc()
	}
	if c.conn != nil {
		if err := c.conn.close(); err != nil {
			c.config.Logger.Debug("close conn err:%v", err)
		}
	}
	c.workers.Wait()
	close(c.publishHandleChan)
	c.closing.Store(false)
	c.config.Logger.Debug("shutdown finished")
}

// An exception occurred on the client
func (c *Client) occurredError(err error) {
	if c.connState.Load() == StatusDisconnecting {
		return
	}
	c.config.Logger.Error("an unexpected error occurred: %v", err)
	var dis *packets.Disconnect
	re := &packets.ReasonCodeError{}
	if errors.As(err, &re) {
		dis = packets.NewDisconnect(re.Code, "")
	} else {
		dis = packets.NewDisconnect(packets.UnspecifiedError, err.Error())
	}
	c.config.Logger.Debug("disconnect by err: %v", err)
	_ = c.DisconnectWith(dis)
}
