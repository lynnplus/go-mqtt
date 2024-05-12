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
	"time"
)

// ReConnector is an interface that implements the client disconnection and reconnection flow.
//
// After the client successfully establishes a connection,
// if the connection is lost, Connection Lost will first be called to reconnect.
// If the connection fails to be established during the reconstruction process,
// Connection Failure will be called to retry the flow.
type ReConnector interface {
	// ConnectionLost is called when the connection is lost or the server sends a disconnection packet.
	// It returns a timer to indicate when the client should reinitiate the connection.
	// If the timer is empty, it means that no reconnection will be initiated.
	ConnectionLost(pkt *packets.Disconnect, err error) *time.Timer
	// ConnectionFailure is called when the connection fails to be established.
	ConnectionFailure(dialer Dialer, err error) (Dialer, *time.Timer)
	// Reset is called when the connection is successful or initialization is required.
	Reset()
}

type AutoReConnector struct {
	AutoReconnect bool //Whether to automatically reconnect after the connection is lost
	ConnectRetry  bool //Whether to retry after the first connection failure
	MaxRetryDelay time.Duration
	count         atomic.Uint32
	lastLost      time.Time
}

func NewAutoReConnector() *AutoReConnector {
	return &AutoReConnector{
		AutoReconnect: true,
		ConnectRetry:  true,
		MaxRetryDelay: 5 * time.Minute,
	}
}

func (a *AutoReConnector) Reset() {
	a.count.Store(0)
}

func (a *AutoReConnector) ConnectionLost(pkt *packets.Disconnect, err error) *time.Timer {
	if !a.AutoReconnect {
		return nil
	}
	if pkt != nil {
		if pkt.ReasonCode == packets.ServerBusy || pkt.ReasonCode == packets.ServerShuttingDown {
			return time.NewTimer(15 * time.Second)
		}
	}

	now := time.Now()
	if now.Sub(a.lastLost) < (5 * time.Second) {
		return time.NewTimer(10 * time.Second)
	} else {
		a.lastLost = now
		return time.NewTimer(2 * time.Second)
	}
}

func (a *AutoReConnector) allowRetry(code packets.ReasonCode) bool {
	switch code {
	case packets.ServerBusy,
		packets.ServerUnavailable:
		return true
	}
	return false
}

func (a *AutoReConnector) ConnectionFailure(dialer Dialer, err error) (Dialer, *time.Timer) {
	if !a.ConnectRetry {
		return dialer, nil
	}
	var reasonErr *packets.ReasonCodeError
	if errors.As(err, &reasonErr) {
		if !a.allowRetry(reasonErr.Code) {
			return dialer, nil
		}
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
