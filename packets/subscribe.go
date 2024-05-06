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

// PacketID is the type of packet identifier
type PacketID = uint16

type Subscription struct {
	Topic             string
	QoS               byte
	NoLocal           bool
	RetainAsPublished bool
	RetainHandling    byte
}

type Subscribe struct {
	PacketID      PacketID
	Subscriptions []Subscription
	Properties    *SubProperties
}

func (s *Subscribe) Type() PacketType {
	return SUBSCRIBE
}

func (s *Subscribe) ID() PacketID {
	return s.PacketID
}

func (s *Subscribe) Pack(w io.Writer, header *FixedHeader) error {
	var err error
	header.Flags = 2
	if err = unsafeWriteUint16(w, &s.PacketID); err != nil {
		return err
	}
	if err = packPacketProperties(w, s.Properties, header.version); err != nil {
		return err
	}

	for _, sub := range s.Subscriptions {
		if err = unsafeWriteString(w, &sub.Topic); err != nil {
			return err
		}
		var opt byte
		opt |= sub.QoS & 0x03
		if header.version >= ProtocolVersion5 {
			if sub.NoLocal {
				opt |= 1 << 2
			}
			if sub.RetainAsPublished {
				opt |= 1 << 3
			}
			opt |= (sub.RetainHandling << 4) & 0x30
		}
		if err = unsafeWriteByte(w, &opt); err != nil {
			return err
		}
	}
	return err
}

func (s *Subscribe) Unpack(r io.Reader, header *FixedHeader) error {
	var err error

	rr := &io.LimitedReader{R: r, N: int64(header.RemainLength)}
	if err = unsafeReadUint16(rr, &s.PacketID); err != nil {
		return err
	}
	if header.version >= ProtocolVersion5 {
		props := &SubProperties{}
		if err = props.Unpack(rr); err != nil {
			return err
		}
		s.Properties = props
	}
	if rr.N == 0 {
		return nil
	}
	//read topics and opts
	for rr.N > 0 {
		var topic string
		if err = unsafeReadString(rr, &topic); err != nil {
			return err
		}
		var b byte
		if err = unsafeReadByte(rr, &b); err != nil {
			return err
		}
		opt := Subscription{Topic: topic, QoS: b & 0x03}
		if header.version >= ProtocolVersion5 {
			opt.NoLocal = b&(1<<2) != 0
			opt.RetainAsPublished = b&(1<<3) != 0
			opt.RetainHandling = 3 & (b >> 4)
		}
		s.Subscriptions = append(s.Subscriptions, opt)
	}
	return err
}

type SubProperties struct {
	//Represents the subscription identifier. The value can be 1 to 268,435,455.
	// If the value is 0, it is a protocol error.
	SubscriptionID *int
	// User-defined properties, which is a string key-value pair
	UserProps UserProperties
}

func (s *SubProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, SUBSCRIBE)
	if err != nil {
		return err
	}
	sid, ok := ps[PropSubscriptionID]
	if ok {
		safeCopyPropPtr(sid, &s.SubscriptionID, &err)
	}
	up, ok := ps[PropUserProperty]
	if ok {
		s.UserProps = up.(UserProperties)
	}
	return err
}

func (s *SubProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error

	writePropIdAndValue(buf, PropSubscriptionID, s.SubscriptionID, &err)
	writeUserPropsData(w, s.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}
