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
	"fmt"
	"io"
)

type ProtocolVersion byte

const (
	ProtocolVersion31  ProtocolVersion = 3
	ProtocolVersion311 ProtocolVersion = 4 // mqtt v3.1.1 protocol version
	ProtocolVersion5   ProtocolVersion = 5
)

type ProtocolName string

const (
	ProtocolMQIsdp ProtocolName = "MQIsdp"
	ProtocolMQTT   ProtocolName = "MQTT"
)

// Connect is
type Connect struct {
	ProtocolName    ProtocolName
	ProtocolVersion ProtocolVersion
	KeepAlive       uint16 //in second
	Properties      *ConnProperties
	ClientID        string
	CleanStart      bool
	Username        string
	Password        []byte
	WillMessage     *WillMessage
}

type WillMessage struct {
	Topic      string
	Payload    []byte
	Retain     bool
	Qos        byte
	Properties *WillProperties
}

func (c *Connect) Pack(w io.Writer) error {
	var err error
	if err = unsafeWriteString(w, (*string)(&c.ProtocolName)); err != nil {
		return err
	}
	if err = unsafeWriteByte(w, (*byte)(&c.ProtocolVersion)); err != nil {
		return err
	}
	//write flags
	if err = c.packFlags(w); err != nil {
		return err
	}
	if err = unsafeWriteUint16(w, &c.KeepAlive); err != nil {
		return err
	}
	if c.Properties != nil {
		if c.ProtocolVersion < ProtocolVersion5 {
			return NewReasonCodeError(ProtocolError, "Protocol version 3 does not support packet properties")
		}
		if err = c.Properties.Pack(w); err != nil {
			return err
		}
	}
	if err = unsafeWriteString(w, &c.ClientID); err != nil {
		return err
	}
	if c.hasWillMsg() {
		will := c.WillMessage
		if err = will.Properties.Pack(w); err != nil {
			return err
		}
		if err := unsafeWriteString(w, &will.Topic); err != nil {
			return err
		}
		if err := unsafeWriteBytes(w, &will.Payload); err != nil {
			return err
		}
	}
	return nil
}

func (c *Connect) hasWillMsg() bool {
	return c.WillMessage != nil && c.WillMessage.Topic != ""
}

func (c *Connect) packFlags(w io.Writer) error {
	var flags byte

	if c.Username != "" {
		flags |= 0x01 << 7
	}
	if len(c.Password) > 0 {
		flags |= 0x01 << 6
	}
	if c.hasWillMsg() {
		flags |= 0x01 << 2
		will := c.WillMessage
		flags |= will.Qos << 3
		if will.Retain {
			flags |= 0x01 << 5
		}
	}

	if c.CleanStart {
		flags |= 0x01 << 1
	}
	return unsafeWriteByte(w, &flags)
}

// Unpack read the packet bytes from io.Reader and decodes it into the packet struct.
func (c *Connect) Unpack(r io.Reader) error {
	var err error
	if err = unsafeReadString(r, (*string)(&c.ProtocolName)); err != nil {
		return err
	}
	if err = unsafeReadByte(r, (*byte)(&c.ProtocolVersion)); err != nil {
		return err
	}

	if c.ProtocolVersion >= ProtocolVersion311 && c.ProtocolName != ProtocolMQTT {
		return NewReasonCodeError(UnsupportedProtocolVersion, fmt.Sprintf("protocol version %d mismatch name %s", c.ProtocolVersion, ProtocolMQTT))
	}
	if c.ProtocolVersion == ProtocolVersion31 || c.ProtocolVersion > ProtocolVersion5 {
		return NewReasonCodeError(UnsupportedProtocolVersion, "")
	}

	flags := byte(0)
	c.CleanStart = 1&(flags>>1) > 0

	if err = unsafeReadByte(r, &flags); err != nil {
		return err
	}
	if err = unsafeReadUint16(r, &c.KeepAlive); err != nil {
		return err
	}
	//Properties x bytes
	if c.ProtocolVersion >= ProtocolVersion5 {
		//read props
		props := &ConnProperties{}
		if err = props.Unpack(r); err != nil {
			return err
		}
		c.Properties = props
	}
	//read payload
	if err = unsafeReadString(r, &c.ClientID); err != nil {
		return err
	}

	//will flag set
	if 1&(flags>>2) > 0 {
		will := &WillMessage{}
		will.Retain = 1&(flags>>5) > 0
		will.Qos = 3 & (flags >> 3)
		if c.ProtocolVersion >= ProtocolVersion5 {
			//read props
			props := &WillProperties{}
			if err = props.Unpack(r); err != nil {
				return err
			}
			will.Properties = props
		}
		if err = unsafeReadString(r, &will.Topic); err != nil {
			return err
		}
		if err = unsafeReadBytes(r, &will.Payload); err != nil {
			return err
		}
		c.WillMessage = will
	}

	if 1&(flags>>7) > 0 {
		if err := unsafeReadString(r, &c.Username); err != nil {
			return err
		}
	}
	if 1&(flags>>6) > 0 {
		if err := unsafeReadBytes(r, &c.Password); err != nil {
			return err
		}
	}
	return nil
}

// ConnProperties is a struct for the Connect properties,reference doc: "[CONNECT Properties]"
//
// [CONNECT Properties]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901046
type ConnProperties struct {
	//representing the Session Expiry Interval in seconds
	//If the Session Expiry Interval is absent the value 0 is used. If it is set to 0, or is absent, the Session ends when the Network Connection is closed.
	//If the Session Expiry Interval is 0xFFFFFFFF (UINT_MAX), the Session does not expire.
	SessionExpiryInterval uint32
	// representing the Receive Maximum value.
	// The Client uses this value to limit the number of QoS 1 and QoS 2 publications that it is willing to process concurrently. There is no mechanism to limit the QoS 0 publications that the Server might try to send.
	// The value of Receive Maximum applies only to the current Network Connection. If the Receive Maximum value is absent then its value defaults to 65,535.
	ReceiveMaximum uint16
	//representing the Maximum Packet Size the Client is willing to accept
	MaximumPacketSize *uint32
	//Indicates the maximum number of acceptable topic aliases.
	//Defaults to 0 when the property is not present.
	//A value of 0 indicates that no topic aliases will be accepted on this connection.
	TopicAliasMaximum uint16
	// Indicates whether the client requests the server to include response information
	// in the CONNACK message. When the value is true, the server may include ResponseInfo in CONNACK.
	// When the property value does not exist, the default is false, indicating that no response information is included.
	RequestResponseInfo bool
	// Indicates whether to send ReasonString or UserProperties when the request fails.
	// When the value is empty, the server defaults to true.
	RequestProblemInfo *bool
	// User-defined properties, which is a string key-value pair
	UserProps UserProperties
	// Identifier used for the authentication method
	AuthMethod string
	AuthData   []byte
}

func (c *ConnProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, CONNECT)
	if err != nil {
		return err
	}
	for id, ptr := range ps {
		if err != nil {
			return err
		}
		switch id {
		case PropSessionExpiryInterval:
			safeCopyPropValue(ptr, &c.SessionExpiryInterval, &err)
		case PropReceiveMaximum:
			safeCopyPropValue(ptr, &c.ReceiveMaximum, &err)
		case PropMaximumPacketSize:
			safeCopyPropPtr(ptr, &c.MaximumPacketSize, &err)
		case PropTopicAliasMaximum:
			safeCopyPropValue(ptr, &c.TopicAliasMaximum, &err)
		case PropRequestResponseInfo:
			safeCopyPropValue(ptr, &c.RequestResponseInfo, &err)
		case PropRequestProblemInfo:
			safeCopyPropPtr(ptr, &c.RequestProblemInfo, &err)
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
	return nil
}

func (c *ConnProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error

	if c.SessionExpiryInterval > 0 {
		writePropIdAndValue(buf, PropSessionExpiryInterval, &c.SessionExpiryInterval, &err)
	}
	if c.ReceiveMaximum > 0 && c.ReceiveMaximum < 65535 {
		writePropIdAndValue(buf, PropReceiveMaximum, &c.ReceiveMaximum, &err)
	}

	writePropIdAndValue(buf, PropMaximumPacketSize, c.MaximumPacketSize, &err)

	if c.TopicAliasMaximum > 0 {
		writePropIdAndValue(buf, PropTopicAliasMaximum, &c.TopicAliasMaximum, &err)
	}
	if c.RequestResponseInfo {
		writePropIdAndValue(buf, PropRequestResponseInfo, &c.RequestResponseInfo, &err)
	}
	writePropIdAndValue(buf, PropRequestProblemInfo, c.RequestProblemInfo, &err)

	if c.AuthMethod != "" {
		writePropIdAndValue(buf, PropAuthMethod, &c.AuthMethod, &err)
		writePropIdAndValue(buf, PropAuthData, &c.AuthData, &err)
	}
	writeUserPropsData(w, c.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}

type WillProperties struct {
	// Will delay interval (in seconds), defaults to 0 when the property does not exist
	WillDelayInterval uint32
	//Indicates whether the payload is UTF-8 encoded character data
	PayloadFormatIndicator bool
	MessageExpiryInterval  *uint32
	ContentType            string
	ResponseTopic          string
	CorrelationData        []byte
	UserProps              UserProperties
}

func (w *WillProperties) Unpack(r io.Reader) error {
	ps, err := ReadWillProperties(r)
	if err != nil {
		return err
	}
	for id, ptr := range ps {
		if err != nil {
			return err
		}
		switch id {
		case PropWillDelayInterval:
			safeCopyPropValue(ptr, &w.WillDelayInterval, &err)
		case PropPayloadFormat:
			safeCopyPropValue(ptr, &w.PayloadFormatIndicator, &err)
		case PropMessageExpiryInterval:
			safeCopyPropPtr(ptr, &w.MessageExpiryInterval, &err)
		case PropContentType:
			safeCopyPropValue(ptr, &w.ContentType, &err)
		case PropResponseTopic:
			safeCopyPropValue(ptr, &w.ResponseTopic, &err)
		case PropCorrelationData:
			safeCopyPropValue(ptr, &w.CorrelationData, &err)
		}
	}
	up, ok := ps[PropUserProperty]
	if ok {
		w.UserProps = up.(UserProperties)
	}
	return nil
}

func (w *WillProperties) Pack(wr io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error

	if w.WillDelayInterval > 0 {
		writePropIdAndValue(buf, PropWillDelayInterval, &w.WillDelayInterval, &err)
	}
	if w.PayloadFormatIndicator {
		writePropIdAndValue(buf, PropPayloadFormat, &w.PayloadFormatIndicator, &err)
	}
	writePropIdAndValue(buf, PropMessageExpiryInterval, w.MessageExpiryInterval, &err)

	if w.ContentType != "" {
		writePropIdAndValue(buf, PropContentType, &w.ContentType, &err)
	}
	if w.ResponseTopic != "" {
		writePropIdAndValue(buf, PropResponseTopic, &w.ResponseTopic, &err)
	}
	if len(w.CorrelationData) > 0 {
		writePropIdAndValue(buf, PropCorrelationData, &w.CorrelationData, &err)
	}
	writeUserPropsData(wr, w.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(wr, buf.Bytes())
}
