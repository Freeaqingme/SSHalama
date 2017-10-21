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
package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

func (s *Server) listen(bind string, proxyProtocol bool) error {
	listener, err := net.Listen("tcp", bind)

	if err != nil {
		return err
	}
	log.Printf("Now listening on %s", bind)

	go func() {
		for {
			lnConn, err := listener.Accept()
			if err != nil {
				log.Fatal("failed to accept incoming connection: %s", err.Error())
			}
			go s.handleIncomingConn(lnConn)
		}
	}()

	return nil
}

func (s *Server) handleIncomingConn(lnConn net.Conn) {
	// TODO: Recover()

	log.Printf("Received connection from %s", lnConn.RemoteAddr())

	var sClient *ssh.Client
	dstServerConfig := s.getDestinationServerSshConfig(&sClient)
	dstConn, dstChans, dstReqs, err := ssh.NewServerConn(lnConn, dstServerConfig)
	if err != nil {
		log.Printf("Could not establish connection with " + lnConn.RemoteAddr().String() + ": " + err.Error())
		return
	}
	defer dstConn.Close()
	defer sClient.Close()

	go ssh.DiscardRequests(dstReqs)

	for newChannel := range dstChans {
		s.handleChannel(newChannel, sClient)
	}

	log.Printf("Lost connection with %s", lnConn.RemoteAddr())
}

func (s *Server) handleChannel(newChannel ssh.NewChannel, rClient *ssh.Client) {
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type: "+newChannel.ChannelType())
		return
	}
	psChannel, psRequests, err := newChannel.Accept()
	if err != nil {
		panic("could not accept channel.")
	}

	sChannel, sRequests, err := rClient.OpenChannel(newChannel.ChannelType(), nil)
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}

	go s.pipeRequests(psChannel, sChannel, psRequests, sRequests)
	time.Sleep(50 * time.Millisecond)
	go s.pipe(sChannel, psChannel)
	go s.pipe(psChannel, sChannel)
}

func (s *Server) pipe(dst, src ssh.Channel) {
	_, err := io.Copy(dst, src)
	if err != nil {
		fmt.Println(err.Error())
	}

	dst.CloseWrite()
}

func (s *Server) getDestinationServerSshConfig(rClient **ssh.Client) *ssh.ServerConfig {
	callback := func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		mode, err := s.getModeForUser(c.User())
		if err != nil {
			return nil, err
		}

		return mode.getPasswordCallback(rClient)(c, pass)
	}

	config := &ssh.ServerConfig{
		ServerVersion:    "SSH-2.0-SshReverseProxy",
		PasswordCallback: callback,
	}

	privateBytes, err := ioutil.ReadFile("/home/dolf/Projects/Sshalama/id_rsa")
	if err != nil {
		panic("Failed to load private key")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Failed to parse private key")
	}

	config.AddHostKey(private)
	return config
}

func (s *Server) pipeRequests(psChannel, sChannel ssh.Channel, psRequests, sRequests <-chan *ssh.Request) {
	defer func() {
		return
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	defer sChannel.Close()
	defer psChannel.Close()

	for {
		select {
		case lRequest, ok := <-psRequests:
			if !ok {
				return
			}
			if err := s.forwardRequest(lRequest, sChannel); err != nil {
				fmt.Println("Error: " + err.Error())
				continue
			}
		case rRequest, ok := <-sRequests:
			if !ok {
				return
			}
			if err := s.forwardRequest(rRequest, psChannel); err != nil {
				fmt.Println("Error: " + err.Error())
				continue
			}
		}
	}
}

func (s *Server) forwardRequest(req *ssh.Request, channel ssh.Channel) error {
	reply, err := channel.SendRequest(req.Type, req.WantReply, req.Payload)
	if err != nil {
		return err
	}
	if req.WantReply {
		req.Reply(reply, nil)
	}

	return nil
}
