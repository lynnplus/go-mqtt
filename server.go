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
	"github.com/lynnplus/go-mqtt/packets"
)

// ServerProperties is a struct that holds server properties
type ServerProperties struct {
	MaximumPacketSize    uint32
	ReceiveMaximum       uint16
	TopicAliasMaximum    uint16
	MaximumQoS           byte
	RetainAvailable      bool
	WildcardSubAvailable bool
	SubIDAvailable       bool
	SharedSubAvailable   bool
}

func NewServerProperties() *ServerProperties {
	return &ServerProperties{
		ReceiveMaximum:       65535,
		MaximumQoS:           2,
		MaximumPacketSize:    0, //0:unlimited
		TopicAliasMaximum:    0, //0:unlimited
		RetainAvailable:      true,
		WildcardSubAvailable: true,
		SubIDAvailable:       true,
		SharedSubAvailable:   true,
	}
}

func (s *ServerProperties) ReconfigureFromResponse(resp *packets.Connack) {
	if resp.Properties != nil {
		prop := resp.Properties
		s.ReceiveMaximum = prop.ReceiveMaximum
		s.TopicAliasMaximum = prop.TopicAliasMaximum
		if prop.MaximumQoS != nil {
			s.MaximumQoS = *prop.MaximumQoS
		}
		if prop.RetainAvailable != nil {
			s.RetainAvailable = *prop.RetainAvailable
		}
		if prop.MaximumPacketSize != nil {
			s.MaximumPacketSize = *prop.MaximumPacketSize
		}
		if prop.WildcardSubAvailable != nil {
			s.WildcardSubAvailable = *prop.WildcardSubAvailable
		}
		if prop.SharedSubAvailable != nil {
			s.SharedSubAvailable = *prop.SharedSubAvailable
		}
		if prop.SubIdAvailable != nil {
			s.SubIDAvailable = *prop.SubIdAvailable
		}
	}
}
