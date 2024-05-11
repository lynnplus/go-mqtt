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
	"math"
	"unsafe"
)

func unsafeReadWrap(r io.Reader, dst any, err *error) {
	if err != nil && *err != nil {
		return
	}
	switch v := dst.(type) {
	case *[]byte:
		*err = unsafeReadBytes(r, v)
	case *byte:
		*err = unsafeReadByte(r, v)
	case *string:
		*err = unsafeReadString(r, v)
	case *uint32:
		*err = unsafeReadUint32(r, v)
	case *uint16:
		*err = unsafeReadUint16(r, v)
	default:
		panic(fmt.Sprintf("unsupported read type %v", v))
	}
}

func unsafeReadString(r io.Reader, dst *string) error {
	var err error
	var n uint16
	if err = unsafeReadUint16(r, &n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	buf := make([]byte, n)
	if _, err = io.ReadFull(r, buf); err != nil {
		return err
	}
	*dst = unsafe.String(&buf[0], n)
	return nil
}

func unsafeReadByte(r io.Reader, dst *byte) error {
	cc := unsafe.Slice(dst, 1)
	if _, err := io.ReadFull(r, cc); err != nil {
		return err
	}
	return nil
}

func unsafeReadBytes(r io.Reader, dst *[]byte) error {
	var err error
	var n uint16
	if err = unsafeReadUint16(r, &n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	buf := make([]byte, n)
	if _, err = io.ReadFull(r, buf); err != nil {
		return err
	}
	*dst = buf
	return nil
}

func unsafeReadUint16(r io.Reader, v *uint16) error {
	cc := unsafe.Slice((*byte)(unsafe.Pointer(v)), 2)
	if _, err := io.ReadFull(r, cc); err != nil {
		return err
	}
	//mqtt use BigEndian
	*v = uint16(cc[0])<<8 | uint16(cc[1])
	return nil
}

func unsafeReadUint32(r io.Reader, v *uint32) error {
	cc := unsafe.Slice((*byte)(unsafe.Pointer(v)), 4)
	if _, err := io.ReadFull(r, cc); err != nil {
		return err
	}
	//mqtt use BigEndian
	*v = uint32(cc[0])<<24 | uint32(cc[1])<<16 | uint32(cc[2])<<8 | uint32(cc[3])
	return nil
}

func unsafeWriteString(w io.Writer, v *string) error {
	size := len(*v)
	if size > math.MaxUint16 {
		return ErrWriteStringLimit
	}
	n := uint16(size)
	if err := unsafeWriteUint16(w, &n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	b := unsafe.Slice(unsafe.StringData(*v), n)
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func unsafeWriteByte(w io.Writer, v *byte) error {
	cc := unsafe.Slice(v, 1)
	_, err := w.Write(cc)
	return err
}

func unsafeWriteUint16(w io.Writer, v *uint16) error {
	cc := unsafe.Slice((*byte)(unsafe.Pointer(v)), 2)
	_, err := w.Write(cc[1:])
	if err != nil {
		return err
	}
	_, err = w.Write(cc[:1])
	return err
}

func unsafeWriteVbi(w io.Writer, v *int) error {
	b := [4]byte{}
	n, e := encodeVBI(*v, b[:])
	if e != nil {
		return e
	}
	_, e = w.Write(b[:n])
	return e
}

func unsafeWriteUint32(w io.Writer, v *uint32) error {
	cc := unsafe.Slice((*byte)(unsafe.Pointer(v)), 4)
	for i := 3; i >= 0; i-- {
		if _, err := w.Write(cc[i : i+1]); err != nil {
			return err
		}
	}
	return nil
}

func unsafeWriteBytes(w io.Writer, v *[]byte) error {
	size := len(*v)
	if size > math.MaxUint16 {
		return ErrWriteStringLimit
	}
	n := uint16(size)
	if err := unsafeWriteUint16(w, &n); err != nil {
		return err
	}
	_, err := w.Write(*v)
	return err
}

func unsafeWriteWrap(w io.Writer, dst any, err *error) {
	if err != nil && *err != nil {
		return
	}
	switch v := dst.(type) {
	case *[]byte:
		*err = unsafeWriteBytes(w, v)
	case *bool:
		b := (*byte)(unsafe.Pointer(v))
		*err = unsafeWriteByte(w, b)
	case *byte:
		*err = unsafeWriteByte(w, v)
	case *string:
		*err = unsafeWriteString(w, v)
	case *uint32:
		*err = unsafeWriteUint32(w, v)
	case *uint16:
		*err = unsafeWriteUint16(w, v)
	case *int:
		*err = unsafeWriteVbi(w, v)
	default:
		panic(fmt.Sprintf("unsupported write type %v", v))
	}
}
