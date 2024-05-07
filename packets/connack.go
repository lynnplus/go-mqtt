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
	"bytes"
	"io"
)

type Connack struct {
	ReasonCode     ReasonCode
	SessionPresent bool //Used to indicate whether the server is using an existing session to resume communication with the client. Session Present may be 1 only if the client sets Clean Start to 0 in the CONNECT connection
	Properties     *ConnackProperties
}

func (c *Connack) Pack(w io.Writer, header *FixedHeader) error {
	var err error
	b := [2]byte{0, byte(c.ReasonCode)}
	if c.SessionPresent {
		b[0] = 1
	}
	if _, err = w.Write(b[:]); err != nil {
		return err
	}
	if err = packPacketProperties(w, c.Properties, header.version); err != nil {
		return err
	}
	return err
}

func (c *Connack) Unpack(r io.Reader, header *FixedHeader) error {
	flags := byte(0)
	var err error
	if err = unsafeReadByte(r, &flags); err != nil {
		return err
	}
	c.SessionPresent = flags&0x01 > 0
	if err := unsafeReadByte(r, (*byte)(&c.ReasonCode)); err != nil {
		return err
	}
	if header.version >= ProtocolVersion5 {
		//read props
		props := &ConnackProperties{}
		if err = props.Unpack(r); err != nil {
			return err
		}
		c.Properties = props
	}
	return nil
}

func (c *Connack) Type() PacketType {
	return CONNACK
}

func (c *Connack) ID() PacketID {
	return 0
}

type ConnackProperties struct {
	//representing the Session Expiry Interval in seconds
	//If the Session Expiry Interval is absent the value 0 is used. If it is set to 0, or is absent, the Session ends when the Network Connection is closed.
	//If the Session Expiry Interval is 0xFFFFFFFF (UINT_MAX), the Session does not expire.
	SessionExpiryInterval uint32
	ReceiveMaximum        uint16
	MaximumQoS            *byte
	// Indicates whether the server supports retained messages.
	// If the value is not present, it means retained messages are supported.
	RetainAvailable *bool
	// Indicates the maximum packet size accepted by the server.
	// If the value does not exist, there is no limit on the size.
	MaximumPacketSize *uint32
	TopicAliasMaximum uint16
	ReasonString      string
	AssignedClientID  string
	//Indicates whether the server supports wildcard subscriptions.
	//A value of false indicates that it is not supported.
	//A value of true or does not exist indicates that it is supported.
	WildcardSubAvailable *bool

	//Represents the server-assigned keep-alive time. If the value is present,
	// the client must use this value as the KeepAlive time instead of the corresponding
	// value sent in the CONNECT packet.(in second)
	ServerKeepAlive *uint16
	AuthMethod      string
	AuthData        []byte
	ResponseInfo    string
	ServerReference string
	UserProps       UserProperties
	//Indicates whether the server supports subscription identifiers.
	// If the value is true or does not exist,
	// it means it is supported, otherwise it is not supported.
	SubIdAvailable     *bool
	SharedSubAvailable *bool
}

func (c *ConnackProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error

	if c.MaximumQoS != nil && *c.MaximumQoS > 1 {
		return newInvalidPropValueError(PropMaximumQOS, *c.MaximumQoS)
	}

	if c.SessionExpiryInterval > 0 {
		writePropIdAndValue(buf, PropSessionExpiryInterval, &c.SessionExpiryInterval, &err)
	}
	if c.ReceiveMaximum > 0 && c.ReceiveMaximum < 65535 {
		writePropIdAndValue(buf, PropReceiveMaximum, &c.ReceiveMaximum, &err)
	}
	writePropIdAndValue(buf, PropMaximumQOS, c.MaximumQoS, &err)
	writePropIdAndValue(buf, PropRetainAvailable, c.RetainAvailable, &err)
	writePropIdAndValue(buf, PropMaximumPacketSize, c.MaximumPacketSize, &err)

	if c.AssignedClientID != "" {
		writePropIdAndValue(buf, PropAssignedClientID, &c.AssignedClientID, &err)
	}
	if c.TopicAliasMaximum > 0 {
		writePropIdAndValue(buf, PropTopicAliasMaximum, &c.TopicAliasMaximum, &err)
	}
	if c.ReasonString != "" {
		writePropIdAndValue(buf, PropReasonString, &c.ReasonString, &err)
	}
	writePropIdAndValue(buf, PropWildcardSubAvailable, c.WildcardSubAvailable, &err)
	writePropIdAndValue(buf, PropSubIDAvailable, c.SubIdAvailable, &err)
	writePropIdAndValue(buf, PropSharedSubAvailable, c.SharedSubAvailable, &err)
	writePropIdAndValue(buf, PropServerKeepAlive, c.ServerKeepAlive, &err)

	if c.ResponseInfo != "" {
		writePropIdAndValue(buf, PropResponseInfo, &c.ResponseInfo, &err)
	}
	if c.ServerReference != "" {
		writePropIdAndValue(buf, PropServerReference, &c.ServerReference, &err)
	}
	if c.AuthMethod != "" {
		writePropIdAndValue(buf, PropAuthMethod, &c.AuthMethod, &err)
		writePropIdAndValue(buf, PropAuthData, &c.AuthData, &err)
	}
	writeUserPropsData(buf, c.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}

func (c *ConnackProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, CONNACK)
	if err != nil {
		return err
	}
	c.ReceiveMaximum = 65535

	for id, ptr := range ps {
		if err != nil {
			return err
		}
		switch id {
		case PropSessionExpiryInterval:
			safeCopyPropValue(ptr, &c.SessionExpiryInterval, &err)
		case PropReceiveMaximum:
			safeCopyPropValue(ptr, &c.ReceiveMaximum, &err)
		case PropMaximumQOS:
			safeCopyPropPtr(ptr, &c.MaximumQoS, &err)
		case PropRetainAvailable:
			safeCopyPropPtr(ptr, &c.RetainAvailable, &err)
		case PropMaximumPacketSize:
			safeCopyPropPtr(ptr, &c.MaximumPacketSize, &err)
		case PropAssignedClientID:
			safeCopyPropValue(ptr, &c.AssignedClientID, &err)
		case PropTopicAliasMaximum:
			safeCopyPropValue(ptr, &c.TopicAliasMaximum, &err)
		case PropReasonString:
			safeCopyPropValue(ptr, &c.ReasonString, &err)
		case PropWildcardSubAvailable:
			safeCopyPropPtr(ptr, &c.WildcardSubAvailable, &err)
		case PropServerKeepAlive:
			safeCopyPropPtr(ptr, &c.ServerKeepAlive, &err)
		case PropResponseInfo:
			safeCopyPropValue(ptr, &c.ResponseInfo, &err)
		case PropServerReference:
			safeCopyPropValue(ptr, &c.ServerReference, &err)
		case PropSubIDAvailable:
			safeCopyPropPtr(ptr, &c.SubIdAvailable, &err)
		case PropSharedSubAvailable:
			safeCopyPropPtr(ptr, &c.SharedSubAvailable, &err)
		case PropAuthMethod:
			safeCopyPropValue(ptr, &c.AuthMethod, &err)
		case PropAuthData:
			safeCopyPropValue(ptr, &c.AuthData, &err)
		}
	}
	up, ok := ps[PropUserProperty]
	if ok {
		c.UserProps = up.(UserProperties)
	}
	return err
}
