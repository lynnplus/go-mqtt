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
	"github.com/lynnplus/go-mqtt/packets"
	"sync/atomic"
	"time"
)

type ReConnector interface {
	ConnectionLost(pkt *packets.Disconnect, err error) *time.Timer
	ConnectionFailure(dialer Dialer, err error) *time.Timer
	// Reset is called when the connection is successful or initialization is required.
	Reset()
}

type AutoReConnector struct {
	AutoReconnect bool //Whether to retry after connection loss
	ConnectRetry  bool //Whether to retry after connection failure
	count         atomic.Uint32
}

func NewAutoReConnector() *AutoReConnector {
	return &AutoReConnector{
		AutoReconnect: true,
		ConnectRetry:  true,
	}
}

func (a *AutoReConnector) Reset() {
	a.count.Store(0)
}

func (a *AutoReConnector) ConnectionLost(pkt *packets.Disconnect, err error) *time.Timer {
	if !a.AutoReconnect {
		return nil
	}
	return nil
}

func (a *AutoReConnector) ConnectionFailure(dialer Dialer, err error) *time.Timer {
	if !a.ConnectRetry {
		return nil
	}
	return nil
}
