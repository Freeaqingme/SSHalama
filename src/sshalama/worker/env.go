// Sshalama
//
// Copyright 2016-2017 Dolf Schimmel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	//"bytes"
	"bytes"
	"fmt"
	"strings"
)

type env struct {
	values map[string]string
}

func NewEnv() *env {
	return &env{
		values: make(map[string]string, 0),
	}
}

// The env payload's first 4 bytes denote the
// length of the environment variable. We assume
// a max length of 255 chars, so only use the last
// byte, and reject it if the first 3 bytes are
// non-zero.
func (e *env) AddPayload(payload []byte) bool {
	key, err := e.getPayloadString(payload)
	if err != nil {
		return false
	}

	value, err := e.getPayloadString(payload[4+len(key):])
	if err != nil {
		return false
	}

	e.values[string(key)] = string(value)
	return true
}

func (e *env) GetAsSlice(ignore []string) []string {
	out := make([]string, 0)
ValueLoop:
	for k, v := range e.values {
		for _, ignoreItem := range ignore {
			if strings.EqualFold(ignoreItem, k) {
				continue ValueLoop
			}
		}

		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}

	return out
}

func (e *env) getPayloadString(payload []byte) ([]byte, error) {
	if len(payload) < 4 {
		return []byte{}, fmt.Errorf("payload is smaller than 4 characters")
	}

	if !bytes.Equal(payload[0:3], []byte{0, 0, 0}) {
		return []byte{}, fmt.Errorf("string is longer than 255 characters")
	}

	stringLength := int(payload[3])
	if len(payload) < (4 + stringLength) {
		return []byte{}, fmt.Errorf("supplied length does not match payload length")
	}

	return payload[4 : 4+stringLength], nil
}
