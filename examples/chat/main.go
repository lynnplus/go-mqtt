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

func main() {

	client := mqtt.NewClient(&mqtt.ConnDialer{
		Address: "tcp://127.0.0.1:1883",
		Timeout: 10 * time.Second,
	}, mqtt.ClientConfig{})

	pkt := packets.NewConnect("test_client", "lynn", nil)

	ack, err := client.Connect(context.Background(), pkt)
	if err != nil {
		panic(err)
	}
	fmt.Println(ack)

	stdin := bufio.NewReader(os.Stdin)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	topic := "test/chat"

	sub := packets.NewSubscribe(topic)

	suback, err := client.Subscribe(context.Background(), sub)
	if err != nil {
		panic(err)
	}

	fmt.Println(suback, *suback.Properties)

	for {

		select {
		case <-sig:
			return
		default:
		}

		message, err := stdin.ReadString('\n')
		if err == io.EOF {
			os.Exit(0)
		}

		pb := packets.NewPublish(topic, []byte(message))
		props := &packets.PubProperties{}
		props.User.Add("nickname", "go-mqtt")
		pb.Properties = props

		if err := client.Publish(context.Background(), pb); err != nil {
			fmt.Println("publish err:", err)
		}
	}

}
