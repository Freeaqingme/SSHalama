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
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/prep/socketpair"
	"golang.org/x/crypto/ssh"
)

type localMode struct {
}

func (s *localMode) getPasswordCallback(dstClient **ssh.Client) func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	holdOff := func() {
		//duration, _ := time.ParseDuration(Config.Ssh_Reverse_Proxy.Auth_Error_Delay)
		time.Sleep(5 * time.Second)
	}

	return func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		var err error
		*dstClient, err = s.getWorkerSshClient(c, string(pass))
		if err != nil {
			log.Printf("Could not authorize %q on %s: %s", c.User(), c.RemoteAddr().String(), err)
			holdOff()
			return nil, fmt.Errorf("Could not authorize %q on %s: %s", c.User(), c.RemoteAddr().String(), err)
		}
		return nil, nil
	}
}

func (s *localMode) getWorkerSshClient(connData ssh.ConnMetadata, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            connData.User(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}

	conn, err := s.spawnWorker()
	if err != nil {
		return nil, err
	}

	hdr, err := s.getProxyProtoHeader(connData.RemoteAddr(), connData.LocalAddr())
	if err != nil {
		panic("TODO, error handling: " + err.Error())
	}
	_, err = fmt.Fprint(conn, hdr)
	if err != nil {
		panic("Todo: error handling")
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, "Unix", config)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}

func (s *localMode) spawnWorker() (net.Conn, error) {
	serverConn, workerConn, err := socketpair.New("unix")
	if err != nil {
		return nil, err
	}

	workerConnFd, err := workerConn.(*net.UnixConn).File()
	if err != nil {
		return nil, err
	}

	go func() {
		log.Print("Starting worker")
		cmd := exec.Command("/home/dolf/Projects/Sshalama/bin/sshalama-worker")
		cmd.ExtraFiles = []*os.File{workerConnFd}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout

		if err := cmd.Start(); err != nil {
			panic(err)
		}
		cmd.Wait()
	}()

	return serverConn, nil
}

// Originally derived from https://github.com/nabeken/mikoi
// Released under BSD-3 license, by Tanabe Ken-ichi
func (s *localMode) getProxyProtoHeader(remoteAddr net.Addr, localAddr net.Addr) (string, error) {
	saddr, sport, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		return "", err
	}

	daddr, dport, err := net.SplitHostPort(localAddr.String())
	if err != nil {
		return "", err
	}

	raddr, ok := remoteAddr.(*net.TCPAddr)
	if !ok {
		return "", errors.New("Cannot proxy protocol other than TCP4 or TCP6")
	}

	var tcpStr string
	if rip4 := raddr.IP.To4(); len(rip4) == net.IPv4len {
		tcpStr = "TCP4"
	} else if len(raddr.IP) == net.IPv6len {
		tcpStr = "TCP6"
	} else {
		return "", errors.New("Unrecognized protocol type")
	}

	hdr := fmt.Sprintf("PROXY %s %s %s %s %s\r\n", tcpStr, saddr, daddr, sport, dport)
	return hdr, err
}
