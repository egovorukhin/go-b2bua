package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/cli"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type CallMap struct {
	global_config     *myConfigParser
	ccmap             map[int64]*callController
	ccmap_lock        sync.Mutex
	gc_timeout        time.Duration
	debug_mode        bool
	safe_restart      bool
	Sip_tm            sippy_types.SipTransactionManager
	Proxy             sippy_types.StatefulProxy
	cc_id             int64
	cc_id_lock        sync.Mutex
	rtp_proxy_clients []sippy_types.RtpProxyClient
	static_route      *B2BRoute
}

/*
class CallMap(object):
    ccmap = nil
    el = nil
    global_config = nil
    //rc1 = nil
    //rc2 = nil
*/

func NewCallMap(global_config *myConfigParser, rtp_proxy_clients []sippy_types.RtpProxyClient,
	static_route *B2BRoute) *CallMap {
	s := &CallMap{
		global_config:     global_config,
		ccmap:             make(map[int64]*callController),
		gc_timeout:        time.Minute,
		debug_mode:        false,
		safe_restart:      false,
		rtp_proxy_clients: rtp_proxy_clients,
		static_route:      static_route,
	}
	go func() {
		sighup_ch := make(chan os.Signal, 1)
		signal.Notify(sighup_ch, syscall.SIGHUP)
		/*sigusr2_ch := make(chan os.Signal, 1)
		signal.Notify(sigusr2_ch, syscall.SIGUSR2)
		sigprof_ch := make(chan os.Signal, 1)
		signal.Notify(sigprof_ch, syscall.SIGPROF)*/
		sigterm_ch := make(chan os.Signal, 1)
		signal.Notify(sigterm_ch, syscall.SIGTERM)
		for {
			select {
			case <-sighup_ch:
				s.discAll(syscall.SIGHUP)
			/*case <-sigusr2_ch:
				s.toggleDebug()
			case <-sigprof_ch:
				s.safeRestart()*/
			case <-sigterm_ch:
				s.safeStop()
			}
		}
	}()
	go func() {
		for {
			time.Sleep(s.gc_timeout)
			s.GClector()
		}
	}()
	return s
}

func (s *CallMap) OnNewDialog(req sippy_types.SipRequest, sip_t sippy_types.ServerTransaction) (sippy_types.UA, sippy_types.RequestReceiver, sippy_types.SipResponse) {
	to_body, err := req.GetTo().GetBody(s.global_config)
	if err != nil {
		s.global_config.ErrorLogger().Error("CallMap::OnNewDialog: #1: " + err.Error())
		return nil, nil, req.GenResponse(500, "Internal Server Error", nil, nil)
	}
	//except Exception as exception:
	//println(datetime.now(), "can\"t parse SIP request: %s:\n" % str(exception))
	//println( "-" * 70)
	//print_exc(file = sys.stdout)
	//println( "-" * 70)
	//println(req)
	//println("-" * 70)
	//sys.stdout.flush()
	//return (nil, nil, nil)
	if to_body.GetTag() != "" {
		// Request within dialog, but no such dialog
		return nil, nil, req.GenResponse(481, "Call Leg/Transaction Does Not Exist", nil, nil)
	}
	if req.GetMethod() == "INVITE" {
		// New dialog
		var via *sippy_header.SipViaBody
		vias := req.GetVias()
		if len(vias) > 1 {
			via, err = vias[1].GetBody()
		} else {
			via, err = vias[0].GetBody()
		}
		if err != nil {
			s.global_config.ErrorLogger().Error("CallMap::OnNewDialog: #2: " + err.Error())
			return nil, nil, req.GenResponse(500, "Internal Server Error", nil, nil)
		}
		remote_ip := via.GetTAddr(s.global_config).Host
		source := req.GetSource()

		// First check if request comes from IP that
		// we want to accept our traffic from
		if !s.global_config.checkIP(source.Host.String()) {
			return nil, nil, req.GenResponse(403, "Forbidden", nil, nil)
		}
		/*
		   var challenge *sippy_header.SipWWWAuthenticate
		   if s.global_config.auth_enable {
		       // Prepare challenge if no authorization header is present.
		       // Depending on configuration, we might try remote ip auth
		       // first and then challenge it or challenge immediately.
		       if s.global_config["digest_auth"] && req.countHFs("authorization") == 0 {
		           challenge = NewSipWWWAuthenticate()
		           challenge.getBody().realm = req.getRURI().host
		       }
		       // Send challenge immediately if digest is the
		       // only method of authenticating
		       if challenge != nil && s.global_config.getdefault("digest_auth_only", false) {
		           resp = req.GenResponse(401, "Unauthorized")
		           resp.appendHeader(challenge)
		           return resp, nil, nil
		       }
		   }
		*/
		pass_headers := []sippy_header.SipHeader{}
		for _, header := range s.global_config.pass_headers {
			hfs := req.GetHFs(header)
			pass_headers = append(pass_headers, hfs...)
		}
		s.cc_id_lock.Lock()
		id := s.cc_id
		s.cc_id++
		s.cc_id_lock.Unlock()
		cguid := req.GetCGUID()
		if cguid == nil && req.GetH323ConfId() != nil {
			cguid = req.GetH323ConfId().AsCiscoGUID()
		}
		if cguid == nil {
			cguid = sippy_header.NewSipCiscoGUID()
		}
		cc := NewCallController(id, remote_ip, source, s.global_config, pass_headers, s.Sip_tm, cguid, s)
		//cc.challenge = challenge
		//rval := cc.uaA.RecvRequest(req, sip_t)
		s.ccmap_lock.Lock()
		s.ccmap[id] = cc
		s.ccmap_lock.Unlock()
		return cc.uaA, cc.uaA, nil
	}
	if s.Proxy != nil && (req.GetMethod() == "REGISTER" || req.GetMethod() == "SUBSCRIBE") {
		return nil, s.Proxy, nil
	}
	if req.GetMethod() == "NOTIFY" || req.GetMethod() == "PING" {
		// Whynot?
		return nil, nil, req.GenResponse(200, "OK", nil, nil)
	}
	return nil, nil, req.GenResponse(501, "Not Implemented", nil, nil)
}

func (s CallMap) safeStop() {
	s.discAll(0)
	time.Sleep(time.Second)
	os.Exit(0)
}

func (s *CallMap) discAll(signum syscall.Signal) {
	if signum > 0 {
		println(fmt.Sprintf("Signal %d received, disconnecting all calls", signum))
	}
	alist := []*callController{}
	s.ccmap_lock.Lock()
	for _, cc := range s.ccmap {
		alist = append(alist, cc)
	}
	s.ccmap_lock.Unlock()
	for _, cc := range alist {
		cc.disconnect(nil)
	}
}

func (s *CallMap) toggleDebug() {
	if s.debug_mode {
		println("Signal received, toggling extra debug output off")
	} else {
		println("Signal received, toggling extra debug output on")
	}
	s.debug_mode = !s.debug_mode
}

func (s *CallMap) safeRestart() {
	println("Signal received, scheduling safe restart")
	s.safe_restart = true
}

func (s *CallMap) GClector() {
	fmt.Printf("GC is invoked, %d calls in map\n", len(s.ccmap))
	if s.debug_mode {
		//println(s.global_config["_sip_tm"].tclient, s.global_config["_sip_tm"].tserver)
		for _, cc := range s.ccmap {
			println(cc.uaA.GetStateName(), cc.uaO.GetStateName())
		}
		//} else {
		//    fmt.Printf("[%d]: %d client, %d server transactions in memory\n",
		//      os.getpid(), len(s.global_config["_sip_tm"].tclient), len(s.global_config["_sip_tm"].tserver))
	}
	if s.safe_restart {
		if len(s.ccmap) == 0 {
			s.Sip_tm.Shutdown()
			//os.chdir(s.global_config["_orig_cwd"])
			cmd := exec.Command(os.Args[0], os.Args[1:]...)
			cmd.Env = os.Environ()
			err := cmd.Start()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			os.Exit(0)
			// Should not reach this point!
		}
		s.gc_timeout = time.Second
	}
}

func (s *CallMap) RecvCommand(clim sippy_cli.CLIManagerIface, data string) {
	args := strings.Split(strings.TrimSpace(data), " ")
	cmd := strings.ToLower(args[0])
	args = args[1:]
	switch cmd {
	case "q":
		clim.Close()
		return
	case "l":
		res := "In-memory calls:\n"
		total := 0
		s.ccmap_lock.Lock()
		defer s.ccmap_lock.Unlock()
		for _, cc := range s.ccmap {
			cc.lock.Lock()
			res += fmt.Sprintf("%s: %s (", cc.cId.CallId, cc.state.String())
			if cc.uaA != nil {
				res += fmt.Sprintf("%s %s %s %s -> ", cc.uaA.GetStateName(), cc.uaA.GetRAddr0().String(),
					cc.uaA.GetCLD(), cc.uaA.GetCLI())
			} else {
				res += "N/A -> "
			}
			if cc.uaO != nil {
				res += fmt.Sprintf("%s %s %s %s)\n", cc.uaO.GetStateName(), cc.uaO.GetRAddr0().String(),
					cc.uaO.GetCLI(), cc.uaO.GetCLD())
			} else {
				res += "N/A)\n"
			}
			total += 1
			cc.lock.Unlock()
		}
		clim.Send(res + fmt.Sprintf("Total: %d\n", total))
		return
		/*
		   case "lt":
		       res = "In-memory server transactions:\n"
		       for tid, t in s.global_config["_sip_tm"].tserver.iteritems() {
		           res += "%s %s %s\n" % (tid, t.method, t.state)
		       }
		       res += "In-memory client transactions:\n"
		       for tid, t in s.global_config["_sip_tm"].tclient.iteritems():
		           res += "%s %s %s\n" % (tid, t.method, t.state)
		       return res
		   case "lt", "llt":
		       if cmd == "llt":
		           mindur = 60.0
		       else:
		           mindur = 0.0
		       ctime = time()
		       res = "In-memory server transactions:\n"
		       for tid, t in s.global_config["_sip_tm"].tserver.iteritems():
		           duration = ctime - t.rtime
		           if duration < mindur:
		               continue
		           res += "%s %s %s %s\n" % (tid, t.method, t.state, duration)
		       res += "In-memory client transactions:\n"
		       for tid, t in s.global_config["_sip_tm"].tclient.iteritems():
		           duration = ctime - t.rtime
		           if duration < mindur:
		               continue
		           res += "%s %s %s %s\n" % (tid, t.method, t.state, duration)
		       return res
		*/
	case "d":
		if len(args) != 1 {
			clim.Send("ERROR: syntax error: d <call-id>\n")
			return
		}
		if args[0] == "*" {
			s.discAll(0)
			clim.Send("OK\n")
			return
		}
		dlist := []*callController{}
		s.ccmap_lock.Lock()
		for _, cc := range s.ccmap {
			if cc.cId.CallId != args[0] {
				continue
			}
			dlist = append(dlist, cc)
		}
		s.ccmap_lock.Unlock()
		if len(dlist) == 0 {
			clim.Send(fmt.Sprintf("ERROR: no call with id of %s has been found\n", args[0]))
			return
		}
		for _, cc := range dlist {
			cc.disconnect(nil)
		}
		clim.Send("OK\n")
		return
	case "r":
		if len(args) != 1 {
			clim.Send("ERROR: syntax error: r [<id>]\n")
			return
		}
		idx, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			clim.Send("ERROR: non-integer argument: " + args[0] + "\n")
			return
		}
		s.ccmap_lock.Lock()
		cc, ok := s.ccmap[idx]
		s.ccmap_lock.Unlock()
		if !ok {
			clim.Send(fmt.Sprintf("ERROR: no call with id of %d has been found\n", idx))
			return
		}
		if cc.proxied {
			ts, _ := sippy_time.NewMonoTime()
			ts = ts.Add(-60 * time.Second)
			if cc.state == CCStateConnected {
				cc.disconnect(ts)
			} else if cc.state == CCStateARComplete {
				cc.uaO.Disconnect(ts, "")
			}
		}
		clim.Send("OK\n")
		return
	default:
		clim.Send("ERROR: unknown command\n")
	}
}

func (s *CallMap) DropCC(cc_id int64) {
	s.ccmap_lock.Lock()
	delete(s.ccmap, cc_id)
	s.ccmap_lock.Unlock()
}
