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

type PubComm struct {
	PacketID   PacketID
	ReasonCode ReasonCode
	Properties *PubCommProperties
}

func (p *PubComm) ID() PacketID {
	return p.PacketID
}

func (p *PubComm) Pack(w io.Writer, header *FixedHeader) error {
	var err error
	if err = unsafeWriteUint16(w, &p.PacketID); err != nil {
		return err
	}
	if p.ReasonCode == 0 && p.Properties == nil {
		// If the reason code is 0x00 (success) and there are no attributes,
		// the reason code and attribute length can be omitted
		return nil
	}
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedValueOnVersion
	}

	//write the third byte
	if err = unsafeWriteByte(w, (*byte)(&p.ReasonCode)); err != nil {
		return err
	}
	//when there are no Properties, the Properties length can be omitted
	if p.Properties == nil {
		return nil
	}
	if err = packPacketProperties(w, p.Properties, header.version); err != nil {
		return err
	}
	return nil
}

func (p *PubComm) Unpack(r io.Reader, header *FixedHeader) error {
	var err error
	if err = unsafeReadUint16(r, &p.PacketID); err != nil {
		return err
	}
	if header.RemainLength == 2 {
		return nil
	}
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedValueOnVersion
	}
	if err = unsafeReadByte(r, (*byte)(&p.ReasonCode)); err != nil {
		return err
	}
	if header.RemainLength < 4 {
		return nil
	}
	props := &PubCommProperties{}
	if err = props.Unpack(r); err != nil {
		return err
	}
	p.Properties = props
	return nil
}

type Puback struct {
	PubComm
}

func (p *Puback) Type() PacketType {
	return PUBACK
}

type Pubcomp struct {
	PubComm
}

func (p *Pubcomp) Type() PacketType {
	return PUBCOMP
}

type Pubrec struct {
	PubComm
}

func (p *Pubrec) Type() PacketType {
	return PUBREC
}

type Pubrel struct {
	PubComm
}

func (p *Pubrel) Type() PacketType {
	return PUBREL
}

func (p *Pubrel) Pack(w io.Writer, header *FixedHeader) error {
	header.Flags = 2
	return p.PubComm.Pack(w, header)
}

type PubCommProperties struct {
	ReasonString string
	// User-defined properties, which is a string key-value pair
	UserProps UserProperties
}

func (p *PubCommProperties) Pack(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func (p *PubCommProperties) Unpack(r io.Reader) error {
	//TODO implement me
	panic("implement me")
}
