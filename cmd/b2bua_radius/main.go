package main

import (
	"os"
	"strings"

	"github.com/sippy/go-b2bua/sippy"
	"github.com/sippy/go-b2bua/sippy/cli"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/types"
)

/*
from sippy.Timeout import Timeout
from sippy.Signal import Signal
from sippy.SipFrom import SipFrom
from sippy.SipTo import SipTo
from sippy.SipCiscoGUID import SipCiscoGUID
from sippy.UA import UA
from sippy.CCEvents import CCEventRing, CCEventConnect, CCEventDisconnect, CCEventTry, CCEventUpdate, CCEventFail
from sippy.UasStateTrying import UasStateTrying
from sippy.UasStateRinging import UasStateRinging
from sippy.UaStateDead import UaStateDead
from sippy.SipConf import SipConf
from sippy.SipHeader import SipHeader
from sippy.RadiusAuthorisation import RadiusAuthorisation
from sippy.RadiusAccounting import RadiusAccounting
from sippy.FakeAccounting import FakeAccounting
from sippy.SipLogger import SipLogger
from sippy.Rtp_proxy_session import Rtp_proxy_session
from sippy.Rtp_proxy_client import Rtp_proxy_client
from signal import SIGHUP, SIGPROF, SIGUSR1, SIGUSR2
from twisted.internet import reactor
from sippy.Cli_server_local import Cli_server_local
from sippy.SipTransactionManager import SipTransactionManager
from sippy.SipCallId import SipCallId
from sippy.StatefulProxy import StatefulProxy
from sippy.misc import daemonize
from sippy.B2BRoute import B2BRoute
import gc, getopt, os, sys
from re import sub
from time import time
from urllib import quote
from hashlib import md5
from sippy.MyConfigParser import MyConfigParser
from traceback import print_exc
from datetime import datetime

def re_replace(ptrn, s):
    s = s.split('#', 1)[0]
    ptrn = ptrn.split('/')
    while len(ptrn) > 0:
        op, p, r, mod = ptrn[:4]
        mod = mod.strip()
        if len(mod) > 0 && mod[0] != ';':
            ptrn[3] = mod[1:]
            mod = mod[0].lower()
        else:
            ptrn[3] = mod
        if 'g' in mod:
            s = sub(p, r, s)
        else:
            s = sub(p, r, s, 1)
        if len(ptrn) == 4 && ptrn[3] == '':
            break
        ptrn = ptrn[3:]
    return s

def reopen(signum, logfile):
    print('Signal %d received, reopening logs' % signum)
    fd = os.open(logfile, os.O_WRONLY | os.O_CREAT | os.O_APPEND)
    os.dup2(fd, sys.__stdout__.fileno())
    os.dup2(fd, sys.__stderr__.fileno())
    os.close(fd)

def usage(global_config, brief = false):
    print('usage: b2bua.py [--option1=value1] [--option2=value2] ... [--optionN==valueN]')
    if ! brief:
        print('\navailable options:\n')
        global_config.options_help()
    sys.exit(1)
*/

func main() {
	global_config := NewMyConfigParser()
	err := global_config.Parse()
	if err != nil {
		println(err.Error())
		return
	}

	var static_route *B2BRoute
	if global_config.Static_route != "" {
		static_route, err = NewB2BRoute(global_config.Static_route, global_config)
		if err != nil {
			println("Error parsing the static route")
			println(err.Error())
			return
		}
		//} else if ! global_config.auth_enable {
	} else if true { // radius is not implemented
		println("ERROR: static route should be specified when Radius auth is disabled")
		return
	}
	/*
	   if writeconf != nil:
	       global_config.write(open(writeconf, 'w'))

	   if ! global_config['foreground']:
	       daemonize(logfile = global_config['logfile'])
	*/
	rtp_proxy_clients := make([]sippy_types.RtpProxyClient, len(global_config.Rtp_proxy_clients))
	for i, address := range global_config.Rtp_proxy_clients {
		opts, err := sippy.NewRtpProxyClientOpts(address, nil /*bind_address*/, global_config, global_config.ErrorLogger())
		if err != nil {
			println("Cannot initialize rtpproxy client: " + err.Error())
			return
		}
		opts.SetHeartbeatInterval(global_config.Hrtb_ival)
		opts.SetHeartbeatRetryInterval(global_config.Hrtb_retr_ival)
		rtpp := sippy.NewRtpProxyClient(opts)
		err = rtpp.Start()
		if err != nil {
			println("Cannot initialize rtpproxy client: " + err.Error())
			return
		}
		rtp_proxy_clients[i] = rtpp
	}
	/*
	   if global_config['auth_enable'] || global_config['acct_enable']:
	       global_config['_radius_client'] = RadiusAuthorisation(global_config)
	*/
	global_config.SetMyUAName("Sippy B2BUA (RADIUS)")

	cmap := NewCallMap(global_config, rtp_proxy_clients, static_route)
	/*
	   if global_config.getdefault('xmpp_b2bua_id', nil) != nil:
	       global_config['_xmpp_mode'] = true
	*/
	sip_tm, err := sippy.NewSipTransactionManager(global_config, cmap)
	if err != nil {
		println("Cannot initialize SipTransactionManager: " + err.Error())
		return
	}
	//sip_tm.nat_traversal = global_config.nat_traversal
	cmap.Sip_tm = sip_tm
	if global_config.Sip_proxy != "" {
		var sip_proxy *sippy_net.HostPort
		host_port := strings.SplitN(global_config.Sip_proxy, ":", 2)
		if len(host_port) == 1 {
			sip_proxy = sippy_net.NewHostPort(host_port[0], "5060")
		} else {
			sip_proxy = sippy_net.NewHostPort(host_port[0], host_port[1])
		}
		cmap.Proxy = sippy.NewStatefulProxy(sip_tm, sip_proxy, global_config)
	}

	cmdfile := global_config.B2bua_socket
	if strings.HasPrefix(cmdfile, "unix:") {
		cmdfile = cmdfile[5:]
	}
	cli_server, err := sippy_cli.NewCLIConnectionManagerUnix(cmap.RecvCommand, cmdfile, os.Getuid(), os.Getgid(), global_config.ErrorLogger())
	if err != nil {
		println("Cannot initialize Cli_server: " + err.Error())
		return
	}
	cli_server.Start()
	/*
	   if ! global_config['foreground']:
	       file(global_config['pidfile'], 'w').write(str(os.getpid()) + '\n')
	       Signal(SIGUSR1, reopen, SIGUSR1, global_config['logfile'])
	*/
	sip_tm.Run()
}
