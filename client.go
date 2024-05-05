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

import "context"

type ClientListener struct {
	OnServerConnected    func(client Client, reconn bool)
	OnServerDisconnected func(client Client, err error)
	OnServerConnFailed   func(client Client, err error)
}

type ClientConfig struct {
	AutoConnect bool //Whether to automatically retry after failure to connect to the server
	Pinger      Pinger
	ClientListener
}

type Client struct {
}

func NewClient(config ClientConfig) *Client {
	return &Client{}
}

func (c *Client) Connect(ctx context.Context) error {

	return nil
}