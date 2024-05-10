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
	"errors"
	"github.com/lynnplus/go-mqtt/packets"
	"sync/atomic"
)

type Context interface {
	Topic() string
	PacketID() packets.PacketID
	SubscriptionID() int
}

type pubMsgContext struct {
	isCopy atomic.Bool

	client *Client
	packet *packets.Publish
}

func (c *pubMsgContext) SubscriptionID() int {
	if c.packet.Properties == nil || c.packet.Properties.SubscriptionID == nil {
		return 0
	}
	return *c.packet.Properties.SubscriptionID
}

func (c *pubMsgContext) Topic() string {
	return c.packet.Topic
}

func (c *pubMsgContext) PacketID() packets.PacketID {
	return c.packet.ID()
}

func (c *pubMsgContext) RawPacket() *packets.Publish {
	//TODO copy from raw
	return c.packet
}

func (c *pubMsgContext) UnmarshalPayload(v any) error {
	if c.packet.Properties == nil || c.packet.Properties.ContentType == "" {
		return errors.New("mqtt.context: invalid packet")
	}
	return UnmarshalPayload(c.packet.Properties.ContentType, c.packet.Payload, v)
}

func (c *pubMsgContext) Clone() Context {
	//TODO clone raw ctx , set isCopy to true
	return &pubMsgContext{}
}

func (c *pubMsgContext) reset() {

}
