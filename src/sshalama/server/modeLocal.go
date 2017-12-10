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
	"runtime"
	"time"

	"sshalama/util/stop"

	"github.com/Freeaqingme/opencontainers-runc/libcontainer"
	"github.com/Freeaqingme/opencontainers-runc/libcontainer/configs"
	_ "github.com/Freeaqingme/opencontainers-runc/libcontainer/nsenter"
	"github.com/prep/socketpair"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, _ := libcontainer.New("")
		if err := factory.StartInitialization(); err != nil {
			log.Fatal(err)
		}
		panic("--this line should have never been executed, congratulations--")
	}
}

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

	factory, err := libcontainer.New("./container", libcontainer.Cgroupfs, libcontainer.InitArgs(os.Args[0], "init"))
	if err != nil {
		log.Fatal(err)
	}

	defaultMountFlags := unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV | unix.MS_RDONLY
	config := &configs.Config{
		Rootfs: "/home/dolf/Projects/Sshalama/rootfs/",
		Capabilities: &configs.Capabilities{
			Bounding: []string{
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SYS_ADMIN",
			},
			Effective: []string{
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SYS_ADMIN",
			},
			Inheritable: []string{},
			Permitted: []string{
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SYS_ADMIN",
			},
			Ambient: []string{},
		},
		Namespaces: configs.Namespaces([]configs.Namespace{
			{Type: configs.NEWNS},
			{Type: configs.NEWUTS},
			{Type: configs.NEWIPC},
			{Type: configs.NEWPID},
			{Type: configs.NEWNET},
		}),
		Cgroups: &configs.Cgroup{
			Name:   "test-container",
			Parent: "system",
			Resources: &configs.Resources{
				MemorySwappiness: nil,
				AllowAllDevices:  nil,
				AllowedDevices:   configs.DefaultAllowedDevices,
			},
		},
		MaskPaths: []string{
			"/proc/kcore",
			"/sys/firmware",
		},
		ReadonlyPaths: []string{
			"/proc/sys", "/proc/sysrq-trigger", "/proc/irq", "/proc/bus",
		},
		Devices:  configs.DefaultAutoCreatedDevices,
		Hostname: "testing",
		Mounts: []*configs.Mount{
			{ // TODO: Find a way so we can get rid of procfs before handing over control to our worker process
				Source:      "proc",
				Destination: "/proc",
				Device:      "proc",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "tmpfs",
				Destination: "/dev",
				Device:      "tmpfs",
				Flags:       unix.MS_NOSUID | unix.MS_STRICTATIME,
				Data:        "mode=755",
			},
			{
				Source:      "devpts",
				Destination: "/dev/pts",
				Device:      "devpts",
				Flags:       unix.MS_NOSUID | unix.MS_NOEXEC,
				Data:        "newinstance,ptmxmode=0666,mode=0620,gid=5",
			},
			{
				Device:      "tmpfs",
				Source:      "shm",
				Destination: "/dev/shm",
				Data:        "mode=1777,size=65536k",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "mqueue",
				Destination: "/dev/mqueue",
				Device:      "mqueue",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "sysfs",
				Destination: "/sys",
				Device:      "sysfs",
				Flags:       defaultMountFlags | unix.MS_RDONLY,
			},
		},
		Networks: []*configs.Network{
			{
				Type:    "loopback",
				Address: "127.0.0.1/0",
				Gateway: "localhost",
			},
		},
		/*Rlimits: []configs.Rlimit{
			{
				Type: unix.RLIMIT_NOFILE,
				Hard: uint64(1025),
				Soft: uint64(1025),
			},
		},*/
	}

	config.Namespaces.Add(configs.NEWNET, "/var/run/netns/pia-ch")

	container, err := factory.Create("container-id", config)
	if err != nil {
		log.Fatal(err)
	}

	noNewPrivs := true
	process := &libcontainer.Process{
		Args:            []string{"/bin/sshalama-worker"},
		Env:             []string{"PATH=/bin:/usr/bin"},
		NoNewPrivileges: &noNewPrivs,
		Stdin:           os.Stdin,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
		ExtraFiles: []*os.File{
			workerConnFd,
		},
	}

	if err := container.Run(process); err != nil {
		container.Destroy()
		log.Fatal(err)
	}

	go func() {
		stopper := stop.NewStopper(func() {
			container.Destroy()
			serverConn.Close()
			workerConn.Close()
		})

		_, err = process.Wait()
		if err != nil {
			log.Fatal(err)
		}

		stopper.Run()
		stopper.Unregister()
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
