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
	"net"
	"sync/atomic"
)

type Context interface {
}

type mqttContext struct {
	conn   net.Conn
	isCopy atomic.Bool
}

func (c *mqttContext) reset() {

}

func (c *mqttContext) MarshalPlay(v any) {

}

func (c *mqttContext) Clone() Context {
	//TODO clone raw ctx , set isCopy to true
	return &mqttContext{}
}
