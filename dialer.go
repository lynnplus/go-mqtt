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
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strings"
	"time"
)

type Dialer interface {
	Dial(ctx context.Context, timeout time.Duration) (net.Conn, error)
}

type ConnDialer struct {
	Address   string
	TLSConfig *tls.Config
}

func (c *ConnDialer) schemeToNetwork(scheme string) string {
	switch scheme {
	case "tcp", "mqtt", "":
		return "tcp"
	case "tls", "ssl", "mqtts", "tcps":
		return "tls"
	}
	return scheme
}

func (c *ConnDialer) Dial(ctx context.Context, timeout time.Duration) (net.Conn, error) {
	u, err := url.Parse(c.Address)
	if err != nil {
		return nil, err
	}
	scheme := strings.ToLower(u.Scheme)
	scheme = c.schemeToNetwork(scheme)

	d := net.Dialer{Timeout: timeout}
	switch scheme {
	case "tls":
		td := tls.Dialer{NetDialer: &d, Config: c.TLSConfig}
		return td.DialContext(ctx, "tcp", u.Host)
	default: //tcp or other
		return d.DialContext(ctx, scheme, u.Host)
	}
}
