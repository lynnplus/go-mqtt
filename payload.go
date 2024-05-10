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
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/lynnplus/gotypes"
)

type PayloadUnmarshaler func(body []byte, v any) error

var payloadUnmarshalers gotypes.SafeMap[string, PayloadUnmarshaler]

func init() {
	payloadUnmarshalers = gotypes.NewSyncMap[string, PayloadUnmarshaler]()
	RegisterPayloadUnmarshaler("json", json.Unmarshal)
	RegisterPayloadUnmarshaler("xml", xml.Unmarshal)
	RegisterPayloadUnmarshaler("text", UnmarshalToString)
	RegisterPayloadUnmarshaler("application/json", json.Unmarshal)
	RegisterPayloadUnmarshaler("application/xml", xml.Unmarshal)
	RegisterPayloadUnmarshaler("text/plain", UnmarshalToString)
}

func RegisterPayloadUnmarshaler(contentType string, unmarshaler PayloadUnmarshaler) {
	payloadUnmarshalers.Store(contentType, unmarshaler)
}

func UnmarshalToString(body []byte, v any) error {

	s, ok := v.(*string)
	if !ok {
		return errors.New("mqtt.context: UnmarshalToString: invalid type assertion")
	}
	*s = string(body)
	return nil
}

func UnmarshalPayload(contentType string, body []byte, v any) error {
	unmarshaler, ok := payloadUnmarshalers.Load(contentType)
	if !ok {
		return errors.New("mqtt.context: invalid packet")
	}
	return unmarshaler(body, v)
}
