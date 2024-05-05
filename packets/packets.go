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

type Packet interface {
	// Pack encodes the packet struct into bytes and writes it into io.Writer.
	Pack(w io.Writer) error
	// Unpack read the packet bytes from io.Reader and decodes it into the packet struct
	Unpack(r io.Reader) error
}

// FixedHeader represents the FixedHeader of the MQTT packet
//
// contain: PacketType(4bit) + Flags(4bit) + RemainLength(1-4 Bytes)
type FixedHeader struct {
	PacketType   PacketType
	Flags        byte
	RemainLength int
	buf          [5]byte
}

func (fh *FixedHeader) ReadFrom(r io.Reader) (int64, error) {
	n, err := io.ReadFull(r, fh.buf[:1])
	if err != nil {
		return int64(n), err
	}
	fh.PacketType = PacketType(fh.buf[0] >> 4)
	fh.Flags = fh.buf[0] & 0x0F

	if !fh.PacketType.IsValid() {
		return int64(n), fmt.Errorf("unknown packet type %d", fh.PacketType)
	}
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

type Content interface {
}

func Read(r io.Reader) (Packet, error) {
	header := &FixedHeader{}
	if _, err := header.ReadFrom(r); err != nil {
		return nil, err
	}
	var content Content

	switch header.PacketType {
	case CONNECT:
		content = &Connect{}
	case CONNACK:
	case PUBLISH:
	case PUBACK:
	case PUBREC:
	case PUBREL:
	case PUBCOMP:
	case SUBSCRIBE:
	case SUBACK:
	case UNSUBSCRIBE:
	case UNSUBACK:
	case PINGREQ:
	case PINGRESP:
	case DISCONNECT:
	case AUTH:
	}

	fmt.Println(content)
	return nil, nil
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
