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
	ConnectionFailure(dialer Dialer, err error) (Dialer, *time.Timer)
	// Reset is called when the connection is successful or initialization is required.
	Reset()
}

type AutoReConnector struct {
	AutoReconnect bool //Whether to automatically reconnect after the connection is lost
	ConnectRetry  bool //Whether to retry after the first connection failure
	MaxRetryDelay time.Duration
	count         atomic.Uint32
}

func NewAutoReConnector() *AutoReConnector {
	return &AutoReConnector{
		AutoReconnect: true,
		ConnectRetry:  true,
		MaxRetryDelay: time.Minute,
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

func (a *AutoReConnector) ConnectionFailure(dialer Dialer, err error) (Dialer, *time.Timer) {
	if !a.ConnectRetry {
		return dialer, nil
	}
	d := a.count.Load() * 2
	if d < 5 {
		d = 5
	}
	delay := time.Duration(d) * time.Second
	if delay > a.MaxRetryDelay {
		delay = a.MaxRetryDelay
	}
	timer := time.NewTimer(delay)
	a.count.Add(1)
	return dialer, timer
}
