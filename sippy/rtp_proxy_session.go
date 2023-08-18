package sippy

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"runtime"
	"strconv"
	"sync"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/types"
)

type Rtp_proxy_session struct {
	call_id           string
	from_tag          string
	to_tag            string
	_rtp_proxy_client sippy_types.RtpProxyClient
	max_index         int
	notify_socket     string
	notify_tag        string
	insert_nortpp     bool
	caller            _rtpps_side
	callee            _rtpps_side
	session_lock      sync.Locker
	config            sippy_conf.Config
	inflight_lock     sync.Mutex
	inflight_cmd      *rtpp_cmd
	rtpp_wi           chan *rtpp_cmd
}

type rtpproxy_update_result struct {
	rtpproxy_address string
	rtpproxy_port    string
	family           string
	sendonly         bool
}

type rtpp_cmd struct {
	cmd              string
	cb               func(string)
	rtp_proxy_client sippy_types.RtpProxyClient
}

func (s *rtpproxy_update_result) Address() string {
	return s.rtpproxy_address
}

func NewRtp_proxy_session(config sippy_conf.Config, rtp_proxy_clients []sippy_types.RtpProxyClient, call_id, from_tag, to_tag, notify_socket, notify_tag string, session_lock sync.Locker) (*Rtp_proxy_session, error) {
	s := &Rtp_proxy_session{
		notify_socket: notify_socket,
		notify_tag:    notify_tag,
		call_id:       call_id,
		from_tag:      from_tag,
		to_tag:        to_tag,
		insert_nortpp: false,
		max_index:     -1,
		session_lock:  session_lock,
		config:        config,
		rtpp_wi:       make(chan *rtpp_cmd, 50),
	}
	s.caller.otherside = &s.callee
	s.callee.otherside = &s.caller
	s.caller.owner = s
	s.callee.owner = s
	s.caller.session_exists = false
	s.callee.session_exists = false
	online_clients := []sippy_types.RtpProxyClient{}
	for _, cl := range rtp_proxy_clients {
		if cl.IsOnline() {
			online_clients = append(online_clients, cl)
		}
	}
	n := len(online_clients)
	if n == 0 {
		return nil, errors.New("No online RTP proxy client has been found")
	}
	if idx, err := rand.Int(rand.Reader, big.NewInt(int64(n))); err != nil {
		s._rtp_proxy_client = online_clients[0]
	} else {
		s._rtp_proxy_client = online_clients[idx.Int64()]
	}
	if s.call_id == "" {
		buf := make([]byte, 16)
		rand.Read(buf)
		s.call_id = hex.EncodeToString(buf)
	}
	if from_tag == "" {
		buf := make([]byte, 16)
		rand.Read(buf)
		s.from_tag = hex.EncodeToString(buf)
	}
	if to_tag == "" {
		buf := make([]byte, 16)
		rand.Read(buf)
		s.to_tag = hex.EncodeToString(buf)
	}
	s.caller.from_tag = s.from_tag
	s.caller.to_tag = s.to_tag
	s.callee.to_tag = s.from_tag
	s.callee.from_tag = s.to_tag
	runtime.SetFinalizer(s, rtp_proxy_session_destructor)
	return s, nil
}

/*
def version(s, result_callback):

	s.send_command("V", s.version_result, result_callback)

def version_result(s, result, result_callback):

	result_callback(result)
*/
func (s *Rtp_proxy_session) PlayCaller(prompt_name string, times int /*= 1*/, result_callback func(string) /*= nil*/, index int /*= 0*/) {
	s.caller._play(prompt_name, times, result_callback, index)
}

func (s *Rtp_proxy_session) send_command(cmd string, cb func(string)) {
	if rtp_proxy_client := s._rtp_proxy_client; rtp_proxy_client != nil {
		s.inflight_lock.Lock()
		defer s.inflight_lock.Unlock()
		new_cmd := &rtpp_cmd{cmd, cb, rtp_proxy_client}
		if s.inflight_cmd == nil {
			s.inflight_cmd = new_cmd
			rtp_proxy_client.SendCommand(cmd, s.cmd_done)
		} else {
			s.rtpp_wi <- new_cmd
		}
	}
}

func (s *Rtp_proxy_session) cmd_done(res string) {
	s.inflight_lock.Lock()
	done_cmd := s.inflight_cmd
	select {
	case s.inflight_cmd = <-s.rtpp_wi:
		s.inflight_cmd.rtp_proxy_client.SendCommand(s.inflight_cmd.cmd, s.cmd_done)
	default:
		s.inflight_cmd = nil
	}
	s.inflight_lock.Unlock()
	if done_cmd != nil && done_cmd.cb != nil {
		s.session_lock.Lock()
		done_cmd.cb(res)
		s.session_lock.Unlock()
	}
}

func (s *Rtp_proxy_session) StopPlayCaller(result_callback func(string) /*= nil*/, index int /*= 0*/) {
	s.caller._stop_play(result_callback, index)
}

func (s *Rtp_proxy_session) StartRecording(rname /*= nil*/ string, result_callback func(string) /*= nil*/, index int /*= 0*/) {
	if !s.caller.session_exists {
		s.caller.update("0.0.0.0", "0", func(*rtpproxy_update_result) { s._start_recording(rname, result_callback, index) }, "", index, "IP4")
		return
	}
	s._start_recording(rname, result_callback, index)
}

func (s *Rtp_proxy_session) _start_recording(rname string, result_callback func(string), index int) {
	if rname == "" {
		command := "R " + s.call_id + "-" + strconv.Itoa(index) + " " + s.from_tag + " " + s.to_tag
		s.send_command(command, func(r string) { s.command_result(r, result_callback) })
		return
	}
	command := "C " + s.call_id + "-" + strconv.Itoa(index) + " " + rname + ".a " + s.from_tag + " " + s.to_tag
	s.send_command(command, func(string) { s._start_recording1(rname, result_callback, index) })
}

func (s *Rtp_proxy_session) _start_recording1(rname string, result_callback func(string), index int) {
	command := "C " + s.call_id + "-" + strconv.Itoa(index) + " " + rname + ".o " + s.to_tag + " " + s.from_tag
	s.send_command(command, func(r string) { s.command_result(r, result_callback) })
}

func (s *Rtp_proxy_session) command_result(result string, result_callback func(string)) {
	//print "%s.command_result(%s)" % (id(s), result)
	if result_callback != nil {
		result_callback(result)
	}
}

func (s *Rtp_proxy_session) Delete() {
	if s._rtp_proxy_client == nil {
		return
	}
	for s.max_index >= 0 {
		command := "D " + s.call_id + "-" + strconv.Itoa(s.max_index) + " " + s.from_tag + " " + s.to_tag
		s.send_command(command, nil)
		s.max_index--
	}
	s._rtp_proxy_client = nil
}

func (s *Rtp_proxy_session) OnCallerSdpChange(sdp_body sippy_types.MsgBody, result_callback func(sippy_types.MsgBody)) error {
	return s.caller._on_sdp_change(sdp_body, result_callback)
}

func (s *Rtp_proxy_session) OnCalleeSdpChange(sdp_body sippy_types.MsgBody, result_callback func(sippy_types.MsgBody)) error {
	return s.callee._on_sdp_change(sdp_body, result_callback)
}

func rtp_proxy_session_destructor(s *Rtp_proxy_session) {
	s.Delete()
}

func (s *Rtp_proxy_session) CallerSessionExists() bool { return s.caller.session_exists }

func (s *Rtp_proxy_session) SetCallerLaddress(addr string) {
	s.caller.laddress = addr
}

func (s *Rtp_proxy_session) SetCallerRaddress(addr *sippy_net.HostPort) {
	s.caller.raddress = addr
}

func (s *Rtp_proxy_session) SetCalleeLaddress(addr string) {
	s.callee.laddress = addr
}

func (s *Rtp_proxy_session) SetCalleeRaddress(addr *sippy_net.HostPort) {
	s.callee.raddress = addr
}

func (s *Rtp_proxy_session) SetInsertNortpp(v bool) {
	s.insert_nortpp = v
}

func (s *Rtp_proxy_session) SetAfterCallerSdpChange(cb func(sippy_types.RtpProxyUpdateResult)) {
	s.caller.after_sdp_change = cb
}

func (s *Rtp_proxy_session) SBindSupported() (bool, error) {
	rtp_proxy_client := s._rtp_proxy_client
	if rtp_proxy_client == nil {
		return true, errors.New("the session already deleted")
	}
	return rtp_proxy_client.SBindSupported(), nil
}

func (s *Rtp_proxy_session) IsLocal() (bool, error) {
	rtp_proxy_client := s._rtp_proxy_client
	if rtp_proxy_client == nil {
		return true, errors.New("the session already deleted")
	}
	return rtp_proxy_client.IsLocal(), nil
}

func (s *Rtp_proxy_session) TNotSupported() (bool, error) {
	rtp_proxy_client := s._rtp_proxy_client
	if rtp_proxy_client == nil {
		return true, errors.New("the session already deleted")
	}
	return rtp_proxy_client.TNotSupported(), nil
}

func (s *Rtp_proxy_session) GetProxyAddress() (string, error) {
	rtp_proxy_client := s._rtp_proxy_client
	if rtp_proxy_client == nil {
		return "", errors.New("the session already deleted")
	}
	return rtp_proxy_client.GetProxyAddress(), nil
}
