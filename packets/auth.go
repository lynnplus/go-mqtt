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

type Auth struct {
	ReasonCode ReasonCode
	Properties *AuthProperties
}

func (a *Auth) Pack(w io.Writer, header *FixedHeader) error {
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedValueOnVersion
	}
	//The Reason Code and Property Length can be omitted if the Reason Code is 0x00 (Success) and there are no Properties.
	//In this case the AUTH has a Remaining Length of 0.
	if a.ReasonCode == Success && a.Properties == nil {
		return nil
	}
	var err error
	if err = unsafeWriteByte(w, (*byte)(&a.ReasonCode)); err != nil {
		return err
	}
	//when there are no Properties, the Properties length can be omitted
	if a.Properties == nil {
		return nil
	}
	return packPacketProperties(w, a.Properties, header.version)
}

func (a *Auth) Unpack(r io.Reader, header *FixedHeader) error {
	if header.version < ProtocolVersion5 {
		return ErrUnsupportedValueOnVersion
	}
	if header.RemainLength == 0 {
		return nil
	}
	var err error
	if err = unsafeReadByte(r, (*byte)(&a.ReasonCode)); err != nil {
		return err
	}
	if header.RemainLength < 2 {
		return nil
	}
	props := &AuthProperties{}
	if err = props.Unpack(r); err != nil {
		return err
	}
	a.Properties = props
	return nil

}

func (a *Auth) Type() PacketType {
	return AUTH
}

func (a *Auth) ID() PacketID {
	return 0
}

type AuthProperties struct {
	AuthMethod   string
	AuthData     []byte
	ReasonString string
	// User-defined properties, which is a string key-value pair
	UserProps UserProperties
}

func (a *AuthProperties) Pack(w io.Writer) error {
	buf := bytes.NewBuffer([]byte{})
	var err error
	if a.ReasonString != "" {
		writePropIdAndValue(buf, PropReasonString, &a.ReasonString, &err)
	}
	if a.AuthMethod != "" {
		writePropIdAndValue(buf, PropAuthMethod, &a.AuthMethod, &err)
		writePropIdAndValue(buf, PropAuthData, &a.AuthData, &err)
	}
	writeUserPropsData(w, a.UserProps, &err)
	if err != nil {
		return err
	}
	return writePropertiesData(w, buf.Bytes())
}

func (a *AuthProperties) Unpack(r io.Reader) error {
	ps, err := ReadPacketProperties(r, CONNACK)
	if err != nil {
		return err
	}
	for id, ptr := range ps {
		if err != nil {
			return err
		}
		switch id {
		case PropReasonString:
			safeCopyPropValue(ptr, &a.ReasonString, &err)
		case PropAuthMethod:
			safeCopyPropValue(ptr, &a.AuthMethod, &err)
		case PropAuthData:
			safeCopyPropValue(ptr, &a.AuthData, &err)
		}
	}
	up, ok := ps[PropUserProperty]
	if ok {
		a.UserProps = up.(UserProperties)
	}
	return err
}
