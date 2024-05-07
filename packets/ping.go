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

type Pingreq struct{}

func (p *Pingreq) Pack(w io.Writer, header *FixedHeader) error {
	return nil
}

func (p *Pingreq) Unpack(r io.Reader, header *FixedHeader) error {
	return nil
}

func (p *Pingreq) Type() PacketType {
	return PINGREQ
}

func (p *Pingreq) ID() PacketID {
	return 0
}

type Pingresp struct {
}

func (p *Pingresp) Pack(w io.Writer, header *FixedHeader) error {
	return nil
}

func (p *Pingresp) Unpack(r io.Reader, header *FixedHeader) error {
	return nil
}

func (p *Pingresp) Type() PacketType {
	return PINGRESP
}

func (p *Pingresp) ID() PacketID {
	return 0
}
