package sippy

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/math"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
	"github.com/sippy/go-b2bua/sippy/utils"
)

const (
	_RTPPLWorker_MAX_RETRIES = 3
)

type rtpp_req_stream struct {
	command         string
	result_callback func(string)
}

type _RTPPLWorker struct {
	userv         *Rtp_proxy_client_stream
	shutdown_chan chan int
}

func newRTPPLWorker(userv *Rtp_proxy_client_stream) *_RTPPLWorker {
	s := &_RTPPLWorker{
		userv:         userv,
		shutdown_chan: make(chan int, 1),
	}
	go s.run()
	return s
}

func (s *_RTPPLWorker) send_raw(command string, stime *sippy_time.MonoTime) (string, time.Duration, error) {
	//print "%s.send_raw(%s)" % (id(s), command)
	if stime == nil {
		stime, _ = sippy_time.NewMonoTime()
	}
	var err error
	var n int
	retries := 0
	rval := ""
	buf := make([]byte, 1024)
	var s net.Conn
	for {
		if retries > _RTPPLWorker_MAX_RETRIES {
			return "", 0, fmt.Errorf("Error sending to the rtpproxy on " + s.userv._address.String() + ": " + err.Error())
		}
		retries++
		if s != nil {
			s.Close()
		}
		s, err = net.Dial(s.userv._address.Network(), s.userv._address.String())
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		_, err = s.Write([]byte(command))
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		n, err = s.Read(buf)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		rval = strings.TrimSpace(string(buf[:n]))
		break
	}
	s.Close()
	rtpc_delay, _ := stime.OffsetFromNow()
	return rval, rtpc_delay, nil
}

func (s *_RTPPLWorker) run() {
	for {
		req := <-s.userv.wi
		if req == nil {
			// Shutdown request, relay it further
			s.userv.wi <- nil
			break
		}
		//command, result_callback, callback_parameters = wi
		data, rtpc_delay, err := s.send_raw(req.command, nil)
		if err != nil {
			s.userv.global_config.ErrorLogger().Debug("Error communicating the rtpproxy: " + err.Error())
			data, rtpc_delay = "", -1
		}
		if len(data) == 0 {
			rtpc_delay = -1
		}
		if req.result_callback != nil {
			sippy_utils.SafeCall(func() { req.result_callback(data) }, nil /*lock*/, s.userv.global_config.ErrorLogger())
		}
		if rtpc_delay != -1 {
			s.userv.register_delay(rtpc_delay)
		}
	}
	s.shutdown_chan <- 1
	s.userv = nil
}

type Rtp_proxy_client_stream struct {
	owner         sippy_types.RtpProxyClient
	_address      net.Addr
	nworkers      int
	workers       []*_RTPPLWorker
	delay_flt     sippy_math.RecFilter
	_is_local     bool
	wi            chan *rtpp_req_stream
	global_config sippy_conf.Config
}

func newRtp_proxy_client_stream(owner sippy_types.RtpProxyClient, global_config sippy_conf.Config, address net.Addr, bind_address *sippy_net.HostPort) (rtp_proxy_transport, error) {
	var err error
	if address == nil {
		address, err = net.ResolveUnixAddr("unix", "/var/run/rtpproxy.sock")
		if err != nil {
			return nil, err
		}
	}
	nworkers := 4
	if owner.GetOpts().GetNWorkers() != nil {
		nworkers = *owner.GetOpts().GetNWorkers()
	}
	s := &Rtp_proxy_client_stream{
		owner:         owner,
		_address:      address,
		nworkers:      nworkers,
		workers:       make([]*_RTPPLWorker, nworkers),
		delay_flt:     sippy_math.NewRecFilter(0.95, 0.25),
		wi:            make(chan *rtpp_req_stream, 1000),
		global_config: global_config,
	}
	if strings.HasPrefix(address.Network(), "unix") {
		s._is_local = true
	} else {
		s._is_local = false
	}
	//s.wi_available = Condition()
	//s.wi = []
	for i := 0; i < s.nworkers; i++ {
		s.workers[i] = newRTPPLWorker(s)
	}
	return s, nil
}

func (s *Rtp_proxy_client_stream) is_local() bool {
	return s._is_local
}

func (s *Rtp_proxy_client_stream) address() net.Addr {
	return s._address
}

func (s *Rtp_proxy_client_stream) send_command(command string, result_callback func(string)) {
	if command[len(command)-1] != '\n' {
		command += "\n"
	}
	s.wi <- &rtpp_req_stream{command, result_callback}
}

func (s *Rtp_proxy_client_stream) reconnect(address net.Addr, bind_addr *sippy_net.HostPort) {
	s.shutdown()
	s._address = address
	s.workers = make([]*_RTPPLWorker, s.nworkers)
	for i := 0; i < s.nworkers; i++ {
		s.workers[i] = newRTPPLWorker(s)
	}
	s.delay_flt = sippy_math.NewRecFilter(0.95, 0.25)
}

func (s *Rtp_proxy_client_stream) shutdown() {
	s.wi <- nil
	for _, rworker := range s.workers {
		<-rworker.shutdown_chan
	}
	s.workers = nil
	<-s.wi // take away the shutdown request
}

func (s *Rtp_proxy_client_stream) register_delay(rtpc_delay time.Duration) {
	s.delay_flt.Apply(rtpc_delay.Seconds())
}

func (s *Rtp_proxy_client_stream) get_rtpc_delay() float64 {
	return s.delay_flt.GetLastval()
}

/*
if __name__ == "__main__":
    from twisted.internet import reactor
    def display(*args):
        print args
        reactor.crash()
    r = Rtp_proxy_client_stream({"_sip_address":"1.2.3.4"})
    r.send_command("VF 123456", display, "abcd")
    reactor.run(installSignalHandlers = 1)
    r.shutdown()
*/
