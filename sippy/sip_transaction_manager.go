package sippy

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type sipTransactionManager struct {
	call_map             sippy_types.CallMap
	l4r                  *local4remote
	l1rcache             map[string]*sipTMRetransmitO
	l2rcache             map[string]*sipTMRetransmitO
	rcache_lock          sync.Mutex
	shutdown_chan        chan int
	config               sippy_conf.Config
	tclient              map[sippy_header.TID]sippy_types.ClientTransaction
	tclient_lock         sync.Mutex
	tserver              map[sippy_header.TID]sippy_types.ServerTransaction
	tserver_lock         sync.Mutex
	nat_traversal        bool
	req_consumers        map[string][]sippy_types.UA
	consumers_lock       sync.Mutex
	pass_t_to_cb         bool
	before_response_sent func(sippy_types.SipResponse)
	rtid2tid             map[sippy_header.RTID]*sippy_header.TID
	rtid2tid_lock        sync.Mutex
}

type sipTMRetransmitO struct {
	userv    sippy_net.Transport
	data     []byte
	address  *sippy_net.HostPort
	call_id  string
	lossemul int
}

func NewSipTransactionManager(config sippy_conf.Config, call_map sippy_types.CallMap) (*sipTransactionManager, error) {
	var err error

	s := &sipTransactionManager{
		call_map:      call_map,
		l1rcache:      make(map[string]*sipTMRetransmitO),
		l2rcache:      make(map[string]*sipTMRetransmitO),
		shutdown_chan: make(chan int),
		config:        config,
		tclient:       make(map[sippy_header.TID]sippy_types.ClientTransaction),
		tserver:       make(map[sippy_header.TID]sippy_types.ServerTransaction),
		nat_traversal: false,
		req_consumers: make(map[string][]sippy_types.UA),
		pass_t_to_cb:  false,
		rtid2tid:      make(map[sippy_header.RTID]*sippy_header.TID),
	}
	// Lock the lock here, otherwise we might get request in too
	// early for us to start processing it
	s.rcache_lock.Lock()
	defer s.rcache_lock.Unlock()
	s.l4r, err = NewLocal4Remote(config, s.handleIncoming)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			time.Sleep(32 * time.Second)
			s.rCachePurge()
		}
	}()
	return s, nil
}

func (s *sipTransactionManager) Run() {
	<-s.shutdown_chan
	s.l4r.shutdown()
}

func (s *sipTransactionManager) rCachePurge() {
	s.rcache_lock.Lock()
	defer s.rcache_lock.Unlock()
	s.l2rcache = s.l1rcache
	s.l1rcache = make(map[string]*sipTMRetransmitO)
	s.l4r.rotateCache()
}

var NETS_1918 = []struct {
	ip   net.IP
	mask net.IPMask
}{
	{net.IPv4(10, 0, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
	{net.IPv4(172, 16, 0, 0), net.IPv4Mask(255, 240, 0, 0)},
	{net.IPv4(192, 168, 0, 0), net.IPv4Mask(255, 255, 0, 0)},
}

func check1918(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip = ip.To4(); ip == nil {
		return false
	}
	for _, it := range NETS_1918 {
		if ip.Mask(it.mask).Equal(it.ip) {
			return true
		}
	}
	return false
}

func (s *sipTransactionManager) rcache_put(checksum string, entry *sipTMRetransmitO) {
	s.rcache_lock.Lock()
	defer s.rcache_lock.Unlock()
	s.rcache_put_no_lock(checksum, entry)
}

func (s *sipTransactionManager) rcache_put_no_lock(checksum string, entry *sipTMRetransmitO) {
	s.l1rcache[checksum] = entry
}

func (s *sipTransactionManager) rcache_get_no_lock(checksum string) (entry *sipTMRetransmitO, ok bool) {
	entry, ok = s.l1rcache[checksum]
	if ok {
		return
	}
	entry, ok = s.l2rcache[checksum]
	return
}

func (s *sipTransactionManager) rcache_set_call_id(checksum, call_id string) {
	s.rcache_lock.Lock()
	defer s.rcache_lock.Unlock()
	if it, ok := s.rcache_get_no_lock(checksum); ok {
		it.call_id = call_id
	} else {
		s.rcache_put_no_lock(checksum, &sipTMRetransmitO{
			userv:   nil,
			data:    nil,
			address: nil,
			call_id: call_id,
		})
	}
}

func (s *sipTransactionManager) logMsg(rtime *sippy_time.MonoTime, call_id string,
	direction string, address *sippy_net.HostPort, data []byte) {
	var ft string
	if direction == "SENDING" {
		ft = " message to "
	} else {
		ft = " message from "
	}
	msg := direction + ft + address.String() + ":\n" + string(data) + "\n"
	s.config.SipLogger().Write(rtime, call_id, msg)
}

func (s *sipTransactionManager) handleIncoming(data []byte, address *sippy_net.HostPort, server sippy_net.Transport, rtime *sippy_time.MonoTime) {
	if len(data) < 32 {
		//s.logMsg(rtime, retrans.call_id, "RECEIVED", address, data)
		//s.logError("The message is too short from " + address.String() + ":\n" + string(data))
		return
	}
	sum := md5.Sum(data)
	checksum := hex.EncodeToString(sum[:])
	s.rcache_lock.Lock()
	retrans, ok := s.rcache_get_no_lock(checksum)
	if ok {
		s.rcache_lock.Unlock()
		s.logMsg(rtime, retrans.call_id, "RECEIVED", address, data)
		if retrans.data == nil {
			return
		}
		s.transmitData(retrans.userv, retrans.data, retrans.address, "", retrans.call_id, 0)
		return
	}
	s.rcache_put_no_lock(checksum, &sipTMRetransmitO{
		userv:   nil,
		data:    nil,
		address: nil,
	})
	s.rcache_lock.Unlock()
	if string(data[:7]) == "SIP/2.0" {
		s.process_response(rtime, data, checksum, address, server)
	} else {
		s.process_request(rtime, data, checksum, address, server)
	}
}

func (s *sipTransactionManager) process_response(rtime *sippy_time.MonoTime, data []byte, checksum string, address *sippy_net.HostPort, server sippy_net.Transport) {
	var resp *sipResponse
	var err error
	var tid *sippy_header.TID
	var contact *sippy_header.SipAddress

	resp, err = ParseSipResponse(data, rtime, s.config)
	if err != nil {
		s.logMsg(rtime, "", "RECEIVED", address, data)
		s.logBadMessage("can't parse SIP response from "+address.String()+":"+err.Error(), data)
		return
	}
	tid, err = resp.GetTId(true /*wCSM*/, true /*wBRN*/, false /*wTTG*/)
	if err != nil {
		s.logMsg(rtime, "", "RECEIVED", address, data)
		s.logBadMessage("can't parse SIP response from "+address.String()+":"+err.Error(), data)
		return
	}
	s.logMsg(rtime, tid.CallId, "RECEIVED", address, data)

	if resp.scode < 100 || resp.scode > 999 {
		s.logBadMessage("invalid status code in SIP response"+address.String()+":\n"+string(data), data)
		s.rcache_set_call_id(checksum, tid.CallId)
		return
	}
	s.tclient_lock.Lock()
	t, ok := s.tclient[*tid]
	s.tclient_lock.Unlock()
	if !ok {
		//print 'no transaction with tid of %s in progress' % str(tid)
		if len(resp.vias) > 1 {
			var via0 *sippy_header.SipViaBody

			via0, err = resp.vias[0].GetBody()
			if err != nil {
				s.logBadMessage(err.Error(), data)
				return
			}
			taddr := via0.GetTAddr(s.config)
			if taddr.Port.String() != s.config.SipPort().String() {
				if len(resp.contacts) == 0 {
					s.logBadMessage("OnUdpPacket: no Contact: in SIP response", data)
					return
				}
				if !resp.contacts[0].Asterisk {
					contact, err = resp.contacts[0].GetBody(s.config)
					if err != nil {
						s.logBadMessage(err.Error(), data)
						return
					}
					curl := contact.GetUrl()
					//print 'curl.host = %s, curl.port = %d,  address[1] = %d' % (curl.host, curl.port, address[1])
					if check1918(curl.Host.String()) || curl.Port.String() != address.Port.String() {
						curl.Host = sippy_net.NewMyAddress(taddr.Host.String())
						curl.Port = sippy_net.NewMyPort(taddr.Port.String())
					}
				}
				data = resp.Bytes()
				call_id := ""
				if resp.call_id != nil {
					call_id = resp.call_id.CallId
				}
				s.transmitData(server, data, taddr, checksum, call_id, 0)
			}
		}
		s.rcache_set_call_id(checksum, tid.CallId)
		return
	}
	t.Lock()
	defer t.Unlock()
	if s.nat_traversal && len(resp.contacts) > 0 && !resp.contacts[0].Asterisk && !check1918(t.GetHost()) {
		contact, err = resp.contacts[0].GetBody(s.config)
		if err != nil {
			s.logBadMessage(err.Error(), data)
			return
		}
		curl := contact.GetUrl()
		if check1918(curl.Host.String()) {
			host, port := address.Host.String(), address.Port.String()
			curl.Host, curl.Port = sippy_net.NewMyAddress(host), sippy_net.NewMyPort(port)
		}
	}
	host, port := address.Host.String(), address.Port.String()
	resp.source = sippy_net.NewHostPort(host, port)
	sippy_utils.SafeCall(func() { t.IncomingResponse(resp, checksum) }, nil, s.config.ErrorLogger())
}

func (s *sipTransactionManager) process_request(rtime *sippy_time.MonoTime, data []byte, checksum string, address *sippy_net.HostPort, server sippy_net.Transport) {
	var req *sipRequest
	var err error
	var tids []*sippy_header.TID
	var via0 *sippy_header.SipViaBody

	req, err = ParseSipRequest(data, rtime, s.config)
	if err != nil {
		switch errt := err.(type) {
		case *ESipParseException:
			if errt.sip_response != nil {
				s.transmitMsg(server, errt.sip_response, address, checksum, errt.sip_response.GetCallId().CallId)
			}
		}
		s.logMsg(rtime, "", "RECEIVED", address, data)
		s.logBadMessage("can't parse SIP request from "+address.String()+": "+err.Error(), data)
		return
	}
	tids, err = req.getTIds()
	if err != nil {
		s.logMsg(rtime, "", "RECEIVED", address, data)
		s.logBadMessage(err.Error(), data)
		return
	}
	s.logMsg(rtime, tids[0].CallId, "RECEIVED", address, data)
	via0, err = req.vias[0].GetBody()
	if err != nil {
		s.logBadMessage(err.Error(), data)
		return
	}
	ahost, aport := via0.GetAddr(s.config)
	rhost, rport := address.Host.String(), address.Port.String()
	if s.nat_traversal && rport != aport && check1918(ahost) {
		req.nated = true
	}
	if ahost != rhost {
		via0.SetReceived(rhost)
	}
	if via0.HasRPort() || req.nated {
		via0.SetRPort(&rport)
	}
	if s.nat_traversal && len(req.contacts) > 0 && !req.contacts[0].Asterisk && len(req.vias) == 1 {
		var contact *sippy_header.SipAddress

		contact, err = req.contacts[0].GetBody(s.config)
		if err != nil {
			s.logBadMessage("Bad Contact: "+err.Error(), data)
			return
		}
		curl := contact.GetUrl()
		if check1918(curl.Host.String()) {
			tmp_host, tmp_port := address.Host.String(), address.Port.String()
			curl.Port = sippy_net.NewMyPort(tmp_port)
			curl.Host = sippy_net.NewMyAddress(tmp_host)
			req.nated = true
		}
	}
	host, port := address.Host.String(), address.Port.String()
	req.source = sippy_net.NewHostPort(host, port)
	s.incomingRequest(req, checksum, tids, server, data)
}

// 1. Client transaction methods
func (s *sipTransactionManager) CreateClientTransaction(req sippy_types.SipRequest,
	resp_receiver sippy_types.ResponseReceiver, session_lock sync.Locker,
	laddress *sippy_net.HostPort, userv sippy_net.Transport,
	eh []sippy_header.SipHeader, req_out_cb func(sippy_types.SipRequest)) (sippy_types.ClientTransaction, error) {
	var tid *sippy_header.TID
	var err error
	var t *clientTransaction

	if s == nil {
		return nil, errors.New("BUG: Attempt to initiate transaction from terminated dialog!!!")
	}
	target := req.GetTarget()
	if userv == nil {
		var uv sippy_net.Transport
		if laddress != nil {
			uv = s.l4r.getServer(laddress /*is_local =*/, true)
		}
		if uv == nil {
			uv = s.l4r.getServer(target /*is_local =*/, false)
		}
		if uv != nil {
			userv = uv
		}
	}
	if userv == nil {
		return nil, errors.New("BUG: cannot get userv from local4remote!!!")
	}
	tid, err = req.GetTId(true /*wCSM*/, true /*wBRN*/, false /*wTTG*/)
	if err != nil {
		return nil, err
	}
	s.tclient_lock.Lock()
	if _, ok := s.tclient[*tid]; ok {
		s.tclient_lock.Unlock()
		return nil, errors.New("BUG: Attempt to initiate transaction with the same TID as existing one!!!")
	}
	data := []byte(req.LocalStr(userv.GetLAddress(), false /* compact */))
	t, err = NewClientTransactionObj(req, tid, userv, data, s, resp_receiver, session_lock, target, eh, req_out_cb)
	if err != nil {
		return nil, err
	}
	s.tclient[*tid] = t
	s.tclient_lock.Unlock()
	return t, nil
}

func (s *sipTransactionManager) BeginClientTransaction(req sippy_types.SipRequest, tr sippy_types.ClientTransaction) {
	tr.StartTimers()
	tr.BeforeRequestSent(req)
	tr.TransmitData()
}

func (s *sipTransactionManager) BeginNewClientTransaction(req sippy_types.SipRequest, resp_receiver sippy_types.ResponseReceiver, session_lock sync.Locker, laddress *sippy_net.HostPort, userv sippy_net.Transport, req_out_cb func(sippy_types.SipRequest)) {
	tr, err := s.CreateClientTransaction(req, resp_receiver, session_lock, laddress, userv, nil, req_out_cb)
	if err != nil {
		s.config.ErrorLogger().Error(err.Error())
	} else {
		s.BeginClientTransaction(req, tr)
	}
}

// 2. Server transaction methods
func (s *sipTransactionManager) incomingRequest(req *sipRequest, checksum string, tids []*sippy_header.TID, server sippy_net.Transport, data []byte) {
	var tid *sippy_header.TID
	var rtid *sippy_header.RTID
	var err error

	s.tclient_lock.Lock()
	for _, tid = range tids {
		if _, ok := s.tclient[*tid]; ok {
			var via0 *sippy_header.SipViaBody

			s.tclient_lock.Unlock()
			resp := req.GenResponse(482, "Loop Detected" /*body*/, nil /*server*/, nil)
			via0, err = resp.GetVias()[0].GetBody()
			if err != nil {
				s.logBadMessage("cannot parse via: "+err.Error(), data)
			} else {
				hostPort := via0.GetTAddr(s.config)
				s.transmitMsg(server, resp, hostPort, checksum, tid.CallId)
			}
			return
		}
	}
	s.tclient_lock.Unlock()
	switch req.GetMethod() {
	case "ACK":
		tid, err = req.GetTId(false /*wCSM*/, false /*wBRN*/, true /*wTTG*/)
	case "PRACK":
		if rtid, err = req.GetRTId(); err != nil {
			s.logBadMessage("cannot get transaction ID: "+err.Error(), data)
			return
		}
		s.rtid2tid_lock.Lock()
		tid = s.rtid2tid[*rtid]
		s.rtid2tid_lock.Unlock()
	default:
		tid, err = req.GetTId(false /*wCSM*/, true /*wBRN*/, false /*wTTG*/)
		if err != nil {
			s.logBadMessage("cannot get transaction ID: "+err.Error(), data)
			return
		}
	}
	if tid == nil {
		s.logBadMessage("cannot get transaction ID: ", data)
		return
	}
	s.tserver_lock.Lock()
	t, ok := s.tserver[*tid]
	if ok {
		s.tserver_lock.Unlock()
		sippy_utils.SafeCall(func() { t.IncomingRequest(req, checksum) }, t, s.config.ErrorLogger())
		return
	}
	switch req.GetMethod() {
	case "ACK":
		s.tserver_lock.Unlock()
		// Some ACK that doesn't match any existing transaction.
		// Drop and forget it - upper layer is unlikely to be interested
		// to seeing this anyway.
		//println("unmatched ACK transaction - ignoring")
		s.rcache_set_call_id(checksum, tid.CallId)
	case "PRACK":
		// Some ACK that doesn't match any existing transaction.
		// Drop and forget it - upper layer is unlikely to be interested
		// to seeing this anyway.
		//print(datetime.now(), 'unmatched PRACK transaction - 481\'ing')
		//print(datetime.now(), 'rtid: %s, tid: %s, s.tserver: %s' % (str(rtid), str(tid), \
		//  str(s.tserver)))
		//sys.stdout.flush()
		via0, err := req.GetVias()[0].GetBody()
		if err != nil {
			s.logBadMessage("Cannot parse Via: "+err.Error(), data)
			return
		}
		resp := req.GenResponse(481, "Huh?" /*body*/, nil /*server*/, nil)
		s.transmitMsg(server, resp, via0.GetTAddr(s.config), checksum, tid.CallId)
	case "CANCEL":
		var via0 *sippy_header.SipViaBody

		s.tserver_lock.Unlock()
		resp := req.GenResponse(481, "Call Leg/Transaction Does Not Exist" /*body*/, nil /*server*/, nil)
		via0, err = resp.GetVias()[0].GetBody()
		if err != nil {
			s.logBadMessage("Cannot parse Via: "+err.Error(), data)
			return
		}
		s.transmitMsg(server, resp, via0.GetTAddr(s.config), checksum, tid.CallId)
	default:
		s.new_server_transaction(server, req, tid, checksum)
	}
}

func (s *sipTransactionManager) new_server_transaction(server sippy_net.Transport, req *sipRequest, tid *sippy_header.TID, checksum string) {
	var t sippy_types.ServerTransaction
	var err error

	/* Here the tserver_lock is already locked */
	var rval *sippy_types.UaContext = nil
	//print 'new transaction', req.GetMethod()
	userv := server
	if server.GetLAddress().Host.String() == "0.0.0.0" || server.GetLAddress().Host.String() == "[::]" {
		// For messages received on the wildcard interface find
		// or create more specific server.
		userv = s.l4r.getServer(req.GetSource() /*is_local*/, false)
		if userv == nil {
			s.logError("BUG! cannot create more specific server for transaction")
			userv = server
		}
	}
	t, err = NewServerTransaction(req, checksum, tid, userv, s)
	if err != nil {
		s.logError("cannot create server transaction: " + err.Error())
		return
	}
	t.Lock()
	defer t.Unlock()
	s.tserver[*tid] = t
	s.tserver_lock.Unlock()
	t.StartTimers()
	s.consumers_lock.Lock()
	consumers, ok := s.req_consumers[tid.CallId]
	var ua sippy_types.UA
	if ok {
		for _, c := range consumers {
			if c.IsYours(req /*br0k3n_to =*/, false) {
				ua = c
				break
			}
		}
	}
	s.consumers_lock.Unlock()
	if ua != nil {
		t.UpgradeToSessionLock(ua.GetSessionLock())
		sippy_utils.SafeCall(func() { rval = ua.RecvRequest(req, t) }, nil, s.config.ErrorLogger())
	} else {
		if s.call_map == nil {
			s.rcache_put(checksum, &sipTMRetransmitO{
				userv:   nil,
				data:    nil,
				address: nil,
			})
			return
		}
		var req_receiver sippy_types.RequestReceiver
		var resp sippy_types.SipResponse
		sippy_utils.SafeCall(func() { ua, req_receiver, resp = s.call_map.OnNewDialog(req, t) }, nil, s.config.ErrorLogger())
		if resp != nil {
			t.SendResponse(resp, false, nil)
			return
		} else {
			if ua != nil {
				t.UpgradeToSessionLock(ua.GetSessionLock())
			}
			if req_receiver != nil {
				rval = req_receiver.RecvRequest(req, t)
			}
		}
	}
	if rval == nil {
		if !t.TimersAreActive() {
			s.tserver_del(tid)
			t.Cleanup()
		}
	} else {
		t.SetCancelCB(rval.CancelCB)
		t.SetNoackCB(rval.NoAckCB)
		if rval.Response != nil {
			t.SendResponse(rval.Response, false, nil)
		}
	}
}

func (s *sipTransactionManager) RegConsumer(consumer sippy_types.UA, call_id string) {
	s.consumers_lock.Lock()
	defer s.consumers_lock.Unlock()
	consumers, ok := s.req_consumers[call_id]
	if !ok {
		consumers = make([]sippy_types.UA, 0)
	}
	consumers = append(consumers, consumer)
	s.req_consumers[call_id] = consumers
}

func (s *sipTransactionManager) UnregConsumer(consumer sippy_types.UA, call_id string) {
	// Usually there will be only one consumer per call_id, so that
	// optimize management for this case
	consumer.OnUnregister()
	s.consumers_lock.Lock()
	defer s.consumers_lock.Unlock()
	consumers, ok := s.req_consumers[call_id]
	if !ok {
		return
	}
	delete(s.req_consumers, call_id)
	if len(consumers) > 1 {
		for idx, c := range consumers {
			if c == consumer {
				consumers[idx] = nil
				consumers = append(consumers[:idx], consumers[idx+1:]...)
				break
			}
		}
		s.req_consumers[call_id] = consumers
	}
}

func (s *sipTransactionManager) SendResponse(resp sippy_types.SipResponse, lock bool, ack_cb func(sippy_types.SipRequest)) {
	s.SendResponseWithLossEmul(resp, lock, ack_cb, 0)
}

func (s *sipTransactionManager) SendResponseWithLossEmul(resp sippy_types.SipResponse, lock bool, ack_cb func(sippy_types.SipRequest), lossemul int) {
	//print s.tserver
	tid, err := resp.GetTId(false /*wCSM*/, true /*wBRN*/, false /*wTTG*/)
	if err != nil {
		s.logError("Cannot get transaction ID for server transaction: " + err.Error())
		return
	}
	s.tserver_lock.Lock()
	t, ok := s.tserver[*tid]
	s.tserver_lock.Unlock()
	if ok {
		if lock {
			t.Lock()
			defer t.Unlock()
		}
		t.SendResponseWithLossEmul(resp /*retrans*/, false, ack_cb, lossemul)
	} else {
		s.logError("Cannot get server transaction")
		return
	}
}

func (s *sipTransactionManager) transmitMsg(userv sippy_net.Transport, msg sippy_types.SipMsg, address *sippy_net.HostPort, cachesum string, call_id string) {
	data := msg.LocalStr(userv.GetLAddress(), false /*compact*/)
	s.transmitData(userv, []byte(data), address, cachesum, call_id, 0)
}

func (s *sipTransactionManager) transmitData(userv sippy_net.Transport, data []byte, address *sippy_net.HostPort, cachesum, call_id string, lossemul int /*=0*/) {
	s.transmitDataWithCb(userv, data, address, cachesum, call_id, lossemul, nil)
}

func (s *sipTransactionManager) transmitDataWithCb(userv sippy_net.Transport, data []byte, address *sippy_net.HostPort, cachesum, call_id string, lossemul int /*=0*/, on_complete func()) {
	logop := "SENDING"
	if lossemul == 0 {
		userv.SendToWithCb(data, address, on_complete)
	} else {
		logop = "DISCARDING"
	}
	s.logMsg(nil, call_id, logop, address, data)
	if len(cachesum) > 0 {
		if lossemul > 0 {
			lossemul--
		}
		s.rcache_put(cachesum, &sipTMRetransmitO{
			userv:    userv,
			data:     data,
			address:  address.GetCopy(),
			call_id:  call_id,
			lossemul: lossemul,
		})
	}
}

func (s *sipTransactionManager) logError(msg string) {
	s.config.ErrorLogger().Error(msg)
}

func (s *sipTransactionManager) logBadMessage(msg string, data []byte) {
	s.config.ErrorLogger().Error(msg)
	arr := strings.Split(string(data), "\n")
	for _, l := range arr {
		s.config.ErrorLogger().Error(l)
	}
}

func (s *sipTransactionManager) tclient_del(tid *sippy_header.TID) {
	if tid == nil {
		return
	}
	s.tclient_lock.Lock()
	defer s.tclient_lock.Unlock()
	delete(s.tclient, *tid)
}

func (s *sipTransactionManager) tserver_del(tid *sippy_header.TID) {
	if tid == nil {
		return
	}
	s.tserver_lock.Lock()
	defer s.tserver_lock.Unlock()
	delete(s.tserver, *tid)
}

func (s *sipTransactionManager) tserver_replace(old_tid, new_tid *sippy_header.TID, t sippy_types.ServerTransaction) {
	s.tserver_lock.Lock()
	defer s.tserver_lock.Unlock()
	delete(s.tserver, *old_tid)
	s.tserver[*new_tid] = t
}

func (s *sipTransactionManager) Shutdown() {
	s.shutdown_chan <- 1
}

func (s *sipTransactionManager) beforeResponseSent(resp sippy_types.SipResponse) {
	if s.before_response_sent != nil {
		s.before_response_sent(resp)
	}
}

func (s *sipTransactionManager) SetBeforeResponseSent(cb func(sippy_types.SipResponse)) {
	s.before_response_sent = cb
}

func (s *sipTransactionManager) rtid_replace(ik *sippy_header.RTID, old_tid, new_tid *sippy_header.TID) {
	if saved_tid, ok := s.rtid2tid[*ik]; ok && *saved_tid == *old_tid {
		s.rtid2tid_lock.Lock()
		defer s.rtid2tid_lock.Unlock()
		s.rtid2tid[*ik] = new_tid
	}
}

func (s *sipTransactionManager) rtid_del(key *sippy_header.RTID) {
	s.rtid2tid_lock.Lock()
	defer s.rtid2tid_lock.Unlock()
	delete(s.rtid2tid, *key)
}

func (s *sipTransactionManager) rtid_put(key *sippy_header.RTID, value *sippy_header.TID) {
	s.rtid2tid_lock.Lock()
	defer s.rtid2tid_lock.Unlock()
	s.rtid2tid[*key] = value
}
