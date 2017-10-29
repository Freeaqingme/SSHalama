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
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/Freeaqingme/go-proxyproto"
	"golang.org/x/crypto/ssh"
)

// Based on example server code from:
// - golang.org/x/crypto/ssh and server_standalone
// - github.com/Scalingo/go-ssh-examples/blob/master/server_complex.go
func (w *worker) handleConn(nConn net.Conn) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			w.setProcessName(c)

			// Credentials have been checked in the parent process
			return nil, nil
		},
	}

	privateRsa, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatal("Cannot generate Rsa Key: ", err)
	}
	private, err := ssh.NewSignerFromKey(privateRsa)
	if err != nil {
		log.Fatal("Cannot use generated Rsa Key: ", err)
	}

	config.AddHostKey(private)

	nConn = proxyproto.NewConn(nConn, 1*time.Second)
	_, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Fatal("failed to handshake", err)
	}

	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		w.handleChannel(newChannel)
	}
}

func (w *worker) handleChannel(newChannel ssh.NewChannel) {
	debugStream := os.Stdout
	// Channels have a type, depending on the application level
	// protocol intended. In the case of an SFTP session, this is "subsystem"
	// with a payload string of "<length=4>sftp"
	fmt.Fprintf(debugStream, "Incoming channel: %s\n", newChannel.ChannelType())
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		fmt.Fprintf(debugStream, "Unknown channel type: %s\n", newChannel.ChannelType())
		return
	}
	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Fatal("could not accept channel.", err)
	}
	fmt.Fprintf(debugStream, "Channel accepted\n")

	pty := NewPty()

	// Sessions have out-of-band requests such as "shell",
	// "pty-req" and "env".
	go func(in <-chan *ssh.Request) {
		for req := range in {
			ok := false
			switch req.Type {
			case "env":
				ok = pty.AddEnvPayload(req.Payload)
			case "pty-req":
				ok = true
				pty.SetDimensionsFromPayload(req.Payload)
			case "shell":
				// We don't accept any commands (Payload),
				// only the default shell.
				if len(req.Payload) == 0 {
					go w.handleShell(channel, pty)
					ok = true
				}
			case "subsystem":
				fmt.Fprintf(debugStream, "Subsystem: %s\n", req.Payload[4:])
				if string(req.Payload[4:]) == "sftp" {
					go w.handleSftp(channel)
					ok = true
				}
			case "window-change":
				pty.SetDimensions(pty.ParseDimensions(req.Payload))
				continue //no response
			}

			if !ok {
				fmt.Fprintf(debugStream, "Could not accept Request: %v - %s\n", req.Type, string(req.Payload))
			}
			req.Reply(ok, nil)
		}
	}(requests)
}
