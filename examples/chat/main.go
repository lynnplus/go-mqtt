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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/lynnplus/go-mqtt/packets"
	"io"
	"unsafe"
)

func unsafeReadByte(r io.Reader, dst *byte) error {
	cc := unsafe.Slice(dst, 1)
	if _, err := io.ReadFull(r, cc); err != nil {
		return err
	}
	return nil
}

func Ret() error {
	return errors.New("11")
	//return nil
}

type Pkt int

func (p Pkt) String() string {
	switch p {
	case PktA:
		return "A"
	case PktB:
		return "B"
	}
	return fmt.Sprintf("err %d", p)
}

const (
	PktA Pkt = 1
	PktB Pkt = 2
)

func Abc(err *error) {
	fmt.Println(err)

	if err != nil {
		fmt.Println("-->", *err, *err == nil)
	}
}

type Users map[string]string

func pri(v *string) {
	fmt.Println(*v)
}

func main() {

	a := map[string]string{"a": "1", "b": "2"}
	fmt.Println(a)

	buf := bytes.NewBuffer([]byte{})

	for k, v := range a {

		fmt.Println(&k, &v)
		if err := packets.TestWrString(buf, &k); err != nil {
			panic(err)
		}
		if err := packets.TestWrString(buf, &v); err != nil {
			panic(err)
		}
	}

	fmt.Println(string(buf.Bytes()), buf.Len())

}
