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
	"bufio"
	"context"
	"fmt"
	"github.com/lynnplus/go-mqtt/packets"
	"net"
	"sync"
	"sync/atomic"
)

type packetInfo struct {
	packet packets.Packet
	err    chan error
}

type Conn struct {
	conn    net.Conn
	version packets.ProtocolVersion
	reader  *bufio.Reader
	writer  *bufio.Writer
	wg      sync.WaitGroup
	client  *Client
	closed  atomic.Bool
}

func attemptConnection(ctx context.Context, dialer Dialer, size int, client *Client) (*Conn, error) {
	conn, err := dialer.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn:    conn,
		client:  client,
		version: client.version,
		reader:  bufio.NewReaderSize(conn, size),
		writer:  bufio.NewWriterSize(conn, size),
	}, nil
}

func (conn *Conn) readPacket() (packets.Packet, error) {
	return packets.ReadFrom(conn.reader, conn.version)
}

func (conn *Conn) writePacket(packet packets.Packet) error {
	return packets.WriteTo(conn.writer, packet, conn.version)
}

func (conn *Conn) flushWrite() error {
	return conn.writer.Flush()
}

func (conn *Conn) run(ctx context.Context, output <-chan *packetInfo) {
	conn.wg.Add(2)
	go conn.loopRead(ctx)
	go conn.loopWrite(ctx, output)
}

func (conn *Conn) loopRead(ctx context.Context) {
	defer conn.wg.Done()
	for {
		select {
		case <-ctx.Done():
			conn.loopEnd(nil)
			return
		default:
		}
		pkt, err := conn.readPacket()
		if err != nil {
			conn.loopEnd(err)
			return
		}
		conn.client.incoming(pkt)
	}
}

func (conn *Conn) loopWrite(ctx context.Context, output <-chan *packetInfo) {
	defer conn.wg.Done()
	var err error
	for {
		select {
		case <-ctx.Done():
			conn.loopEnd(nil)
			return
		case info, ok := <-output:
			if !ok {
				conn.loopEnd(nil)
				return
			}
			err = conn.writePacket(info.packet)
			if err != nil {
				info.err <- err
				conn.loopEnd(err)
				return
			}
			if len(output) == 0 {
				if err = conn.writer.Flush(); err != nil {
					info.err <- err
					conn.loopEnd(err)
					return
				}
			}
			info.err <- nil
		}
	}
}

func (conn *Conn) loopEnd(err error) {
	fmt.Println(err)
	if conn.closed.Load() {
		return
	}
	if conn.client != nil {
		go conn.client.occurredError(err)
	}
}

func (conn *Conn) close() error {
	if !conn.closed.CompareAndSwap(false, true) {
		return nil
	}
	err := conn.conn.Close()
	conn.wg.Wait()

	conn.client = nil
	conn.conn = nil
	return err
}
