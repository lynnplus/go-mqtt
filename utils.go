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

import "sync/atomic"

type ConnState struct {
	_ noCopy
	v ConnStatus
}

func (x *ConnState) String() string {
	return x.Load().String()
}

type noCopy struct{}

type ConnStatus int32

const (
	StatusNone ConnStatus = iota
	StatusConnecting
	StatusConnected
	StatusDisconnected
)

func (c ConnStatus) String() string {
	switch c {
	case StatusNone:
		return "StatusNone"
	case StatusConnecting:
		return "StatusConnecting"
	case StatusConnected:
		return "StatusConnected"
	case StatusDisconnected:
		return "StatusDisconnected"
	}
	return "Unknown"
}

func (x *ConnState) Load() ConnStatus { return ConnStatus(atomic.LoadInt32((*int32)(&x.v))) }

func (x *ConnState) Store(val ConnStatus) { atomic.StoreInt32((*int32)(&x.v), int32(val)) }

func (x *ConnState) CompareAndSwap(old, new ConnStatus) (swapped bool) {
	return atomic.CompareAndSwapInt32((*int32)(&x.v), int32(old), int32(new))
}

func (x *ConnState) IsConnected() bool {
	return x.Load() == StatusConnected
}
