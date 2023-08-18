package sippy

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/fmt"
	"github.com/sippy/go-b2bua/sippy/math"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
	"github.com/sippy/go-b2bua/sippy/utils"
)

type Rtp_proxy_client_udp struct {
	_address         net.Addr
	uopts            *udpServerOpts
	pending_requests map[string]*rtpp_req_udp
	global_config    sippy_conf.Config
	delay_flt        sippy_math.RecFilter
	worker           *UdpServer
	hostPort         *sippy_net.HostPort
	bind_address     *sippy_net.HostPort
	lock             sync.Mutex
	owner            sippy_types.RtpProxyClient
}

type rtpp_req_udp struct {
	next_retr       float64
	triesleft       int64
	timer           *Timeout
	command         string
	result_callback func(string)
	stime           *sippy_time.MonoTime
	retransmits     int
}

func new_rtpp_req_udp(next_retr float64, triesleft int64, timer *Timeout, command string, result_callback func(string)) *rtpp_req_udp {
	stime, _ := sippy_time.NewMonoTime()
	return &rtpp_req_udp{
		next_retr:       next_retr,
		triesleft:       triesleft,
		timer:           timer,
		command:         command,
		result_callback: result_callback,
		stime:           stime,
		retransmits:     0,
	}
}

func getnretrans(first_retr, timeout float64) (int64, error) {
	if first_retr < 0 {
		return 0, errors.New(sippy_fmt.Sprintf("getnretrans(%f, %f)", first_retr, timeout))
	}
	var n int64 = 0
	for {
		timeout -= first_retr
		if timeout < 0 {
			break
		}
		first_retr *= 2.0
		n += 1
	}
	return n, nil
}

func newRtp_proxy_client_udp(owner sippy_types.RtpProxyClient, global_config sippy_conf.Config, address net.Addr, bind_address *sippy_net.HostPort) (rtp_proxy_transport, error) {
	s := &Rtp_proxy_client_udp{
		owner:            owner,
		pending_requests: make(map[string]*rtpp_req_udp),
		global_config:    global_config,
		delay_flt:        sippy_math.NewRecFilter(0.95, 0.25),
		bind_address:     bind_address,
	}
	laddress, err := s._setup_addr(address, bind_address)
	if err != nil {
		return nil, err
	}
	s.uopts = NewUdpServerOpts(laddress, s.process_reply)
	//s.uopts.ploss_out_rate = s.ploss_out_rate
	//s.uopts.pdelay_out_max = s.pdelay_out_max
	if owner.GetOpts().GetNWorkers() != nil {
		s.uopts.nworkers = *owner.GetOpts().GetNWorkers()
	}
	s.worker, err = NewUdpServer(global_config, s.uopts)
	return s, err
}

func (s *Rtp_proxy_client_udp) _setup_addr(address net.Addr, bind_address *sippy_net.HostPort) (*sippy_net.HostPort, error) {
	var err error

	s.hostPort, err = sippy_net.NewHostPortFromAddr(address)
	if err != nil {
		return nil, err
	}
	s._address = address
	s.bind_address = bind_address

	if bind_address == nil {
		if s.hostPort.Host.String()[0] == '[' {
			if s.hostPort.Host.String() == "[::1]" {
				bind_address = sippy_net.NewHostPort("[::1]", "0")
			} else {
				bind_address = sippy_net.NewHostPort("[::]", "0")
			}
		} else {
			if strings.HasPrefix(s.hostPort.Host.String(), "127.") {
				bind_address = sippy_net.NewHostPort("127.0.0.1", "0")
			} else {
				bind_address = sippy_net.NewHostPort("0.0.0.0", "0")
			}
		}
	}
	return bind_address, nil
}

func (*Rtp_proxy_client_udp) is_local() bool {
	return false
}

func (s *Rtp_proxy_client_udp) address() net.Addr {
	return s._address
}

func (s *Rtp_proxy_client_udp) send_command(command string, result_callback func(string)) {
	buf := make([]byte, 16)
	rand.Read(buf)
	cookie := hex.EncodeToString(buf)
	next_retr := s.delay_flt.GetLastval() * 4.0
	exp_time := 3.0
	if command[0] == 'I' {
		exp_time = 10.0
	} else if command[0] == 'G' {
		exp_time = 1.0
	}
	nretr, err := getnretrans(next_retr, exp_time)
	if err != nil {
		s.global_config.ErrorLogger().Debug("getnretrans error: " + err.Error())
		return
	}
	command = cookie + " " + command
	timer := StartTimeout(func() { s.retransmit(cookie) }, nil, time.Duration(next_retr*float64(time.Second)), 1, s.global_config.ErrorLogger())
	preq := new_rtpp_req_udp(next_retr, nretr-1, timer, command, result_callback)
	s.worker.SendTo([]byte(command), s.hostPort)
	s.lock.Lock()
	s.pending_requests[cookie] = preq
	s.lock.Unlock()
}

func (s *Rtp_proxy_client_udp) retransmit(cookie string) {
	s.lock.Lock()
	req, ok := s.pending_requests[cookie]
	if !ok {
		s.lock.Unlock()
		return
	}
	if req.triesleft <= 0 || s.worker == nil {
		delete(s.pending_requests, cookie)
		s.lock.Unlock()
		s.owner.GoOffline()
		if req.result_callback != nil {
			sippy_utils.SafeCall(func() { req.result_callback("") }, nil /*lock*/, s.global_config.ErrorLogger())
		}
		return
	}
	s.lock.Unlock()
	req.next_retr *= 2
	req.retransmits += 1
	req.timer = StartTimeout(func() { s.retransmit(cookie) }, nil, time.Duration(req.next_retr*float64(time.Second)), 1, s.global_config.ErrorLogger())
	req.stime, _ = sippy_time.NewMonoTime()
	s.worker.SendTo([]byte(req.command), s.hostPort)
	req.triesleft -= 1
}

func (s *Rtp_proxy_client_udp) process_reply(data []byte, address *sippy_net.HostPort, worker sippy_net.Transport, rtime *sippy_time.MonoTime) {
	arr := sippy_utils.FieldsN(string(data), 2)
	if len(arr) != 2 {
		s.global_config.ErrorLogger().Debug("Rtp_proxy_client_udp.process_reply(): invalid response " + string(data))
		return
	}
	cookie, result := arr[0], arr[1]
	s.lock.Lock()
	req, ok := s.pending_requests[cookie]
	delete(s.pending_requests, cookie)
	s.lock.Unlock()
	if !ok {
		return
	}
	req.timer.Cancel()
	if req.result_callback != nil {
		sippy_utils.SafeCall(func() { req.result_callback(strings.TrimSpace(result)) }, nil /*lock*/, s.global_config.ErrorLogger())
	}
	if req.retransmits == 0 {
		// When we had to do retransmit it is not possible to figure out whether
		// or not this reply is related to the original request or one of the
		// retransmits. Therefore, using it to estimate delay could easily produce
		// bogus value that is too low or even negative if we cook up retransmit
		// while the original response is already in the queue waiting to be
		// processed. This should not be a big issue since UDP command channel does
		// not work very well if the packet loss goes to more than 30-40%.
		s.delay_flt.Apply(rtime.Sub(req.stime).Seconds())
		//print "Rtp_proxy_client_udp.process_reply(): delay %f" % (rtime - stime)
	}
}

func (s *Rtp_proxy_client_udp) reconnect(address net.Addr, bind_address *sippy_net.HostPort) {
	if s._address.String() != address.String() || bind_address.String() != s.bind_address.String() {
		s.uopts.laddress, _ = s._setup_addr(address, bind_address)
		s.worker.Shutdown()
		s.worker, _ = NewUdpServer(s.global_config, s.uopts)
		s.delay_flt = sippy_math.NewRecFilter(0.95, 0.25)
	}
}

func (s *Rtp_proxy_client_udp) shutdown() {
	s.worker.Shutdown()
	s.worker = nil
}

func (s *Rtp_proxy_client_udp) get_rtpc_delay() float64 {
	return s.delay_flt.GetLastval()
}

/*
class selftest(object):
    def gotreply(s, *args):
        from twisted.internet import reactor
        print args
        reactor.crash()

    def run(s):
        import os
        from twisted.internet import reactor
        global_config = {}
        global_config["my_pid"] = os.getpid()
        rtpc = Rtp_proxy_client_udp(global_config, ("127.0.0.1", 22226), nil)
        os.system("sockstat | grep -w %d" % global_config["my_pid"])
        rtpc.send_command("Ib", s.gotreply)
        reactor.run()
        rtpc.reconnect(("localhost", 22226), ("0.0.0.0", 34222))
        os.system("sockstat | grep -w %d" % global_config["my_pid"])
        rtpc.send_command("V", s.gotreply)
        reactor.run()
        rtpc.reconnect(("localhost", 22226), ("127.0.0.1", 57535))
        os.system("sockstat | grep -w %d" % global_config["my_pid"])
        rtpc.send_command("V", s.gotreply)
        reactor.run()
        rtpc.shutdown()

if __name__ == "__main__":
    selftest().run()
*/
