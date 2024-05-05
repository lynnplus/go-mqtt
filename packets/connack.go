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

import "io"

type Connack struct {
	ReasonCode     byte
	SessionPresent bool //Used to indicate whether the server is using an existing session to resume communication with the client. Session Present may be 1 only if the client sets Clean Start to 0 in the CONNECT connection
	Properties     *ConnackProperties
}

func (c *Connack) Unpack(r io.Reader) error {
	flags := byte(0)
	var err error
	if err = unsafeReadByte(r, &flags); err != nil {
		return err
	}
	c.SessionPresent = flags&0x01 > 0

	if err := unsafeReadByte(r, &c.ReasonCode); err != nil {
		return err
	}
	return c.Properties.Unpack(r)
}

type ConnackProperties struct {
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
	MaximumQoS            *byte
	RetainAvailable       *bool
	UserProps             UserProperties
	MaximumPacketSize     uint32
	WildcardSubAvailable  *bool
	SubIdAvailable        bool
	SharedSubAvailable    bool
}

func (c *ConnackProperties) Pack(w io.Writer) error {
	if c.MaximumQoS != nil {
		if *c.MaximumQoS != 0 && *c.MaximumQoS != 1 {
			return newInvalidPropValueError(PropMaximumQOS, *c.MaximumQoS)
		}
	}
	return nil
}

func (c *ConnackProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, CONNACK)
	if err != nil {
		return err
	}
	CopyPropPtrValue(ps, PropSessionExpiryInterval, &c.SessionExpiryInterval, 0)
	CopyPropPtrValue(ps, PropReceiveMaximum, &c.ReceiveMaximum, 65535)
	CopyPropPtrValue(ps, PropMaximumQOS, c.MaximumQoS, 2)
	CopyPropPtrValue(ps, PropRetainAvailable, c.RetainAvailable, true)
	CopyPropPtrValue(ps, PropMaximumPacketSize, &c.MaximumPacketSize, uint32(0))
	CopyPropPtrValue(ps, PropAssignedClientID, &c.AssignedClientID, "")
	CopyPropPtrValue(ps, PropTopicAliasMaximum, &c.TopicAliasMaximum, uint16(0))
	CopyPropPtrValue(ps, PropReasonString, &c.ReasonString, "")
	CopyPropPtrValue(ps, PropWildcardSubAvailable, c.WildcardSubAvailable, false)

	//load

	CopyPropPtrValue(ps, PropAuthMethod, &c.AuthMethod, "")
	CopyPropPtrValue(ps, PropAuthData, &c.AuthData, nil)
	if c.AuthMethod == "" && c.AuthData != nil {
		return NewReasonCodeError(ProtocolError, "AuthData accidentally included when AuthMethod is empty")
	}
	up, ok := ps[PropUserProperty]
	if ok {
		c.UserProps = up.(UserProperties)
	}
	return nil
}
