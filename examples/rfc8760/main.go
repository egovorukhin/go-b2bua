//
// Copyright (c) 2003-2005 Maxim Sobolev. All rights reserved.
// Copyright (c) 2006-2021 Sippy Software, Inc. All rights reserved.
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

package main

import (
	crand "crypto/rand"
	"flag"
	mrand "math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

func init() {
	Next_cc_id = make(chan int64)
	go func() {
		var id int64 = 1
		for {
			Next_cc_id <- id
			id++
		}
	}()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	buf := make([]byte, 8)
	crand.Read(buf)
	var salt int64
	for _, c := range buf {
		salt = (salt << 8) | int64(c)
	}
	mrand.Seed(salt)

	var laddr, nh_addr, logfile string
	var lport int
	var authname_in, authname_out, passwd_in, passwd_out, hash_alg string

	flag.StringVar(&laddr, "l", "", "Local addr")
	flag.IntVar(&lport, "p", 5060, "Local port")
	flag.StringVar(&nh_addr, "n", "192.168.0.102:5060", "Next hop address")
	flag.StringVar(&logfile, "L", "/var/log/sip.log", "Log file")
	flag.StringVar(&authname_in, "authname_in", "", "Expected authname")
	flag.StringVar(&passwd_in, "passwd_in", "", "Expected password")
	flag.StringVar(&authname_out, "authname_out", "", "Outgoing authname")
	flag.StringVar(&passwd_out, "passwd_out", "", "Ougoing password")
	flag.StringVar(&hash_alg, "alg", "MD5", "Hash algorithm")
	flag.Parse()

	//config.SetIPV6Enabled(false)
	error_logger := sippy_log.NewErrorLogger()
	sip_logger, err := sippy_log.NewSipLogger("b2bua", logfile)
	if err != nil {
		error_logger.Error(err)
		return
	}
	config := NewMyConfig(error_logger, sip_logger)
	config.Authname_in = authname_in
	config.Authname_out = authname_out
	config.Passwd_in = passwd_in
	config.Passwd_out = passwd_out
	config.Hash_alg = hash_alg

	if nh_addr != "" {
		var parts []string
		var addr string

		if strings.HasPrefix(nh_addr, "[") {
			parts = strings.SplitN(nh_addr, "]", 2)
			addr = parts[0] + "]"
			if len(parts) == 2 {
				parts = strings.SplitN(parts[1], ":", 2)
			}
		} else {
			parts = strings.SplitN(nh_addr, ":", 2)
			addr = parts[0]
		}
		port := "5060"
		if len(parts) == 2 {
			port = parts[1]
		}
		config.Nh_addr = sippy_net.NewHostPort(addr, port)
	}
	config.SetMyUAName("Sippy B2BUA (Simple)")
	config.SetAllowFormats([]int{0, 8, 18, 100, 101})
	if laddr != "" {
		config.SetMyAddress(sippy_net.NewMyAddress(laddr))
	}
	config.SetSipAddress(config.GetMyAddress())
	if lport > 0 {
		config.SetMyPort(sippy_net.NewMyPort(strconv.Itoa(lport)))
	}
	config.SetSipPort(config.GetMyPort())
	cmap, err := NewCallMap(config, error_logger)
	if err != nil {
		error_logger.Error(err)
		return
	}
	sip_tm, err := sippy.NewSipTransactionManager(config, cmap)
	if err != nil {
		error_logger.Error(err)
		return
	}
	cmap.Sip_tm = sip_tm
	cmap.Proxy = sippy.NewStatefulProxy(sip_tm, config.Nh_addr, config)
	go sip_tm.Run()

	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	select {
	case <-signal_chan:
		cmap.Shutdown()
		sip_tm.Shutdown()
		time.Sleep(time.Second)
		break
	}
}
