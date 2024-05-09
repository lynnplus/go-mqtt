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

type Publish struct {
	Topic string
	QoS   byte
	//Indicates whether it is a re-delivery of a previous attempt to send the packet. If the value is false,
	//it indicates that this is the first time the client or server has tried to send this packet.
	Duplicate bool
	Retain    bool
	//Packet identifier, only exists in data packets with Qos level 1 or 2
	PacketID   PacketID
	Payload    []byte
	Properties *PubProperties
}

func NewPublish(topic string, payload []byte) *Publish {
	return NewPublishWith(topic, payload, 0, false)
}

func NewRetainPublish(topic string, payload []byte) *Publish {
	return NewPublishWith(topic, payload, 0, true)
}

func NewPublishWith(topic string, payload []byte, qos byte, retain bool) *Publish {
	return &Publish{
		Topic:   topic,
		QoS:     qos,
		Retain:  retain,
		Payload: payload,
	}
}

func (p *Publish) Pack(w io.Writer, header *FixedHeader) error {
	f := p.QoS << 1
	if p.Duplicate {
		f |= 1 << 3
	}
	if p.Retain {
		f |= 1
	}
	header.Flags = f
	var err error
	if err = unsafeWriteString(w, &p.Topic); err != nil {
		return err
	}
	if p.QoS > 0 {
		if err = unsafeWriteUint16(w, &p.PacketID); err != nil {
			return err
		}
	}
	if err = p.packProperties(w, header.version); err != nil {
		return err
	}
	if len(p.Payload) <= 0 {
		return nil
	}
	return unsafeWriteBytes(w, &p.Payload)
}

func (p *Publish) Unpack(r io.Reader, header *FixedHeader) error {
	p.QoS = (header.Flags >> 1) & 0x3
	p.Duplicate = header.Flags&(1<<3) != 0
	p.Retain = header.Flags&1 != 0

	rr := &io.LimitedReader{R: r, N: int64(header.RemainLength)}
	var err error
	if err = unsafeReadString(rr, &p.Topic); err != nil {
		return err
	}
	if p.QoS > 0 {
		if err = unsafeReadUint16(rr, &p.PacketID); err != nil {
			return err
		}
	}
	if header.version >= ProtocolVersion5 {
		props := &PubProperties{}
		if err = props.Unpack(rr); err != nil {
			return err
		}
		p.Properties = props
	}
	if rr.N == 0 {
		return nil
	}
	p.Payload = make([]byte, rr.N)
	_, err = rr.Read(p.Payload)
	return err
}

func (p *Publish) packProperties(w io.Writer, version ProtocolVersion) error {
	var prop Packable
	if p.Properties != nil {
		prop = p.Properties
	}
	return packPacketProperties(w, prop, version)
}

func (p *Publish) Type() PacketType {
	return PUBLISH
}

func (p *Publish) ID() PacketID {
	return p.PacketID
}

func (p *Publish) SetID(id PacketID) {
	p.PacketID = id
}

type PubProperties struct {
	//Indicates whether the payload is UTF-8 encoded character data
	PayloadFormatIndicator bool
	MessageExpiryInterval  *uint32
	TopicAlias             *uint16
	ResponseTopic          string
	CorrelationData        []byte
	ContentType            string
	//Represents the subscription identifier. The value can be 1 to 268,435,455.
	// If the value is 0, it is a protocol error.
	SubscriptionID *int
	User           UserProperties
}

func NewPubProperties() *PubProperties {
	return &PubProperties{User: map[string][]string{}}
}

func (p *PubProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error
	if p.TopicAlias != nil && *p.TopicAlias == 0 {
		return newInvalidPropValueError(PropTopicAlias, 0)
	}
	if p.SubscriptionID != nil && *p.SubscriptionID == 0 {
		return newInvalidPropValueError(PropSubscriptionID, 0)
	}

	if p.PayloadFormatIndicator {
		writePropIdAndValue(buf, PropPayloadFormat, &p.PayloadFormatIndicator, &err)
	}
	writePropIdAndValue(buf, PropMessageExpiryInterval, p.MessageExpiryInterval, &err)
	writePropIdAndValue(buf, PropTopicAlias, p.TopicAlias, &err)
	if p.ResponseTopic != "" {
		writePropIdAndValue(buf, PropResponseTopic, &p.ResponseTopic, &err)
	}
	if len(p.CorrelationData) > 0 {
		writePropIdAndValue(buf, PropCorrelationData, &p.CorrelationData, &err)
	}
	if p.ContentType != "" {
		writePropIdAndValue(buf, PropContentType, &p.ContentType, &err)
	}
	writePropIdAndValue(buf, PropSubscriptionID, p.SubscriptionID, &err)
	writeUserPropsData(buf, p.User, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}

func (p *PubProperties) Unpack(r io.Reader) error {
	ps, err := ReadWillProperties(r)
	if err != nil {
		return err
	}
	for id, ptr := range ps {
		if err != nil {
			return err
		}
		switch id {
		case PropPayloadFormat:
			safeCopyPropValue(ptr, &p.PayloadFormatIndicator, &err)
		case PropMessageExpiryInterval:
			safeCopyPropPtr(ptr, &p.MessageExpiryInterval, &err)
		case PropTopicAlias:
			safeCopyPropPtr(ptr, &p.TopicAlias, &err)
		case PropContentType:
			safeCopyPropValue(ptr, &p.ContentType, &err)
		case PropResponseTopic:
			safeCopyPropValue(ptr, &p.ResponseTopic, &err)
		case PropCorrelationData:
			safeCopyPropValue(ptr, &p.CorrelationData, &err)
		case PropSubscriptionID:
			safeCopyPropPtr(ptr, &p.SubscriptionID, &err)
		}
	}
	up, ok := ps[PropUserProperty]
	if ok {
		p.User = up.(UserProperties)
	}
	return err
}
