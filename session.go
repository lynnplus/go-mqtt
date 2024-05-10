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

package mqtt

import (
	"errors"
	"fmt"
	"github.com/lynnplus/go-mqtt/packets"
	"github.com/lynnplus/gotypes"
	"math"
	"sync/atomic"
)

type SessionState interface {
	ConnectionCompleted(conn *packets.Connect, ack *packets.Connack) error
	ConnectionLost(dp *packets.Disconnect) error

	// SubmitPacket submits a data packet to the session state and assigns an available id to the data packet
	SubmitPacket(pkt IdentifiablePacket) (<-chan packets.Packet, error)
	// RevokePacket revokes a submitted packet from the session state.
	// It only supports the revocation of limited packet types.
	RevokePacket(pkt packets.Packet) error
	ResponsePacket(pkt packets.Packet) error
}

type SessionStore interface {
}

type IdentifiablePacket interface {
	packets.Packet
	SetID(id packets.PacketID)
}

type RequestedPacket struct {
	PacketType packets.PacketType
	Response   chan packets.Packet
	Consumed   bool
}

var ErrPacketIdentifiersExhausted = errors.New("packet identifiers exhausted")

type DefaultSession struct {
	store         SessionStore
	clientPackets gotypes.SafeMap[packets.PacketID, *RequestedPacket]
	lastPid       atomic.Uint32
	logger        Logger
}

func NewDefaultSession() *DefaultSession {
	return &DefaultSession{
		clientPackets: gotypes.NewRWMutexMap[packets.PacketID, *RequestedPacket](),
		logger:        &EmptyLogger{},
	}
}

func (s *DefaultSession) SubmitPacket(pkt IdentifiablePacket) (<-chan packets.Packet, error) {
	req := &RequestedPacket{
		PacketType: pkt.Type(),
		Response:   make(chan packets.Packet, 1),
	}
	pid, err := s.assignPacketSlot(req)
	if err != nil {
		return nil, err
	}
	pkt.SetID(pid)
	return req.Response, nil
}

func (s *DefaultSession) RevokePacket(pkt packets.Packet) error {
	if pkt.ID() == 0 {
		return errors.New("unable to revoke a Packet with packet identifier 0")
	}
	switch pkt.Type() {
	case packets.SUBSCRIBE, packets.UNSUBSCRIBE:
		if _, ok := s.clientPackets.LoadAndDelete(pkt.ID()); !ok {
			return fmt.Errorf("revoke Packet(%v) failed, no data found", pkt.ID())
		}
		return nil
	default:
		return fmt.Errorf("Packet %s  do not support revoke operations", pkt.Type())
	}
}

func (s *DefaultSession) ResponsePacket(pkt packets.Packet) error {
	switch pkt.Type() {
	case packets.PUBACK:
	case packets.PUBREC:
	case packets.PUBREL:
	case packets.PUBCOMP:
	case packets.SUBACK, packets.UNSUBACK:
		if v, ok := s.clientPackets.LoadAndDelete(pkt.ID()); ok {
			if v.Response != nil {
				v.Response <- pkt
			}
			v.Consumed = true
		} else {
			return fmt.Errorf("response processing data corresponding to the packet(%v %s) was not found", pkt.ID(), pkt.Type())
		}
	default:
		return fmt.Errorf("unknown packet：%v %s", pkt.ID(), pkt.Type())
	}
	return nil
}

func (s *DefaultSession) ConnectionCompleted(conn *packets.Connect, ack *packets.Connack) error {
	return nil
}

func (s *DefaultSession) ConnectionLost(dp *packets.Disconnect) error {
	return nil
}

func (s *DefaultSession) assignPacketSlot(pkt *RequestedPacket) (packets.PacketID, error) {
	id := s.lastPid.Load()
	//TODO max id,use prop config
	if id >= math.MaxUint16-30 {
		s.lastPid.Swap(0)
	}
	for id < math.MaxUint16 {
		id = s.lastPid.Add(1)
		_, loaded := s.clientPackets.LoadOrStore(packets.PacketID(id), pkt)
		if loaded {
			continue
		}
		return packets.PacketID(id), nil
	}

	for i := uint16(1); i < uint16(id); i++ {
		_, loaded := s.clientPackets.LoadOrStore(packets.PacketID(id), pkt)
		if loaded {
			continue
		}
		return packets.PacketID(id), nil
	}
	return 0, ErrPacketIdentifiersExhausted
}
