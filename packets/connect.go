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
	Properties      *ConnProperties
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

// ConnProperties is a struct for the Connect properties,reference doc: "[CONNECT Properties]"
//
// [CONNECT Properties]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901046
type ConnProperties struct {
	//the Four Byte Integer representing the Session Expiry Interval in seconds.
	//If the Session Expiry Interval is absent the value 0 is used. If it is set to 0, or is absent, the Session ends when the Network Connection is closed.
	//If the Session Expiry Interval is 0xFFFFFFFF (UINT_MAX), the Session does not expire.
	SessionExpiryInterval uint32
	// the Two Byte Integer representing the Receive Maximum value.
	// The Client uses this value to limit the number of QoS 1 and QoS 2 publications that it is willing to process concurrently. There is no mechanism to limit the QoS 0 publications that the Server might try to send.
	// The value of Receive Maximum applies only to the current Network Connection. If the Receive Maximum value is absent then its value defaults to 65,535.
	ReceiveMaximum      uint16
	MaximumPacketSize   uint32
	TopicAliasMaximum   uint16
	RequestResponseInfo bool
	RequestProblemInfo  bool
	UserProps           UserProperties
	AuthMethod          string
	AuthData            []byte
}

func (c *ConnProperties) Unpack(r io.Reader) error {

	return nil

}

type ConnAckProperties struct {
	SessionExpiryInterval uint32
	AssignedClientID      string
	ServerKeepAlive       uint16
	AuthMethod            string
	AuthData              []byte
	ResponseInfo          string
	ServerReference       string
	ReasonString          string
	ReceiveMaximum        uint16
	TopicAliasMaximum     uint16
	MaximumQoS            uint8
	RetainAvailable       bool
	UserProps             UserProperties
	MaximumPacketSize     uint32
	WildcardSubAvailable  bool
	SubIdAvailable        bool
	SharedSubAvailable    bool
}
