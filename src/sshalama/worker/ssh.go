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
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/Freeaqingme/go-proxyproto"
	"golang.org/x/crypto/ssh"
)

// Snake Eyes equivalent. It's only used on the socketpair()
// between the master and worker process.
var privateBytes = []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA3NBahi99AF8PqS6TV4m2A/lUsUB4tIaJPZ6CwKsICzZGgAD8
PNsrn20w8h6q1CF/xDJaxYr8iQ3y1ueixlc+/6LMv3bhF76e4e4FfcPE8wolPxvY
ohEef6so8p32LFg36Y7enkM7H/WfCuNg9WzHrK31KvWiHwG+QhqJldj94bVmt60u
kYxoXaBCTfU97UfN03JS8y3Q13Iq+GkuSuoWjXUtyvJ5rp47ySvWR1m6tKKkTvPT
IICAWFZ/Fvq/A0ME66Zq/bshIplm2d0pysFMjyUUftwTyUQEmIU5Sxrs/AbRMMbJ
Rpc4fX50ntE3oeuBdI+Hyd5pfK0DlLOv+cf3iQIDAQABAoIBAQCwfn0MqiJwzIq5
AHhWzMTGcmDmeJDCQpKpxOvf0hTQ2WYKZD846Tn56Q3pSOfkPI5iJJl3MfteFN8Y
NPdfL1c0f0zGcN/D2eIm1dhfyL3AQUi6I6jJCYPmKcnF+spMcYrnTQHVYAl/JxUj
X9Ec+gCznivLVaBqxjrrnUiBlHqBD8BaKzxNmnhhjHhaOd2h7zondA17twPdz93z
EfyJI9OVcggZuTrJgNRKsoJNRdS1CN6HbmrxerKG4e8EbvD1Wj1WkTgbKIr5P/2l
rEtkk/TGTehGd0eualndS2MuprPrB4/Q4IqHMVAhCRIM5NaqAe0vBu88CnR/WBoZ
GnTFFX19AoGBAPCkjbfO6mDGM7E24L1IeWdWvRbrsm43kbETF/4XGIdbDF3CFQlW
RmbU/cXKFszrzDLwtykMBHvitHYKff/sCh3WFEWwqMXgJN3JQY09nXQDjJxdvE1o
Ytg1JEQ9a6BLHEkJVxylNiNDZjFkjZABEMR29q+/PmEC6kWJOveB0CNDAoGBAOrn
2Y1JHyOW6WNgvLOveHXoTknWHDidstJ79PTRoWYFnIZqtSgyyunyN1/ShpSwVL3o
emVqwgKavkI8UT6fYBp8zslFRMzgH1bkDri2GNGjebUGhqHyXOr2MLlrzP2ekVQm
AeiTnjQ+WHa6U261kWHYxbX/ivFWFkPrp2mRpf9DAoGAYviquLBHQSoDVJ1nbTID
jHbmKikiJ6Z/Kz7ZHU3Obs0Jlv4dvMtZBS4QeWqWWg2Y3FKYYi9pILKq2emSzND9
kCveBpOTtl5rizQc28Q9n9td12nN6mBGVvn0QoSoYTLDHV7UDxn73CD6RNJATrvB
c6wh5UJYm3mhdJvuPqGLQxUCgYEAkNiT6i3LeKuGkBPHZ9jsI3AyTg8rabG74VQz
8H4O0pTlNnE38WiYfHcxs/FhsO+l4VAnoL+aj/aRGNCOnFmz7cFF1Q/UY6xTRsXr
WfRXC3WNB5XVkKicqPlThBI33a9YF5Y0GRBlPfuvms47wglNcxMyno3LRBL8Obdm
jI8V13cCgYAO+iSUx0EHiWCGbUNoFipxkk6HAv2YPzduM2C3rtbBW2EFcoOfMZ7E
NrpbDDuK0dzeGR1bUvz3pcXM2iC8uHxf54C2MJJ7HXzfHNwfwJrDIttOkDyZzIg2
ZrdaW4Nf7X7b+tvksK7WqmVsODYZtXecwuMIbhrqo9Gtc5W1War5Rw==
-----END RSA PRIVATE KEY-----
`)

// Based on example server code from golang.org/x/crypto/ssh and server_standalone
func (w *worker) handleConn(nConn net.Conn) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			w.setProcessName(c)

			// Credentials have been checked in the parent process
			return nil, nil
		},
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key", err)
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

	// Sessions have out-of-band requests such as "shell",
	// "pty-req" and "env".  Here we handle only the
	// "subsystem" request.
	go func(in <-chan *ssh.Request) {
		for req := range in {
			fmt.Fprintf(debugStream, "Request: %v - %s\n", req.Type, string(req.Payload))
			ok := false
			switch req.Type {
			case "subsystem":
				fmt.Fprintf(debugStream, "Subsystem: %s\n", req.Payload[4:])
				if string(req.Payload[4:]) == "sftp" {
					go w.handleSftp(channel)
					ok = true
				}
			}
			fmt.Fprintf(debugStream, " - accepted: %v\n", ok)
			req.Reply(ok, nil)
		}
	}(requests)
}
