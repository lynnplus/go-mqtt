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
	"time"
)

type Pinger interface {
	Run(ctx context.Context, keepAlive time.Duration, c *Client) error
	// Ping is called when a packet is sent to the server.
	// If the packet is packets.PINGREQ, it will not be called
	Ping()
	// Pong is called when a packets.PINGRESP packet is received from the server
	Pong()
}

type DefaultPinger struct {
	pingChan chan time.Time
	pongChan chan time.Time
}

func NewDefaultPinger() *DefaultPinger {
	return &DefaultPinger{
		pingChan: make(chan time.Time, 2),
		pongChan: make(chan time.Time, 2),
	}
}

var ErrPongTimeout = errors.New("pong timeout")

func (p *DefaultPinger) Run(ctx context.Context, keepAlive time.Duration, c *Client) error {
	if keepAlive == 0 {
		return nil
	}
	//create a timer that fires immediately
	timer := time.NewTimer(0)
	defer timer.Stop()
	var lastPing, lastPktSend, lastPong time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-p.pingChan:
			lastPktSend = t
		case t := <-p.pongChan:
			lastPong = t
		case t := <-timer.C:
			if !lastPing.IsZero() && lastPing.After(lastPong) {
				return ErrPongTimeout
			}
			due := lastPktSend.Add(keepAlive)
			if t.Before(due) {
				timer.Reset(due.Sub(t))
				continue
			}
			lastPing = time.Now()
			if err := c.SendPing(ctx); err != nil {
				return err
			}
			timer.Reset(keepAlive)
		}
	}
}

func (p *DefaultPinger) Ping() {
	if len(p.pingChan) > 0 {
		return
	}
	p.pingChan <- time.Now()
}

func (p *DefaultPinger) Pong() {
	if len(p.pongChan) > 0 {
		return
	}
	p.pongChan <- time.Now()
}
