// Copyright (c) 2003-2005 Maxim Sobolev. All rights reserved.
// Copyright (c) 2006 Sippy Software, Inc. All rights reserved.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package sippy_cli

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type CLIManagerIface interface {
	Close()
	Send(string)
	RemoteAddr() net.Addr
}

type CLIConnectionManager struct {
	tcp            bool
	sock           net.Listener
	commandCb      func(clim CLIManagerIface, cmd string)
	acceptList     map[string]bool
	acceptListLock sync.RWMutex
	logger         sippy_log.ErrorLogger
}

func NewCLIConnectionManagerUnix(commandCb func(clim CLIManagerIface, cmd string), address string, uid, gid int, logger sippy_log.ErrorLogger) (*CLIConnectionManager, error) {
	addr, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err == nil {
		_ = conn.Close()
		return nil, fmt.Errorf("Another process listens on %s", address)
	}
	_ = os.Remove(address)
	sock, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}
	_ = os.Chown(address, uid, gid)
	_ = os.Chmod(address, 0660)
	return &CLIConnectionManager{
		commandCb: commandCb,
		sock:      sock,
		tcp:       false,
		logger:    logger,
	}, nil
}

func NewCLIConnectionManagerTcp(commandCb func(clim CLIManagerIface, cmd string), address string, logger sippy_log.ErrorLogger) (*CLIConnectionManager, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	sock, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &CLIConnectionManager{
		commandCb: commandCb,
		sock:      sock,
		tcp:       true,
		logger:    logger,
	}, nil
}

func (c *CLIConnectionManager) Start() {
	go c.run()
}

func (c *CLIConnectionManager) run() {
	defer c.sock.Close()
	for {
		conn, err := c.sock.Accept()
		if err != nil {
			c.logger.Error(err.Error())
			break
		}
		go c.handleAccept(conn)
	}
}

func (c *CLIConnectionManager) handleAccept(conn net.Conn) {
	if c.tcp {
		raddr, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			c.logger.Error("SplitHostPort failed. Possible bug: " + err.Error())
			// Not reached
			_ = conn.Close()
			return
		}
		c.acceptListLock.RLock()
		defer c.acceptListLock.RUnlock()
		if c.acceptList != nil {
			if _, ok := c.acceptList[raddr]; !ok {
				_ = conn.Close()
				return
			}
		}
	}
	cm := NewCLIManager(conn, c.commandCb, c.logger)
	go cm.run()
}

func (c *CLIConnectionManager) Shutdown() {
	c.sock.Close()
}

func (c *CLIConnectionManager) GetAcceptList() []string {
	c.acceptListLock.RLock()
	defer c.acceptListLock.RUnlock()
	if c.acceptList != nil {
		ret := make([]string, 0, len(c.acceptList))
		for addr, _ := range c.acceptList {
			ret = append(ret, addr)
		}
		return ret
	}
	return nil
}

func (c *CLIConnectionManager) SetAcceptList(acl []string) {
	acceptList := make(map[string]bool)
	for _, addr := range acl {
		acceptList[addr] = true
	}
	c.acceptListLock.Lock()
	c.acceptList = acceptList
	c.acceptListLock.Unlock()
}

func (c *CLIConnectionManager) AcceptListAppend(ip string) {
	c.acceptListLock.Lock()
	if c.acceptList == nil {
		c.acceptList = make(map[string]bool)
	}
	c.acceptList[ip] = true
	c.acceptListLock.Unlock()
}

func (c *CLIConnectionManager) AcceptListRemove(ip string) {
	c.acceptListLock.Lock()
	if c.acceptList != nil {
		delete(c.acceptList, ip)
	}
	c.acceptListLock.Unlock()

}

type CLIManager struct {
	sock      net.Conn
	commandCb func(CLIManagerIface, string)
	logger    sippy_log.ErrorLogger
}

func NewCLIManager(sock net.Conn, commandCb func(CLIManagerIface, string), logger sippy_log.ErrorLogger) *CLIManager {
	return &CLIManager{
		sock:      sock,
		commandCb: commandCb,
		logger:    logger,
	}
}

func (m *CLIManager) run() {
	defer m.sock.Close()
	reader := bufio.NewReader(m.sock)
	for {
		line, _, err := reader.ReadLine()
		if err != nil && err != syscall.EINTR {
			return
		} else {
			sippy_utils.SafeCall(func() { m.commandCb(m, string(line)) }, nil, m.logger)
		}
	}
}

func (m *CLIManager) Send(data string) {
	for len(data) > 0 {
		n, err := m.sock.Write([]byte(data))
		if err != nil && err != syscall.EINTR {
			return
		}
		data = data[n:]
	}
}

func (m *CLIManager) Close() {
	m.sock.Close()
}

func (m *CLIManager) RemoteAddr() net.Addr {
	return m.sock.RemoteAddr()
}
