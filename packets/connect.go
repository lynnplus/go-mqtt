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

package packets

import (
	"io"
)

type ProtocolVersion byte

const (
	ProtocolVersion31  ProtocolVersion = 3
	ProtocolVersion311 ProtocolVersion = 4 // mqtt v3.1.1 protocol version
	ProtocolVersion5   ProtocolVersion = 5
)

type Connect struct {
	ProtocolName    string
	ProtocolVersion ProtocolVersion
	KeepAlive       uint16
}

// Unpack read the packet bytes from io.Reader and decodes it into the packet struct.
func (c *Connect) Unpack(r io.Reader) error {

	var err error

	if err = unsafeReadString(r, &c.ProtocolName); err != nil {
		return err
	}
	if err = unsafeReadByte(r, (*byte)(&c.ProtocolVersion)); err != nil {
		return err
	}

	flags := byte(0)
	if err = unsafeReadByte(r, &flags); err != nil {
		return err
	}

	if err = unsafeReadUint16(r, &c.KeepAlive); err != nil {
		return err
	}
	//Properties x bytes

	if c.ProtocolVersion >= ProtocolVersion5 {
		//read props
	}

	//read payload
	return nil
}
