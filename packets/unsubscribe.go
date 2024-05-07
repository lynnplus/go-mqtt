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

type Unsubscribe struct {
	Topics     []string
	PacketID   PacketID
	Properties *UnSubProperties
}

func (u *Unsubscribe) Pack(w io.Writer, header *FixedHeader) error {
	var err error
	header.Flags = 2
	if err = unsafeWriteUint16(w, &u.PacketID); err != nil {
		return err
	}
	if err = packPacketProperties(w, u.Properties, header.version); err != nil {
		return err
	}
	for _, topic := range u.Topics {
		if err = unsafeWriteString(w, &topic); err != nil {
			return err
		}
	}
	return nil
}

func (u *Unsubscribe) Unpack(r io.Reader, header *FixedHeader) error {
	var err error
	rr := &io.LimitedReader{R: r, N: int64(header.RemainLength)}
	if err = unsafeReadUint16(rr, &u.PacketID); err != nil {
		return err
	}

	if header.version >= ProtocolVersion5 {
		props := &UnSubProperties{}
		if err = props.Unpack(rr); err != nil {
			return err
		}
		u.Properties = props
	}

	for rr.N > 0 {
		var topic string
		if err = unsafeReadString(rr, &topic); err != nil {
			return err
		}
		u.Topics = append(u.Topics, topic)
	}
	return err
}

func (u *Unsubscribe) Type() PacketType {
	return UNSUBSCRIBE
}

func (u *Unsubscribe) ID() PacketID {
	return u.PacketID
}

type UnSubProperties struct {
	// User-defined properties, which is a string key-value pair
	UserProps UserProperties
}

func (s *UnSubProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, UNSUBSCRIBE)
	if err != nil {
		return err
	}
	up, ok := ps[PropUserProperty]
	if ok {
		s.UserProps = up.(UserProperties)
	}
	return err
}

func (s *UnSubProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error
	writeUserPropsData(buf, s.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}
