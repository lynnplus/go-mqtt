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

// Auther is the interface that implements the enhanced identity authentication flow in the mqtt5 protocol.
//
// The general flow is that the user carries authentication data in the Connect packet. After the server authenticates,
// if it needs to continue to authenticate, it will send back an Auth packet with the reason code Continue Authentication.
// The client starts the next step of authentication and sends it to the server again until Server sends Connack.
// Authenticated may be called multiple times in succession between Connect and Connack packets.
//
// An example of identity authentication based on Scram sha256:
//
//	import "github.com/xdg-go/scram"
//
//	type ScramAuth struct {
//		conv   *scram.ClientConversation
//		method string
//	}
//
//	 func (s *ScramAuth) New(username, password string, hash crypto.Hash, methodName string) ([]byte, error) {
//	 	c, err := scram.HashGeneratorFcn(hash.New).NewClient(username, password, "")
//	 	if err != nil {
//	 		return nil, err
//	 	}
//		s.conv = c.NewConversation()
//		s.method = methodName
//		resp, err := s.conv.Step("")
//		return []byte(resp), err
//	}
//
//	func (s *ScramAuth) Authenticate(clientId string, auth *packets.Auth) (*packets.Auth, error) {
//		data, err := s.conv.Step(string(auth.Properties.AuthData))
//		if err != nil {
//			return nil, err
//		}
//		resp := packets.NewAuthWith(packets.ContinueAuthentication, s.method, []byte(data))
//		return resp, nil
//	}
//
//	func (s *ScramAuth) Authenticated(clientId string, ack *packets.Connack) error {
//		if ack.Properties == nil || ack.Properties.AuthMethod == "" {
//			return nil
//		}
//		_, err := s.conv.Step(string(ack.Properties.AuthData))
//		return err
//	}
//	 func main(){
//		user:="user"
//		password:="password"
//		method := "SCRAM-SHA-256"
//		auth := &ScramAuth{}
//		data, err := auth.New(user, password, crypto.SHA256, method)
//		if err != nil {
//			panic(err)
//		}
//		client := mqtt.NewClient(&mqtt.ConnDialer{
//			Address: "tcp://127.0.0.1:1883",
//		}, mqtt.ClientConfig{ Auther: auth, })
//		pkt := packets.NewConnect("client_id", "", nil)
//		pkt.Properties = &packets.ConnProperties{
//			AuthMethod: method,
//			AuthData:   data,
//		}
//		client.Connect(context.Background(), pkt)
//	}
//
// Note: It is not recommended to enable disconnection retry and enhanced authentication at the same time,
// because unexpected problems may occur during authentication processes such as scram.
type Auther interface {
	Authenticate(clientId string, auth *packets.Auth) (*packets.Auth, error)
	Authenticated(clientId string, ack *packets.Connack) error
}
