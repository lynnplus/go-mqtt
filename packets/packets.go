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

// Package packets implements the data packet type definition of mqtt v3.1.1
// and mqtt v5.0 as well as the reader and writer of data packets
package packets

import (
	"errors"
	"fmt"
	"io"
)

type PacketType byte

// IsValid reports whether the PacketType is a valid mqtt packet type.
func (p PacketType) IsValid() bool {
	return p >= CONNECT && p <= AUTH
}

func (p PacketType) String() string {
	switch p {
	case CONNECT:
		return "CONNECT"
	case CONNACK:
		return "CONNACK"
	case PUBLISH:
		return "PUBLISH"
	case PUBACK:
		return "PUBACK"
	case PUBREC:
		return "PUBREC"
	case PUBREL:
		return "PUBREL"
	case PUBCOMP:
		return "PUBCOMP"
	case SUBSCRIBE:
		return "SUBSCRIBE"
	case SUBACK:
		return "SUBACK"
	case UNSUBSCRIBE:
		return "UNSUBSCRIBE"
	case UNSUBACK:
		return "UNSUBACK"
	case PINGREQ:
		return "PINGREQ"
	case PINGRESP:
		return "PINGRESP"
	case DISCONNECT:
		return "DISCONNECT"
	case AUTH:
		return "AUTH"
	default:
		panic(fmt.Errorf("unknown packet type 0x%x", byte(p)))
	}
}

const (
	CONNECT PacketType = iota + 1
	CONNACK
	PUBLISH
	PUBACK
	PUBREC
	PUBREL
	PUBCOMP
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ
	PINGRESP
	DISCONNECT
	AUTH
)

// PacketID is the type of packet identifier
type PacketID = uint16

type Packet interface {
	// Pack encodes the packet struct into bytes and writes it into io.Writer.
	Pack(w io.Writer, header *FixedHeader) error
	// Unpack read the packet bytes from io.Reader and decodes it into the packet struct
	Unpack(r io.Reader, header *FixedHeader) error
	// Type returns the packet type of this packet
	Type() PacketType
	// ID returns the current PacketID. If the packet does not support PacketID, returns 0.
	ID() PacketID
}

// FixedHeader represents the FixedHeader of the MQTT packet
//
// contain: PacketType(4bit) + Flags(4bit) + RemainLength(1-4 Bytes)
type FixedHeader struct {
	PacketType   PacketType
	Flags        byte
	RemainLength int
	buf          [5]byte
	version      ProtocolVersion
}

func (fh *FixedHeader) ReadFrom(r io.Reader) (int64, error) {
	n, err := io.ReadFull(r, fh.buf[:1])
	if err != nil {
		return int64(n), err
	}
	fh.PacketType = PacketType(fh.buf[0] >> 4)
	fh.Flags = fh.buf[0] & 0x0F
	fh.RemainLength, n, err = decodeVBI(r, fh.buf[1:])
	return int64(n + 1), err
}

func (fh *FixedHeader) WriteTo(w io.Writer) (int64, error) {
	fh.buf[0] = byte(fh.PacketType)<<4 | fh.Flags
	n, err := encodeVBI(fh.RemainLength, fh.buf[1:])
	if err != nil {
		return 0, err
	}
	if _, err = w.Write(fh.buf[:1+n]); err != nil {
		return 0, err
	}
	return int64(1 + n), nil
}

// ValidateHeaderFlags takes a byte value and a PacketType and returns
// a bool indicating if that flags value is valid for that PacketType
func ValidateHeaderFlags(flags byte, pt PacketType) bool {
	if pt == SUBSCRIBE || pt == UNSUBSCRIBE || pt == PUBREL {
		return flags == 2
	}
	if pt == PUBLISH {
		return true
	}
	return flags == 0
}

func Read(r io.Reader, version ProtocolVersion) (Packet, error) {
	header := &FixedHeader{version: version}
	if _, err := header.ReadFrom(r); err != nil {
		return nil, err
	}
	if !header.PacketType.IsValid() {
		return nil, NewReasonCodeError(ProtocolError, fmt.Sprintf("unknown packet type %d", header.PacketType))
	}
	if version < ProtocolVersion5 && header.PacketType == AUTH {
		return nil, NewReasonCodeError(ProtocolError, "the current version does not support AUTH packets")
	}
	if !ValidateHeaderFlags(header.Flags, header.PacketType) {
		return nil, ErrInvalidPktFlags
	}

	var pkt Packet
	switch header.PacketType {
	case CONNECT:
		pkt = &Connect{}
	case CONNACK:
		pkt = &Connack{}
	case PUBLISH:
		pkt = &Publish{}
	case PUBACK:
		pkt = &Puback{}
	case PUBREC:
		pkt = &Pubrec{}
	case PUBREL:
		pkt = &Pubrel{}
	case PUBCOMP:
		pkt = &Pubcomp{}
	case SUBSCRIBE:
		pkt = &Subscribe{}
	case SUBACK:
		pkt = &Suback{}
	case UNSUBSCRIBE:
		pkt = &Unsubscribe{}
	case UNSUBACK:
		pkt = &Unsuback{}
	case PINGREQ:
		pkt = &Pingreq{}
	case PINGRESP:
		pkt = &Pingresp{}
	case DISCONNECT:
		pkt = &Disconnect{}
	case AUTH:
		pkt = &Auth{}
	default:
		return nil, NewReasonCodeError(ProtocolError, fmt.Sprintf("unknown packet type %d", header.PacketType))
	}
	if err := pkt.Unpack(r, header); err != nil {
		return nil, err
	}
	return pkt, nil
}

// encode to bytes for variable byte integer,maximum is 2^28 - 1(268,435,455)B,256MB
func encodeVBI(length int, buf []byte) (int, error) {
	for i := 0; i < len(buf); i++ {
		buf[i] = byte(length % 128)
		length /= 128
		if length > 0 {
			buf[i] |= 0x80
		}
		if length == 0 {
			return i + 1, nil
		}
	}
	return 0, errors.New("fail to encode VBI")
}

func decodeVBI(r io.Reader, buf []byte) (length int, n int, err error) {
	var vbi uint32
	var i int
	for i = 0; i < len(buf); i++ {
		if n, err = io.ReadFull(r, buf[i:i+1]); err != nil {
			return 0, n, err
		}
		vbi |= uint32(buf[i]&127) << (i * 7)
		if (buf[i] & 128) == 0 {
			break
		}
	}
	return int(vbi), i + 1, nil
}
