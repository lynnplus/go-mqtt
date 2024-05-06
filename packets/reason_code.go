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
	"errors"
	"fmt"
)

// ReasonCode is a single-byte unsigned value used to indicate the result of the operation.
// The reason code less than 0x80 indicates that the result of the operation is successful.
// Under normal circumstances, the reason code value returned by the operation is 0.
// If the reason code returned is greater than or equal to 0x80, the operation has failed.
//
// Reason codes can provide detailed operational feedback to clients and servers.
// For more information about these reason codes, see the markdown "[REASON CODE]"
//
// [REASON CODE]: https://github.com/lynnplus/go-mqtt/blob/master/packets/REASON_CODE.md
type ReasonCode byte

// IsError returns whether the reason code is an error code.
func (c ReasonCode) IsError() bool { return c >= 0x80 }

// readable string for reason code,copy from "REASON_CODE.md"
//
// [REASON_CODE.md]: https://github.com/lynnplus/go-mqtt/blob/master/packets/REASON_CODE.md
func (c ReasonCode) String() string {
	switch c {
	case 0:
		return "Success"
	case GrantedQoS1:
		return "Granted QoS 1"
	case GrantedQoS2:
		return "Granted QoS 2"
	case DisconnectWithWillMessage:
		return "Disconnect with Will Message"
	case NoMatchingSubscribers:
		return "No matching subscribers"
	case NoSubscriptionExisted:
		return "No subscription existed"
	case ContinueAuthentication:
		return "Continue authentication"
	case ReAuthenticate:
		return "Re-authenticate"
	case UnspecifiedError:
		return "Unspecified error"
	case MalformedPacket:
		return "Malformed Packet"
	case ProtocolError:
		return "Protocol Error"
	case ImplementationSpecificError:
		return "Implementation specific error"
	case UnsupportedProtocolVersion:
		return "Unsupported Protocol Version"
	case ClientIdentifierNotValid:
		return "Client Identifier not valid"
	case BadUsernameOrPassword:
		return "Bad User Name or Password"
	case NotAuthorized:
		return "Not authorized"
	case ServerUnavailable:
		return "Server unavailable"
	case ServerBusy:
		return "Server busy"
	case Banned:
		return "Banned"
	case ServerShuttingDown:
		return "Server shutting down"
	case BadAuthenticationMethod:
		return "Bad authentication method"
	case KeepAliveTimeout:
		return "Keep Alive timeout"
	case SessionTakenOver:
		return "Session taken over"
	case TopicFilterInvalid:
		return "Topic Filter invalid"
	case TopicNameInvalid:
		return "Topic Name invalid"
	case PacketIdentifierInUse:
		return "Packet Identifier in use"
	case PacketIdentifierNotFound:
		return "Packet Identifier not found"
	case ReceiveMaximumExceeded:
		return "Receive Maximum exceeded"
	case TopicAliasInvalid:
		return "Topic Alias invalid"
	case PacketTooLarge:
		return "Packet too large"
	case MessageRateTooHigh:
		return "Message rate too high"
	case QuotaExceeded:
		return "Quota exceeded"
	case AdministrativeAction:
		return "Administrative action"
	case PayloadFormatInvalid:
		return "Payload format invalid"
	case RetainNotSupported:
		return "Retain not supported"
	case QoSNotSupported:
		return "QoS not supported"
	case UseAnotherServer:
		return "Use another server"
	case ServerMoved:
		return "Server moved"
	case SharedSubscriptionsNotSupported:
		return "Shared Subscriptions not supported"
	case ConnectionRateExceeded:
		return "Connection rate exceeded"
	case MaximumConnectTime:
		return "Maximum connect time"
	case SubscriptionIdentifiersNotSupported:
		return "Subscription Identifiers not supported"
	case WildcardSubscriptionsNotSupported:
		return "Wildcard Subscriptions not supported"
	default:
		return fmt.Sprintf("Unknown reason code: 0x%x", byte(c))
	}
}

const (
	Success                             ReasonCode = 0x00 // Success
	NormalDisconnection                 ReasonCode = 0x00 // Normal disconnection
	GrantedQoS0                         ReasonCode = 0x00 // Granted QoS 0
	GrantedQoS1                         ReasonCode = 0x01 // Granted QoS 1
	GrantedQoS2                         ReasonCode = 0x02 // Granted QoS 2
	DisconnectWithWillMessage           ReasonCode = 0x04 // Disconnect with Will Message
	NoMatchingSubscribers               ReasonCode = 0x10 // No matching subscribers
	NoSubscriptionExisted               ReasonCode = 0x11 // No subscription existed
	ContinueAuthentication              ReasonCode = 0x18 // Continue authentication
	ReAuthenticate                      ReasonCode = 0x19 // Re-authenticate
	UnspecifiedError                    ReasonCode = 0x80 // Unspecified error
	MalformedPacket                     ReasonCode = 0x81 // Malformed Packet
	ProtocolError                       ReasonCode = 0x82 // Protocol Error
	ImplementationSpecificError         ReasonCode = 0x83 // Implementation specific error
	UnsupportedProtocolVersion          ReasonCode = 0x84 // Unsupported Protocol Version
	ClientIdentifierNotValid            ReasonCode = 0x85 // Client Identifier not valid
	BadUsernameOrPassword               ReasonCode = 0x86 // Bad Username or Password
	NotAuthorized                       ReasonCode = 0x87 // Not authorized
	ServerUnavailable                   ReasonCode = 0x88 // Server unavailable
	ServerBusy                          ReasonCode = 0x89 // Server busy
	Banned                              ReasonCode = 0x8A // Banned
	ServerShuttingDown                  ReasonCode = 0x8B // Server shutting down
	BadAuthenticationMethod             ReasonCode = 0x8C // Bad authentication method
	KeepAliveTimeout                    ReasonCode = 0x8D // Keep Alive timeout
	SessionTakenOver                    ReasonCode = 0x8E // Session taken over
	TopicFilterInvalid                  ReasonCode = 0x8F // Topic Filter invalid
	TopicNameInvalid                    ReasonCode = 0x90 // Topic Name invalid
	PacketIdentifierInUse               ReasonCode = 0x91 // Packet Identifier in use
	PacketIdentifierNotFound            ReasonCode = 0x92 // Packet Identifier not found
	ReceiveMaximumExceeded              ReasonCode = 0x93 // Receive Maximum exceeded
	TopicAliasInvalid                   ReasonCode = 0x94 // Topic Alias invalid
	PacketTooLarge                      ReasonCode = 0x95 // Packet too large
	MessageRateTooHigh                  ReasonCode = 0x96 // Message rate too high
	QuotaExceeded                       ReasonCode = 0x97 // Quota exceeded
	AdministrativeAction                ReasonCode = 0x98 // Administrative action
	PayloadFormatInvalid                ReasonCode = 0x99 // Payload format invalid
	RetainNotSupported                  ReasonCode = 0x9A // Retain not supported
	QoSNotSupported                     ReasonCode = 0x9B // QoS not supported
	UseAnotherServer                    ReasonCode = 0x9C // Use another server
	ServerMoved                         ReasonCode = 0x9D // Server moved
	SharedSubscriptionsNotSupported     ReasonCode = 0x9E // Shared Subscriptions not supported
	ConnectionRateExceeded              ReasonCode = 0x9F // Connection rate exceeded
	MaximumConnectTime                  ReasonCode = 0xA0 // Maximum connect time
	SubscriptionIdentifiersNotSupported ReasonCode = 0xA1 // Subscription Identifiers not supported
	WildcardSubscriptionsNotSupported   ReasonCode = 0xA2 // Wildcard Subscriptions not supported
)

type reasonCodeError struct {
	code    ReasonCode
	message string
}

func (r *reasonCodeError) Error() string {
	if r.message != "" {
		return fmt.Sprintf("mqtt 0x%x err,%s", byte(r.code), r.message)
	}
	//TODO a more readable message should be used
	return fmt.Sprintf("mqtt 0x%x err,%s", byte(r.code), r.code)
}

func (r *reasonCodeError) Is(target error) bool {
	var x *reasonCodeError
	if errors.As(target, &x) {
		return x.code == r.code
	}
	return false
}

var (
	ErrUnsupportedValueOnVersion = NewReasonCodeError(ProtocolError, "Packet content field not supported by current version")
	ErrUnsupportedPropSetup      = NewReasonCodeError(ProtocolError, "The current version does not support packet properties")
	ErrInvalidPktFlags           = NewReasonCodeError(MalformedPacket, "Invalid packet control flags")
)

// NewReasonCodeError returns a new error based on the reason code,
// If the message parameter is a non-empty string, the new error will use this message,
// otherwise the built-in default message will be used.
// for the default reason code string, see ReasonCode.String()
func NewReasonCodeError(code ReasonCode, message string) error {
	if !code.IsError() {
		panic(fmt.Errorf("reason code 0x%x is not an error", byte(code)))
	}
	return &reasonCodeError{code, message}
}
