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

type Unsuback struct {
}

func (u *Unsuback) Pack(w io.Writer, header *FixedHeader) error {
	//TODO implement me
	panic("implement me")
}

func (u *Unsuback) Unpack(r io.Reader, header *FixedHeader) error {
	//TODO implement me
	panic("implement me")
}

func (u *Unsuback) Type() PacketType {
	return UNSUBACK
}

func (u *Unsuback) ID() PacketID {
	//TODO implement me
	panic("implement me")
}