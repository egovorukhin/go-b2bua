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
	"sync"
	"syscall"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

var next_cc_id chan int64

func init() {
	next_cc_id = make(chan int64)
	go func() {
		var id int64 = 1
		for {
			next_cc_id <- id
			id++
		}
	}()
}

type callController struct {
	uaA  sippy_types.UA
	uaO  sippy_types.UA
	lock *sync.Mutex // this must be a reference to prevent memory leak
	id   int64
	cmap *callMap
}

func NewCallController(cmap *callMap) *callController {
	s := &callController{
		id:   <-next_cc_id,
		uaO:  nil,
		lock: new(sync.Mutex),
		cmap: cmap,
	}
	s.uaA = sippy.NewUA(cmap.sip_tm, cmap.config, cmap.config.nh_addr, s, s.lock, nil)
	s.uaA.SetDeadCb(s.aDead)
	//s.uaA.SetCreditTime(5 * time.Second)
	return s
}

func (s *callController) RecvEvent(event sippy_types.CCEvent, ua sippy_types.UA) {
	if ua == s.uaA {
		if s.uaO == nil {
			if _, ok := event.(*sippy.CCEventTry); !ok {
				// Some weird event received
				s.uaA.RecvEvent(sippy.NewCCEventDisconnect(nil, event.GetRtime(), ""))
				return
			}
			s.uaO = sippy.NewUA(s.cmap.sip_tm, s.cmap.config, s.cmap.config.nh_addr, s, s.lock, nil)
			s.uaO.SetDeadCb(s.oDead)
			s.uaO.SetRAddr(s.cmap.config.nh_addr)
		}
		s.uaO.RecvEvent(event)
	} else {
		s.uaA.RecvEvent(event)
	}
}

func (s *callController) aDead() {
	s.cmap.Remove(s.id)
}

func (s *callController) oDead() {
	s.cmap.Remove(s.id)
}

func (s *callController) Shutdown() {
	s.uaA.Disconnect(nil, "")
}

func (s *callController) String() string {
	res := "uaA:" + s.uaA.String() + ", uaO: "
	if s.uaO == nil {
		res += "nil"
	} else {
		res += s.uaO.String()
	}
	return res
}

type callMap struct {
	config     *myconfig
	logger     sippy_log.ErrorLogger
	sip_tm     sippy_types.SipTransactionManager
	proxy      sippy_types.StatefulProxy
	ccmap      map[int64]*callController
	ccmap_lock sync.Mutex
}

func NewCallMap(config *myconfig, logger sippy_log.ErrorLogger) *callMap {
	return &callMap{
		logger: logger,
		config: config,
		ccmap:  make(map[int64]*callController),
	}
}

func (s *callMap) OnNewDialog(req sippy_types.SipRequest, tr sippy_types.ServerTransaction) (sippy_types.UA, sippy_types.RequestReceiver, sippy_types.SipResponse) {
	to_body, err := req.GetTo().GetBody(s.config)
	if err != nil {
		s.logger.Error("CallMap::OnNewDialog: #1: " + err.Error())
		return nil, nil, req.GenResponse(500, "Internal Server Error", nil, nil)
	}
	if to_body.GetTag() != "" {
		// Request within dialog, but no such dialog
		return nil, nil, req.GenResponse(481, "Call Leg/Transaction Does Not Exist", nil, nil)
	}
	if req.GetMethod() == "INVITE" {
		// New dialog
		cc := NewCallController(s)
		s.ccmap_lock.Lock()
		s.ccmap[cc.id] = cc
		s.ccmap_lock.Unlock()
		return cc.uaA, cc.uaA, nil
	}
	if req.GetMethod() == "REGISTER" {
		// Registration
		return nil, s.proxy, nil
	}
	if req.GetMethod() == "NOTIFY" || req.GetMethod() == "PING" {
		// Whynot?
		return nil, nil, req.GenResponse(200, "OK", nil, nil)
	}
	return nil, nil, req.GenResponse(501, "Not Implemented", nil, nil)
}

func (s *callMap) Remove(ccid int64) {
	s.ccmap_lock.Lock()
	defer s.ccmap_lock.Unlock()
	delete(s.ccmap, ccid)
}

func (s *callMap) Shutdown() {
	acalls := []*callController{}
	s.ccmap_lock.Lock()
	for _, cc := range s.ccmap {
		acalls = append(acalls, cc)
	}
	s.ccmap_lock.Unlock()
	for _, cc := range acalls {
		//println(cc.String())
		cc.Shutdown()
	}
}

type myconfig struct {
	sippy_conf.Config

	nh_addr *sippy_net.HostPort
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
	var foreground bool

	flag.StringVar(&laddr, "l", "", "Local addr")
	flag.IntVar(&lport, "p", -1, "Local port")
	flag.StringVar(&nh_addr, "n", "", "Next hop address")
	flag.BoolVar(&foreground, "f", false, "Run in foreground")
	flag.StringVar(&logfile, "L", "/var/log/sip.log", "Log file")
	flag.Parse()

	error_logger := sippy_log.NewErrorLogger()
	sip_logger, err := sippy_log.NewSipLogger("b2bua", logfile)
	if err != nil {
		error_logger.Error(err)
		return
	}
	config := &myconfig{
		Config:  sippy_conf.NewConfig(error_logger, sip_logger),
		nh_addr: sippy_net.NewHostPort("192.168.0.102", "5060"), // next hop address
	}
	//config.SetIPV6Enabled(false)
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
		config.nh_addr = sippy_net.NewHostPort(addr, port)
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
	cmap := NewCallMap(config, error_logger)
	sip_tm, err := sippy.NewSipTransactionManager(config, cmap)
	if err != nil {
		error_logger.Error(err)
		return
	}
	cmap.sip_tm = sip_tm
	cmap.proxy = sippy.NewStatefulProxy(sip_tm, config.nh_addr, config)
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
