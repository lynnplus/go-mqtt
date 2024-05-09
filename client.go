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
	Pinger Pinger
	ClientListener
	Session SessionState
}

type Client struct {
	dialer      Dialer
	conn        *Conn
	bufferSize  int
	version     packets.ProtocolVersion
	config      ClientConfig
	properties  *ServerProperties
	connState   ConnState
	autoConnect bool

	clientId      string
	sendQueue     chan *packetInfo
	ctxCancelFunc context.CancelFunc
	workers       sync.WaitGroup
}

var (
	ErrInvalidArguments = errors.New("invalid argument")
)

func NewClient(dialer Dialer, config ClientConfig) *Client {
	if config.Pinger == nil {
		config.Pinger = NewDefaultPinger()
	}
	if config.Session == nil {
		config.Session = NewDefaultSession()
	}
	return &Client{dialer: dialer, version: packets.ProtocolVersion5,
		bufferSize: 4096,
		config:     config,
		properties: NewServerProperties(),
		sendQueue:  make(chan *packetInfo, 30)}
}

// StartConnect connects to the MQTT server and sends packets.Connect packets.
// If the connection fails, it will automatically retry until the connection is successful.
// If Disconnect is called, retries will stop
func (c *Client) StartConnect(pkt *packets.Connect) error {
	if !pkt.ProtocolVersion.IsValid() {
		return fmt.Errorf("protocol version is invalid")
	}
	if !c.connState.CompareAndSwap(StatusNone, StatusConnecting) {
		return fmt.Errorf("connection already in %s", c.connState)
	}
	c.autoConnect = true
	//TODO copy pkt

	return nil

}

func (c *Client) Connect(ctx context.Context, pkt *packets.Connect) (*packets.Connack, error) {
	if !pkt.ProtocolVersion.IsValid() {
		return nil, fmt.Errorf("protocol version is invalid")
	}
	if !c.connState.CompareAndSwap(StatusNone, StatusConnecting) {
		return nil, fmt.Errorf("connection already in %s", c.connState)
	}
	if pkt.ProtocolName == "" {
		switch pkt.ProtocolVersion {
		case packets.ProtocolVersion5, packets.ProtocolVersion311:
			pkt.ProtocolName = packets.ProtocolMQTT
		default:
			pkt.ProtocolName = packets.ProtocolMQIsdp
		}
	}
	conn, err := attemptConnection(ctx, c.dialer, c.bufferSize, c)
	if err != nil {
		callConnectFailed(c, err)
		return nil, err
	}
	c.clientId = pkt.ClientID
	keepAlive := time.Duration(pkt.KeepAlive) * time.Second
	connack, err := connect(conn, pkt)
	if err != nil {
		_ = conn.close()
		callConnectFailed(c, err)
		return nil, err
	}
	if connack.Properties != nil {
		if connack.Properties.ServerKeepAlive != nil {
			keepAlive = time.Duration(*connack.Properties.ServerKeepAlive) * time.Second
		}
		if connack.Properties.AssignedClientID != "" {
			c.clientId = connack.Properties.AssignedClientID
		}
	}
	c.properties.ReconfigureFromResponse(connack)
	callConnectComplete(c, connack)

	fmt.Println(time.Now().Local())

	_ = conn.conn.SetDeadline(time.Time{})

	clientCtx, cancelFunc := context.WithCancel(context.Background())
	c.ctxCancelFunc = cancelFunc
	c.conn = conn
	c.conn.run(clientCtx, c.sendQueue)

	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		err := c.config.Pinger.Run(clientCtx, keepAlive, c)
		fmt.Println("pinger exit")
		if err != nil {
			go c.occurredError(fmt.Errorf("pinger run error: %w", err))
		}
	}()

	return connack, nil
}

func callConnectFailed(c *Client, err error) {
	if c.config.OnConnectFailed != nil {
		go c.config.OnConnectFailed(c, err)
	}
}

func callConnectComplete(c *Client, pkt *packets.Connack) {
	if c.config.OnConnected != nil {
		go c.config.OnConnected(c, false, pkt)
	}
}

func (c *Client) SendPing(ctx context.Context) error {
	return c.awaitSendComplete(ctx, &packets.Pingreq{})
}

func (c *Client) Publish(ctx context.Context, pkt *packets.Publish) error {
	if pkt.QoS > c.properties.MaximumQoS {
		return fmt.Errorf("%w: cannot send Publish with QoS %d, server maximum QoS is %d", ErrInvalidArguments, pkt.QoS, c.properties.MaximumQoS)
	}
	//if p.Properties != nil && p.Properties.TopicAlias != nil {
	//	if c.serverProps.TopicAliasMaximum > 0 && *p.Properties.TopicAlias > c.serverProps.TopicAliasMaximum {
	//		return nil, fmt.Errorf("%w: cannot send publish with TopicAlias %d, server topic alias maximum is %d", ErrInvalidArguments, *p.Properties.TopicAlias, c.serverProps.TopicAliasMaximum)
	//	}
	//}
	//if !c.serverProps.RetainAvailable && p.Retain {
	//	return nil, fmt.Errorf("%w: cannot send Publish with retain flag set, server does not support retained messages", ErrInvalidArguments)
	//}
	//if (p.Properties == nil || p.Properties.TopicAlias == nil) && p.Topic == "" {
	//	return nil, fmt.Errorf("%w: cannot send a publish with no TopicAlias and no Topic set", ErrInvalidArguments)
	//}

	err := c.awaitSendComplete(ctx, pkt)
	if err != nil {
		go c.occurredError(err)
		return err
	}
	c.config.Pinger.Ping()
	return err
}

func (c *Client) Subscribe(ctx context.Context, pkt *packets.Subscribe) (*packets.Suback, error) {
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
	if err = c.awaitSendComplete(ctx, pkt); err != nil {
		return nil, err
	}
	c.config.Pinger.Ping()

	var respPkt packets.Packet
	select {
	//TODO timeout ctx
	case <-ctx.Done():
	case respPkt = <-resp:
		if respPkt.Type() != packets.SUBACK {
			return nil, errors.New("invalid packet type received, expected SUBACK")
		}
	}
	ack := respPkt.(*packets.Suback)
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
	err := c.awaitSendComplete(context.Background(), pkt)
	//TODO close conn
	return err
}

func (c *Client) awaitSendComplete(ctx context.Context, pkt packets.Packet) error {
	info := &packetInfo{packet: pkt, err: make(chan error, 1)}
	err := c.sendPacketToQueue(info)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-info.err:
		return e
	}
}

func (c *Client) sendPacketToQueue(pkt *packetInfo) error {
	//TODO check connected state
	if len(c.sendQueue) == cap(c.sendQueue) {
		return errors.New("send queue is full")
	}
	c.sendQueue <- pkt
	return nil
}

func (c *Client) incoming(pkt packets.Packet) {

	fmt.Println("incoming packet", pkt.Type())
	switch pkt.Type() {
	case packets.PUBLISH:
	case packets.PUBACK:
	case packets.PUBREC:
	case packets.PUBREL:
	case packets.PUBCOMP:
	case packets.SUBACK:
		err := c.config.Session.ResponsePacket(pkt)
		if err != nil {
			//TODO err handle
		}
	case packets.UNSUBACK:
	case packets.PINGRESP:
		c.config.Pinger.Pong()
	case packets.DISCONNECT:
		c.serverDisconnect(pkt.(*packets.Disconnect))
	case packets.AUTH:

	default:
		go c.occurredError(fmt.Errorf("received unexpected %s for Packet", pkt.Type()))
	}
}

func (c *Client) serverDisconnect(pkt *packets.Disconnect) {
	fmt.Println("server disconnect", pkt.ReasonCode.String())

	c.shutdown()
	if pkt.Properties != nil {
		fmt.Println("props=", *pkt.Properties)
	}
}

func (c *Client) shutdown() {
	c.ctxCancelFunc()
}

func (c *Client) occurredError(err error) {
	fmt.Println(time.Now().Local())

	c.ctxCancelFunc()

	closeErr := c.conn.close()
	fmt.Println("close err=", closeErr)
	fmt.Println("occurredError", err)
}
