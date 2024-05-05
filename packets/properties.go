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
	"fmt"
	"io"
	"unsafe"
)

// PropertyID is the identifier for MQTT properties,only MQTT v5 supported,reference doc: "[Properties]"
//
// [Properties]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901027
type PropertyID byte

const (
	_                         PropertyID = 0 // Reserved
	PropPayloadFormat         PropertyID = 1
	PropMessageExpiryInterval PropertyID = 2
	PropContentType           PropertyID = 3
	PropResponseTopic         PropertyID = 8
	PropCorrelationData       PropertyID = 9
	PropSubscriptionID        PropertyID = 11
	PropSessionExpiryInterval PropertyID = 17
	PropAssignedClientID      PropertyID = 18
	PropServerKeepAlive       PropertyID = 19
	PropAuthMethod            PropertyID = 21 // Property identifier of the authentication method prop
	PropAuthData              PropertyID = 22 // Property identifier of the authentication data prop
	PropRequestProblemInfo    PropertyID = 23
	PropWillDelayInterval     PropertyID = 24
	PropRequestResponseInfo   PropertyID = 25
	PropResponseInfo          PropertyID = 26
	PropServerReference       PropertyID = 28
	PropReasonString          PropertyID = 31
	PropReceiveMaximum        PropertyID = 33
	PropTopicAliasMaximum     PropertyID = 34
	PropTopicAlias            PropertyID = 35
	PropMaximumQOS            PropertyID = 36
	PropRetainAvailable       PropertyID = 37
	PropUserProperty          PropertyID = 38
	PropMaximumPacketSize     PropertyID = 39
	PropWildcardSubAvailable  PropertyID = 40 // Property identifier of the wildcard subscription available prop
	PropSubIDAvailable        PropertyID = 41 // Property identifier of the subscription identifier available prop
	PropSharedSubAvailable    PropertyID = 42 // Property identifier of the shared subscription available prop
)

// UserProperties is a map for the User Properties.
// although user attributes are allowed to appear more than once in the protocol,
// various broker servers handle repeated attributes differently, at the same time,
// in order to eliminate ambiguity, user attributes are defined as a map with a non-repeatable key.
type UserProperties map[string]string

var validPacketProperties = map[PropertyID]map[PacketType]uint8{
	PropPayloadFormat:         {PUBLISH: 1},
	PropMessageExpiryInterval: {PUBLISH: 1},
	PropContentType:           {PUBLISH: 1},
	PropResponseTopic:         {PUBLISH: 1},
	PropCorrelationData:       {PUBLISH: 1},
	PropSubscriptionID:        {PUBLISH: 1},
	PropSessionExpiryInterval: {CONNECT: 1, CONNACK: 1, DISCONNECT: 1},
	PropAssignedClientID:      {CONNACK: 1},
	PropServerKeepAlive:       {CONNACK: 1},
	PropAuthMethod:            {CONNECT: 1, CONNACK: 1, AUTH: 1},
	PropAuthData:              {CONNECT: 1, CONNACK: 1, AUTH: 1},
	PropRequestProblemInfo:    {CONNECT: 1},
	PropWillDelayInterval:     {},
	PropRequestResponseInfo:   {CONNECT: 1},
	PropResponseInfo:          {CONNACK: 1},
	PropServerReference:       {CONNACK: 1, DISCONNECT: 1},
	PropReasonString:          {CONNACK: 1, PUBACK: 1, PUBREC: 1, PUBREL: 1, PUBCOMP: 1, SUBACK: 1, UNSUBACK: 1, DISCONNECT: 1, AUTH: 1},
	PropReceiveMaximum:        {CONNECT: 1, CONNACK: 1},
	PropTopicAliasMaximum:     {CONNECT: 1, CONNACK: 1},
	PropTopicAlias:            {PUBLISH: 1},
	PropMaximumQOS:            {CONNACK: 1},
	PropRetainAvailable:       {CONNACK: 1},
	PropUserProperty:          {CONNECT: 1, CONNACK: 1, PUBLISH: 1, PUBACK: 1, PUBREC: 1, PUBREL: 1, PUBCOMP: 1, SUBSCRIBE: 1, SUBACK: 1, UNSUBSCRIBE: 1, UNSUBACK: 1, DISCONNECT: 1, AUTH: 1},
	PropMaximumPacketSize:     {CONNECT: 1, CONNACK: 1},
	PropWildcardSubAvailable:  {CONNACK: 1},
	PropSubIDAvailable:        {CONNACK: 1},
	PropSharedSubAvailable:    {CONNACK: 1},
}

var validWillProperties = map[PropertyID]uint8{
	PropPayloadFormat:         1,
	PropMessageExpiryInterval: 1,
	PropContentType:           1,
	PropResponseTopic:         1,
	PropCorrelationData:       1,
	PropWillDelayInterval:     1,
	PropUserProperty:          1,
}

// ValidatePropID takes a PropertyID and a PacketType and returns
// a bool indicating if that property is valid for that PacketType
func ValidatePropID(prop PropertyID, pt PacketType) bool {
	_, ok := validPacketProperties[prop][pt]
	return ok
}

// ValidateWillPropID takes a PropertyID and returns
// a bool indicating if that property is valid for that WillProperties
func ValidateWillPropID(prop PropertyID) bool {
	_, ok := validWillProperties[prop]
	return ok
}

func newInvalidPropValueError(id PropertyID, value any) error {
	return NewReasonCodeError(ProtocolError, fmt.Sprintf("invalid prop value %v for property %v", value, id))
}

func CopyPropPtrValue[T any](src map[PropertyID]any, id PropertyID, dstPtr *T, defaultValue T) bool {
	if dstPtr == nil {
		panic("destination pointer is nil")
	}
	v, ok := src[id]
	if !ok {
		*dstPtr = defaultValue
		return false
	}
	m, ok2 := v.(*T)
	if !ok2 {
		panic("type mismatch")
	}
	*dstPtr = *m
	return true
}

func ReadPacketProperties(r io.Reader, packetType PacketType) (map[PropertyID]any, error) {
	return readAllProperties(r, func(id PropertyID) error {
		if !ValidatePropID(id, packetType) {
			return NewReasonCodeError(MalformedPacket, fmt.Sprintf("invalid property %d for %s", id, packetType))
		}
		return nil
	})
}

func ReadWillProperties(r io.Reader) (map[PropertyID]any, error) {
	return readAllProperties(r, func(id PropertyID) error {
		if !ValidateWillPropID(id) {
			return NewReasonCodeError(ProtocolError, fmt.Sprintf("invalid property %d for will properties", id))
		}
		return nil
	})
}

func writePropIdAndValue(w io.Writer, id PropertyID, value any, err *error) {
	unsafeWriteWrap(w, (*byte)(&id), err)
	unsafeWriteWrap(w, value, err)
}

func writeUserPropsData(w io.Writer, props UserProperties, err *error) {
	if len(props) == 0 {
		return
	}
	if err != nil && *err != nil {
		return
	}
	id := PropUserProperty
	unsafeWriteWrap(w, (*byte)(&id), err)
	for k, v := range props {
		unsafeWriteWrap(w, (*byte)(&id), err)
		unsafeWriteWrap(w, &k, err)
		unsafeWriteWrap(w, &v, err)
	}
}

func writePropertiesData(w io.Writer, data []byte) error {
	vbi := [4]byte{}
	n, err := encodeVBI(len(data), vbi[:])
	if err != nil {
		return err
	}
	if _, err = w.Write(vbi[:n]); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readAllProperties(r io.Reader, validate func(PropertyID) error) (map[PropertyID]any, error) {
	buf := [4]byte{}
	totalSize, _, err := decodeVBI(r, buf[:])

	props := map[PropertyID]any{}
	var propId PropertyID
	for {
		if err != nil {
			return nil, err
		}
		if totalSize == 0 {
			return props, err
		}
		if err = unsafeReadByte(r, (*byte)(&propId)); err != nil {
			return nil, err
		}

		if err = validate(propId); err != nil {
			return nil, err
		}
		totalSize -= 1
		switch propId {
		case PropMaximumQOS:
			var v byte
			unsafeReadWrap(r, &v, &err)
			if err != nil {
				return nil, err
			}
			if v != 0 && v != 1 {
				return nil, newInvalidPropValueError(propId, v)
			}
			props[propId] = &v
			totalSize -= 1
		case PropMessageExpiryInterval, PropSessionExpiryInterval, PropMaximumPacketSize, PropWillDelayInterval:
			var v uint32
			unsafeReadWrap(r, &v, &err)
			if propId == PropMaximumPacketSize && v == 0 {
				return nil, newInvalidPropValueError(propId, v)
			}
			props[propId] = &v
			totalSize -= 4
		case PropContentType, PropResponseTopic, PropAssignedClientID, PropAuthMethod, PropResponseInfo, PropServerReference, PropReasonString:
			var v string
			unsafeReadWrap(r, &v, &err)
			props[propId] = &v
			totalSize -= 2 + len(v)
		case PropCorrelationData, PropAuthData:
			var v []byte
			unsafeReadWrap(r, &v, &err)
			props[propId] = &v
			totalSize -= 2 + len(v)
		case PropSubscriptionID:
			//vbi the value of 1 to 268,435,455
			sidBs := [4]byte{}
			l, n, e := decodeVBI(r, sidBs[:])
			if e != nil {
				return nil, e
			}
			if l == 0 {
				return nil, newInvalidPropValueError(propId, l)
			}
			props[propId] = &l
			totalSize -= n

		case PropServerKeepAlive, PropReceiveMaximum, PropTopicAliasMaximum, PropTopicAlias:
			var v uint16
			unsafeReadWrap(r, &v, &err)
			props[propId] = &v
			totalSize -= 2
			if propId == PropReceiveMaximum && v == 0 {
				return nil, newInvalidPropValueError(propId, v)
			}
		case PropPayloadFormat, PropRequestProblemInfo, PropRequestResponseInfo, PropRetainAvailable, PropWildcardSubAvailable, PropSubIDAvailable, PropSharedSubAvailable:
			var v bool
			b := (*byte)(unsafe.Pointer(&v))
			unsafeReadWrap(r, &b, &err)
			if err != nil {
				return nil, err
			}
			if *b != 0 && *b != 1 {
				return nil, newInvalidPropValueError(propId, *b)
			}
			props[propId] = &v
			totalSize -= 1
		case PropUserProperty:
			up, ok := props[propId]
			if !ok {
				props[propId] = UserProperties{}
				up = props[propId]
			}
			u := up.(UserProperties)
			var k, v string
			unsafeReadWrap(r, &k, &err)
			unsafeReadWrap(r, &v, &err)
			u[k] = v
			totalSize -= 4 + len(k) + len(v)
		default:
			return nil, NewReasonCodeError(ProtocolError, fmt.Sprintf("unknown prop identifier %v", propId))
		}
	}
}
