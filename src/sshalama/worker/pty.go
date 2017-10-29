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
	"encoding/binary"
	"os"

	"github.com/Freeaqingme/pty"
)

type ptyContainer struct {
	env *env

	width  uint32
	height uint32

	pty *os.File
	tty *os.File
}

func NewPty() *ptyContainer {
	return &ptyContainer{
		env: NewEnv(),
	}
}

func (p *ptyContainer) Open() error {
	var err error

	p.pty, p.tty, err = pty.Open()
	if err != nil {
		return err
	}

	p.setWindowSize()
	return nil
}

func (p *ptyContainer) AddEnvPayload(payload []byte) bool {
	return p.env.AddPayload(payload)
}

func (p *ptyContainer) SetDimensions(width, height uint32) {
	p.width = width
	p.height = height

	p.setWindowSize()
}

func (p *ptyContainer) SetDimensionsFromPayload(payload []byte) {
	termLen := payload[3]
	w, h := p.ParseDimensions(payload[termLen+4:])
	p.SetDimensions(w, h)
}

func (p *ptyContainer) ParseDimensions(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])

	return w, h
}

func (p *ptyContainer) setWindowSize() {
	if p.pty == nil {
		return
	}

	pty.SetSize(p.pty, uint(p.height), uint(p.width))
}
