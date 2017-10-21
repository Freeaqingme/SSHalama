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
	"golang.org/x/crypto/ssh"
	"log"
	"strconv"
	"time"
)

type proxyMode struct {
}

func (s *proxyMode) getPasswordCallback(dstClient **ssh.Client) func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	holdOff := func() {
		//duration, _ := time.ParseDuration(Config.Ssh_Reverse_Proxy.Auth_Error_Delay)
		time.Sleep(5 * time.Second)
	}

	return func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		host, port := "localhost", "22" // TODO backend.GetServerForUser(c.User())
		if host == "" {
			log.Printf(fmt.Sprintf("Unknown user %q on %s", c.User(), c.RemoteAddr().String()))
			holdOff()
			return nil, fmt.Errorf("Unknown user %q on %s", c.User(), c.RemoteAddr().String())
		}

		var err error
		*dstClient, err = s.getDestinationServerSshClient(host, port, c.User(), string(pass))
		if err != nil {
			log.Printf("Could not authorize %q on %s: %s",
				c.User(), c.RemoteAddr().String(), err)
			holdOff()
			return nil, fmt.Errorf("Could not authorize %q on %s: %s",
				c.User(), c.RemoteAddr().String(), err)
		}
		return nil, nil
	}
}

func (s *proxyMode) getDestinationServerSshClient(host, portStr, user, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}
	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		panic("Port must be an integer")
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect: " + err.Error())
	}

	return conn, nil
}
