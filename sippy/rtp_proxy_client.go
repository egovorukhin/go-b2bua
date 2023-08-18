package sippy

import (
	"bufio"
	"net"
	"strconv"
	"strings"

	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/types"
)

func NewRtpProxyClient(opts *rtpProxyClientOpts) sippy_types.RtpProxyClient {
	return NewRtp_proxy_client_base(nil, opts)
}

type Rtp_proxy_client_base struct {
	heir             sippy_types.RtpProxyClient
	opts             *rtpProxyClientOpts
	transport        rtp_proxy_transport
	online           bool
	sbind_supported  bool
	tnot_supported   bool
	copy_supported   bool
	stat_supported   bool
	wdnt_supported   bool
	caps_done        bool
	shut_down        bool
	active_sessions  int64
	sessions_created int64
	active_streams   int64
	preceived        int64
	ptransmitted     int64
}

type rtp_proxy_transport interface {
	address() net.Addr
	get_rtpc_delay() float64
	is_local() bool
	send_command(string, func(string))
	shutdown()
	reconnect(net.Addr, *sippy_net.HostPort)
}

func (s *Rtp_proxy_client_base) IsLocal() bool {
	return s.transport.is_local()
}

func (s *Rtp_proxy_client_base) IsOnline() bool {
	return s.online
}

func (s *Rtp_proxy_client_base) WdntSupported() bool {
	return s.wdnt_supported
}

func (s *Rtp_proxy_client_base) SBindSupported() bool {
	return s.sbind_supported
}

func (s *Rtp_proxy_client_base) TNotSupported() bool {
	return s.tnot_supported
}

func (s *Rtp_proxy_client_base) GetProxyAddress() string {
	return s.opts.proxy_address
}

func (s *Rtp_proxy_client_base) me() sippy_types.RtpProxyClient {
	if s.heir != nil {
		return s.heir
	}
	return s
}

func (s *Rtp_proxy_client_base) Address() net.Addr {
	return s.transport.address()
}

func NewRtp_proxy_client_base(heir sippy_types.RtpProxyClient, opts *rtpProxyClientOpts) *Rtp_proxy_client_base {
	return &Rtp_proxy_client_base{
		heir:            heir,
		caps_done:       false,
		shut_down:       false,
		opts:            opts,
		active_sessions: -1,
	}
}

func (s *Rtp_proxy_client_base) Start() error {
	var err error

	s.transport, err = s.opts.rtpp_class(s.me(), s.opts.config, s.opts.rtppaddr, s.opts.bind_address)
	if err != nil {
		return err
	}
	if !s.opts.no_version_check {
		s.version_check()
	} else {
		s.caps_done = true
		s.online = true
	}
	return nil
}

func (s *Rtp_proxy_client_base) SendCommand(cmd string, cb func(string)) {
	s.transport.send_command(cmd, cb)
}

func (s *Rtp_proxy_client_base) Reconnect(addr net.Addr, bind_addr *sippy_net.HostPort) {
	s.transport.reconnect(addr, bind_addr)
}

func (s *Rtp_proxy_client_base) version_check() {
	if s.shut_down {
		return
	}
	s.transport.send_command("V", s.version_check_reply)
}

func (s *Rtp_proxy_client_base) version_check_reply(version string) {
	if s.shut_down {
		return
	}
	if version == "20040107" {
		s.me().GoOnline()
	} else if s.online {
		s.me().GoOffline()
	} else {
		StartTimeoutWithSpread(s.version_check, nil, s.opts.hrtb_retr_ival, 1, s.opts.logger, 0.1)
	}
}

func (s *Rtp_proxy_client_base) heartbeat() {
	//print "heartbeat", s, s.address
	if s.shut_down {
		return
	}
	s.transport.send_command("Ib", s.heartbeat_reply)
}

func (s *Rtp_proxy_client_base) heartbeat_reply(stats string) {
	//print "heartbeat_reply", s.address, stats, s.online
	if s.shut_down || !s.online {
		return
	}
	if stats == "" {
		s.active_sessions = -1
		s.me().GoOffline()
	} else {
		sessions_created := int64(0)
		active_sessions := int64(0)
		active_streams := int64(0)
		preceived := int64(0)
		ptransmitted := int64(0)
		scanner := bufio.NewScanner(strings.NewReader(stats))
		for scanner.Scan() {
			line_parts := strings.SplitN(scanner.Text(), ":", 2)
			if len(line_parts) != 2 {
				continue
			}
			switch line_parts[0] {
			case "sessions created":
				sessions_created, _ = strconv.ParseInt(strings.TrimSpace(line_parts[1]), 10, 64)
			case "active sessions":
				active_sessions, _ = strconv.ParseInt(strings.TrimSpace(line_parts[1]), 10, 64)
			case "active streams":
				active_streams, _ = strconv.ParseInt(strings.TrimSpace(line_parts[1]), 10, 64)
			case "packets received":
				preceived, _ = strconv.ParseInt(strings.TrimSpace(line_parts[1]), 10, 64)
			case "packets transmitted":
				ptransmitted, _ = strconv.ParseInt(strings.TrimSpace(line_parts[1]), 10, 64)
			}
		}
		s.me().UpdateActive(active_sessions, sessions_created, active_streams, preceived, ptransmitted)
	}
	StartTimeoutWithSpread(s.heartbeat, nil, s.opts.hrtb_ival, 1, s.opts.logger, 0.1)
}

func (s *Rtp_proxy_client_base) GoOnline() {
	if s.shut_down {
		return
	}
	if !s.online {
		if !s.caps_done {
			newRtppCapsChecker(s)
			return
		}
		s.online = true
		s.heartbeat()
	}
}

func (s *Rtp_proxy_client_base) GoOffline() {
	if s.shut_down {
		return
	}
	//print "go_offline", s.address, s.online
	if s.online {
		s.online = false
		StartTimeoutWithSpread(s.version_check, nil, s.opts.hrtb_retr_ival, 1, s.opts.logger, 0.1)
	}
}

func (s *Rtp_proxy_client_base) UpdateActive(active_sessions, sessions_created, active_streams, preceived, ptransmitted int64) {
	s.sessions_created = sessions_created
	s.active_sessions = active_sessions
	s.active_streams = active_streams
	s.preceived = preceived
	s.ptransmitted = ptransmitted
}

func (s *Rtp_proxy_client_base) GetActiveSessions() int64 {
	return s.active_sessions
}

func (s *Rtp_proxy_client_base) GetActiveStreams() int64 {
	return s.active_streams
}

func (s *Rtp_proxy_client_base) GetPReceived() int64 {
	return s.preceived
}

func (s *Rtp_proxy_client_base) GetSessionsCreated() int64 {
	return s.sessions_created
}

func (s *Rtp_proxy_client_base) GetPTransmitted() int64 {
	return s.ptransmitted
}

func (s *Rtp_proxy_client_base) Shutdown() {
	if s.shut_down { // do not crash when shutdown() called twice
		return
	}
	s.shut_down = true
	s.transport.shutdown()
}

func (s *Rtp_proxy_client_base) IsShutDown() bool {
	return s.shut_down
}

func (s *Rtp_proxy_client_base) GetOpts() sippy_types.RtpProxyClientOpts {
	return s.opts
}

func (s *Rtp_proxy_client_base) GetRtpcDelay() float64 {
	return s.transport.get_rtpc_delay()
}

type rtppCapsChecker struct {
	caps_requested int
	caps_received  int
	rtpc           *Rtp_proxy_client_base
}

func newRtppCapsChecker(rtpc *Rtp_proxy_client_base) *rtppCapsChecker {
	s := &rtppCapsChecker{
		rtpc: rtpc,
	}
	rtpc.caps_done = false
	CAPSTABLE := []struct {
		vers string
		attr *bool
	}{
		{"20071218", &s.rtpc.copy_supported},
		{"20080403", &s.rtpc.stat_supported},
		{"20081224", &s.rtpc.tnot_supported},
		{"20090810", &s.rtpc.sbind_supported},
		{"20150617", &s.rtpc.wdnt_supported},
	}
	s.caps_requested = len(CAPSTABLE)
	for _, it := range CAPSTABLE {
		attr := it.attr // For some reason the it.attr cannot be passed into the following
		// function directly - the resulting value is always that of the
		// last 'it.attr' value.
		rtpc.transport.send_command("VF "+it.vers, func(res string) { s.caps_query_done(res, attr) })
	}
	return s
}

func (s *rtppCapsChecker) caps_query_done(result string, attr *bool) {
	s.caps_received += 1
	if result == "1" {
		*attr = true
	} else {
		*attr = false
	}
	if s.caps_received == s.caps_requested {
		s.rtpc.caps_done = true
		s.rtpc.me().GoOnline()
		s.rtpc = nil
	}
}
