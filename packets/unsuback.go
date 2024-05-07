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
	"unsafe"
)

type UnsubackProperties = PubCommProperties

type Unsuback struct {
	PacketID    PacketID
	ReasonCodes []ReasonCode
	Properties  *UnsubackProperties
}

func (u *Unsuback) Pack(w io.Writer, header *FixedHeader) error {
	var err error
	if err = unsafeWriteUint16(w, &u.PacketID); err != nil {
		return err
	}
	if err = packPacketProperties(w, u.Properties, header.version); err != nil {
		return err
	}
	if len(u.ReasonCodes) <= 0 {
		return nil
	}
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedPropSetup
	}
	rc := unsafe.Slice((*byte)(unsafe.SliceData(u.ReasonCodes)), uint32(len(u.ReasonCodes)))
	return unsafeWriteBytes(w, &rc)
}

func (u *Unsuback) Unpack(r io.Reader, header *FixedHeader) error {
	var err error
	rr := &io.LimitedReader{R: r, N: int64(header.RemainLength)}
	if err = unsafeReadUint16(rr, &u.PacketID); err != nil {
		return err
	}
	if header.version < ProtocolVersion5 {
		if rr.N != 0 {
			return ErrUnsupportedPropSetup
		}
		return nil
	}
	props := &UnsubackProperties{}
	if err = props.Unpack(rr); err != nil {
		return err
	}
	u.Properties = props
	if rr.N <= 0 {
		return nil
	}

	u.ReasonCodes = make([]ReasonCode, rr.N)
	p := unsafe.Slice((*byte)(unsafe.SliceData(u.ReasonCodes)), rr.N)
	if _, err = rr.Read(p); err != nil {
		return err
	}
	return nil
}

func (u *Unsuback) Type() PacketType {
	return UNSUBACK
}

func (u *Unsuback) ID() PacketID {
	return u.PacketID
}
