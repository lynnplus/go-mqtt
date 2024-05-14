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
	"testing"
)

func BenchmarkDefaultRouter_Register(b *testing.B) {
	router := NewDefaultRouter()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			router.Register("a/b/c/d/e/f/g/h/i/j/k", nil)
		}
	})
}

func BenchmarkDefaultRouter_Route(b *testing.B) {
	router := NewDefaultRouter()
	router.Register("a/b/c/d/e/f/g/h/i/j/k", nil)
	ctx := &pubMsgContext{
		packet: &packets.Publish{Topic: "a/b/c/d/e/f/g/h/i/j/k"},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			router.Route(ctx)
		}
	})
}
