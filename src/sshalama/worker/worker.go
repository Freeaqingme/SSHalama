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
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"

	"github.com/erikdubbelboer/gspt"
	"golang.org/x/crypto/ssh"
)

/*
#include <sys/mount.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>


__attribute__((constructor))
static void init() {
	if (umount2("/proc", MNT_DETACH) != 0) {
		printf("umount: Unable to drop /proc: %s", strerror(errno));
		exit(1);
	}

	if (setgid(65534) != 0) {
		printf("setgid: Unable to drop group privileges: %s", strerror(errno));
		exit(1);
	}

	if (setuid(65534) != 0) {
		printf("setuid: Unable to drop user privileges: %s", strerror(errno));
		exit(1);
	}

	if (setuid(0) != -1) {
		printf("ERROR: Managed to regain root privileges?");
		exit(1);
	}

}
*/
import "C"

type worker struct{}

func main() {

	log.SetPrefix(strconv.Itoa(os.Getpid()) + " ")

	checkDroppedCaps()

	conn, err := net.FileConn(os.NewFile(3, "unix"))
	if err != nil {
		log.Print("failed to accept incoming connection", err)
		os.Exit(1)
	}

	w := &worker{}
	w.handleConn(conn)

	os.Exit(0)
}

func (w *worker) setProcessName(c ssh.ConnMetadata) {
	gspt.SetProcTitle(fmt.Sprintf("%s %s@%s", "sshalama-worker", c.User(), c.RemoteAddr()))
}

// Verify if all capabilities have been dropped successfully
//
// The C constructor stuff is a bit of magic, so we ensure
// once more it did what it was supposed to do.
func checkDroppedCaps() {

	if files, _ := ioutil.ReadDir("/proc"); len(files) > 0 {
		log.Fatal("/proc appears not empty")
	}

	user, err := user.Current()
	if err != nil {
		log.Fatal("Could not determine current user: " + err.Error())
	}

	uid, _ := strconv.Atoi(user.Uid)
	if uid != 65534 {
		log.Fatalf("UID should be 65534, but was: %d", uid)
	}

	gid, _ := strconv.Atoi(user.Gid)
	if gid != 65534 {
		log.Fatalf("GID should be 65534, but was: %d", gid)
	}

}
