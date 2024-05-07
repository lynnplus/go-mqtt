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

import "io"

// A UserProperties represents the key-value pairs in a user properties for Packet.
type UserProperties map[string][]string

// Get gets the first value associated with the given key.
// If there is no value associated with the key, Get returns "".
// It is case-sensitive.
func (u *UserProperties) Get(key string) string {
	vs, ok := (*u)[key]
	if !ok || len(vs) == 0 {
		return ""
	}
	return vs[0]
}

// Add adds a key,value pair to UserProperties, appending it to any existing values associated with the key.
// The key is case-sensitive.
func (u *UserProperties) Add(key, value string) *UserProperties {
	(*u)[key] = append((*u)[key], value)
	return u
}

// AddSlice adds key,values to UserProperties
func (u *UserProperties) AddSlice(key string, value ...string) *UserProperties {
	(*u)[key] = append((*u)[key], value...)
	return u
}

// AddMap adds a map to UserProperties
func (u *UserProperties) AddMap(v map[string]string) *UserProperties {
	for k, s := range v {
		u.Add(k, s)
	}
	return u
}

// Remove delete all data associated with key
func (u *UserProperties) Remove(key string) *UserProperties {
	delete(*u, key)
	return u
}

// Copy Returns a deep copy of the UserProperties
func (u *UserProperties) Copy() *UserProperties {
	nm := make(map[string][]string, len(*u))
	for k, v := range *u {
		s := make([]string, len(v))
		copy(s, v)
		nm[k] = s
	}
	return u
}

// WriteTo writes data to w according to the protocol format
func (u *UserProperties) WriteTo(w io.Writer) (int64, error) {
	if len(*u) == 0 {
		return 0, nil
	}
	id := PropUserProperty
	n := int64(0)
	var err error
	for k, vs := range *u {
		for _, v := range vs {
			if err = unsafeWriteByte(w, (*byte)(&id)); err != nil {
				return n, err
			}
			n += 1
			if err = unsafeWriteString(w, &k); err != nil {
				return n, err
			}
			n += int64(len(k)) + 2
			if err = unsafeWriteString(w, &v); err != nil {
				return n, err
			}
			n += int64(len(v)) + 2
		}
	}
	return n, err
}
