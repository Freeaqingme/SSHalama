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
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/crypto/ssh"
)

func (w *worker) handleShell(channel ssh.Channel, pty *ptyContainer) {
	if err := pty.Open(); err != nil {
		log.Fatalf("could not start pty (%s)", err)
	}

	cmd := exec.Command("/bin/bash")
	cmd.Env = append(pty.env.GetAsSlice([]string{"TERM"}), "TERM=xterm")
	err := PtyRun(cmd, pty.tty)
	if err != nil {
		log.Printf("%s", err)
	}

	var once sync.Once
	close := func() {
		channel.Close()
		log.Printf("session closed")
	}

	// Pipe session to bash and visa-versa
	go func() {
		io.Copy(channel, pty.pty)
		once.Do(close)
	}()

	go func() {
		io.Copy(pty.pty, channel)
		once.Do(close)
	}()
}

// Start assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
func PtyRun(c *exec.Cmd, tty *os.File) (err error) {
	defer tty.Close()
	c.Stdout = tty
	c.Stdin = tty
	c.Stderr = tty
	c.SysProcAttr = &syscall.SysProcAttr{
		Setctty: true,
		Setsid:  true, // TODO: Evaluate me?
	}
	return c.Start()
}
