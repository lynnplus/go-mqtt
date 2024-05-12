# go-mqtt

[![Go Reference](https://pkg.go.dev/badge/github.com/lynnplus/go-mqtt.svg)](https://pkg.go.dev/github.com/lynnplus/go-mqtt)
![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/lynnplus/go-mqtt)
![GitHub tag (with filter)](https://img.shields.io/github/v/tag/lynnplus/go-mqtt)
[![Go Report Card](https://goreportcard.com/badge/github.com/lynnplus/go-mqtt)](https://goreportcard.com/report/github.com/lynnplus/go-mqtt)
[![GitHub](https://img.shields.io/github/license/lynnplus/go-mqtt)](https://github.com/lynnplus/go-mqtt/blob/master/LICENSE)


go-mqtt is a mqtt golang client that implements the mqttv3 and mqttv5 protocols

## Done
- basic client
- packet reading and writing
- publish qos 0
- subscribe and unsubscribe
- re-connector
## TODO
- mqttv3 check
- qos 1 and 2
- prefix tree based message router
- event hook

## Links

[MQTT-v5.0 oasis doc](https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html)

[MQTT-v3.1.1 oasis doc](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)

## How to use

get package:
```shell
go get github.com/lynnplus/go-mqtt
```
connect to mqtt broker
```go
package main

import (
	"github.com/lynnplus/go-mqtt"
	"github.com/lynnplus/go-mqtt/packets"
)

func main() {
	client := mqtt.NewClient(&mqtt.ConnDialer{
		Address: "tcp://127.0.0.1:1883",
	}, mqtt.ClientConfig{})

	pkt := packets.NewConnect("client_id", "username", []byte("password"))
	ack, err := client.Connect(context.Background(), pkt)
	if err != nil {
		panic(err)
	}
	if ack.ReasonCode != 0 {
		panic(packets.NewReasonCodeError(ack.ReasonCode, ""))
	}
	//do something
	_ = client.Disconnect()
}
```

The package provides two connection methods,
synchronous and asynchronous. When asynchronous,
the response result can only be obtained in the callback.

Asynchronous call connection:

```go
package main

func main() {
	client := mqtt.NewClient(&mqtt.ConnDialer{
		Address: "tcp://127.0.0.1:1883",
	}, mqtt.ClientConfig{
		OnConnected: func(c *mqtt.Client, ack *packets.Connack) {
			fmt.Println(ack.ReasonCode)
		},
	})
	pkt := packets.NewConnect("client_id", "username", []byte("password"))
	err := client.StartConnect(context.Background(), pkt)
}
```

一个基本的聊天示例请查看 https://github.com/lynnplus/go-mqtt/blob/master/examples/chat/main.go

## Features

### Dialer
The package provides a dialer that implements tcp and tls by default.
If the user needs other connection protocol support, such as websocket,
the Dialer interface provided in the package can be implemented.

### MQTTv5 Enhanced authentication
- scram : [example](https://github.com/lynnplus/go-mqtt/blob/master/auther.go)

### Re-connect

...

### Router

...
