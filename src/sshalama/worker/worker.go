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

import "C"

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/erikdubbelboer/gspt"
	"golang.org/x/crypto/ssh"
)

func main() {}

type worker struct{}

//export Run
func Run() int {
	log.SetPrefix(strconv.Itoa(os.Getpid()) + " ")

	conn, err := net.FileConn(os.NewFile(3, "unix"))
	if err != nil {
		log.Print("failed to accept incoming connection", err)
		return 1
	}

	w := &worker{}
	w.handleConn(conn)

	return 0
}

func (w *worker) setProcessName(c ssh.ConnMetadata) {
	gspt.SetProcTitle(fmt.Sprintf("%s %s@%s", "sshalama-worker", c.User(), c.RemoteAddr()))
}
