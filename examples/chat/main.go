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
	"bufio"
	"context"
	"fmt"
	"github.com/lynnplus/go-mqtt"
	"github.com/lynnplus/go-mqtt/packets"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type fmtLogger struct{}

func (l *fmtLogger) Debug(format string, args ...any) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ":[D] " + fmt.Sprintf(format, args...))
}

func (l *fmtLogger) Error(format string, args ...any) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ":[E] " + fmt.Sprintf(format, args...))
}

var logger = &fmtLogger{}

func onConnected(cl *mqtt.Client, ack *packets.Connack) {
	logger.Debug("callback onConnected")
}

func onConnectionLost(client *mqtt.Client, err error) {
	logger.Debug("callback onConnectionLost")
}

func onConnectFailed(client *mqtt.Client, err error) {
	logger.Debug("callback onConnectFailed")
}

func onServerDisconnect(client *mqtt.Client, pkt *packets.Disconnect) {
	logger.Debug("callback onServerDisconnect")
}
func onClientError(client *mqtt.Client, err error) {
	logger.Debug("callback onClientError")
}

func onReceivedChatMsg(ctx mqtt.Context) {
	logger.Debug("callback onReceivedChatMsg: %v", ctx.Topic())
}

func main() {
	router := mqtt.NewDefaultRouter()
	config := &mqtt.ClientConfig{
		Logger: logger,
		ClientListener: mqtt.ClientListener{
			OnConnected:        onConnected,
			OnClientError:      onClientError,
			OnServerDisconnect: onServerDisconnect,
			OnConnectFailed:    onConnectFailed,
			OnConnectionLost:   onConnectionLost,
		},
		ReConnector: mqtt.NewAutoReConnector(),
		Router:      router,
	}

	client := mqtt.NewClient(&mqtt.ConnDialer{
		Address: "tcp://127.0.0.1:1883",
	}, *config)

	pkt := packets.NewConnect("client_id", "", nil)

	ack, err := client.Connect(context.Background(), pkt)
	if err != nil {
		panic(err)
	}
	if ack.ReasonCode != 0 {
		panic(packets.NewReasonCodeError(ack.ReasonCode, ""))
	}

	stdin := bufio.NewReader(os.Stdin)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	topic := "test/chat"

	regCancel := router.RegisterCancelable(topic, onReceivedChatMsg)
	defer regCancel()

	sub := packets.NewSubscribe(topic, "test/empty")
	suback, err := client.Subscribe(context.Background(), sub)
	if err != nil {
		panic(err)
	}

	logger.Debug("chat-client suback: %v", suback)
	go func() {
		time.Sleep(2 * time.Second)
		unsub := packets.NewUnsubscribe("test/empty")
		unsuback, err := client.Unsubscribe(context.Background(), unsub)
		if err != nil {
			panic(err)
		}
		logger.Debug("chat-client unsuback: %v", *unsuback)
	}()

	for {
		select {
		case <-sig:
			goto loopEnd
		default:
		}

		message, err := stdin.ReadString('\n')
		if err == io.EOF {
			os.Exit(0)
		}

		pb := packets.NewPublish(topic, []byte(message))
		pb.QoS = 1
		props := &packets.PubProperties{}
		props.User.Add("nickname", "go-mqtt")
		pb.Properties = props

		if err := client.PublishNR(context.Background(), pb); err != nil {
			logger.Error("chat-client publish err: %v", err)
		}
	}
loopEnd:

	_ = client.Disconnect()
}
