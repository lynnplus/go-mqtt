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

type Disconnect struct {
	ReasonCode ReasonCode
	Properties *DisConnProperties
}

func (d *Disconnect) Pack(w io.Writer, header *FixedHeader) error {
	if d.ReasonCode == Success && d.Properties == nil {
		return nil
	}
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedValueOnVersion
	}
	var err error
	if err = unsafeWriteByte(w, (*byte)(&d.ReasonCode)); err != nil {
		return err
	}
	//when there are no Properties, the Properties length can be omitted
	if d.Properties == nil {
		return nil
	}
	return packPacketProperties(w, d.Properties, header.version)
}

func (d *Disconnect) Unpack(r io.Reader, header *FixedHeader) error {
	if header.RemainLength == 0 {
		return nil
	}
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedValueOnVersion
	}
	var err error
	if err = unsafeReadByte(r, (*byte)(&d.ReasonCode)); err != nil {
		return err
	}
	if header.RemainLength < 2 {
		return nil
	}
	props := &DisConnProperties{}
	if err = props.Unpack(r); err != nil {
		return err
	}
	d.Properties = props
	return nil
}

func (d *Disconnect) Type() PacketType {
	return DISCONNECT
}

func (d *Disconnect) ID() PacketID {
	return 0
}

type DisConnProperties struct {
	SessionExpiryInterval uint32
	ServerReference       string
	ReasonString          string
	// User-defined properties, which is a string key-value pair
	UserProps UserProperties
}

func (d *DisConnProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error

	if d.SessionExpiryInterval > 0 {
		writePropIdAndValue(buf, PropSessionExpiryInterval, &d.SessionExpiryInterval, &err)
	}
	if d.ServerReference != "" {
		writePropIdAndValue(buf, PropServerReference, &d.ServerReference, &err)
	}
	if d.ReasonString != "" {
		writePropIdAndValue(buf, PropReasonString, &d.ReasonString, &err)
	}
	writeUserPropsData(buf, d.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}

func (d *DisConnProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, DISCONNECT)
	if err != nil {
		return err
	}
	for id, ptr := range ps {
		if err != nil {
			return err
		}
		switch id {
		case PropSessionExpiryInterval:
			safeCopyPropValue(ptr, &d.SessionExpiryInterval, &err)
		case PropReasonString:
			safeCopyPropValue(ptr, &d.ReasonString, &err)
		case PropServerReference:
			safeCopyPropValue(ptr, &d.ServerReference, &err)
		}
	}
	up, ok := ps[PropUserProperty]
	if ok {
		d.UserProps = up.(UserProperties)
	}
	return err
}
