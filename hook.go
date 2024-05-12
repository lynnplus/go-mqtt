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

import "github.com/lynnplus/go-mqtt/packets"

type HookName string

const (
	HookConnect     HookName = "connect"
	HookPublish     HookName = "publish"
	HookSubscribe   HookName = "subscribe"
	HookUnsubscribe HookName = "unsubscribe"
	HookDisconnect  HookName = "disconnect"
)

type Trigger struct{}

func (g *Trigger) onConnectionCompleted(client *Client, ack *packets.Connack) {
	if client.config.OnConnected != nil {
		go client.config.OnConnected(client, ack)
	}
}

func (g *Trigger) onConnectionLost(client *Client, err error) {
	if client.config.OnConnectionLost != nil {
		go client.config.OnConnectionLost(client, err)
	}
}

func (g *Trigger) onConnectFailed(client *Client, err error) {
	if client.config.OnConnectFailed != nil {
		go client.config.OnConnectFailed(client, err)
	}
}

func (g *Trigger) onServerDisconnect(client *Client, pkt *packets.Disconnect) {
	if client.config.OnServerDisconnect != nil {
		go client.config.OnServerDisconnect(client, pkt)
	}
}

func (g *Trigger) onClientError(client *Client, err error) {
	if client.config.OnClientError != nil {
		go client.config.OnClientError(client, err)
	}
}
