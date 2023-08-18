package sippy

import (
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type provInFlight struct {
	t    *Timeout
	rtid *sippy_header.RTID
}

type serverTransaction struct {
	*baseTransaction
	lock                 sync.Mutex
	checksum             string
	teD                  *Timeout
	teE                  *Timeout
	r487                 sippy_types.SipResponse
	cancel_cb            func(*sippy_time.MonoTime, sippy_types.SipRequest)
	method               string
	server               *sippy_header.SipServer
	noack_cb             func(*sippy_time.MonoTime)
	branch               string
	session_lock         sync.Locker
	expires              time.Duration
	ack_cb               func(sippy_types.SipRequest)
	before_response_sent func(sippy_types.SipResponse)
	prov_inflight        *provInFlight
	prov_inflight_lock   sync.Mutex
	prack_cb             func(sippy_types.SipRequest, sippy_types.SipResponse)
	noprack_cb           func(*sippy_time.MonoTime)
	rseq                 *sippy_header.SipRSeq
	pr_rel               bool
}

func NewServerTransaction(req sippy_types.SipRequest, checksum string, tid *sippy_header.TID, userv sippy_net.Transport, sip_tm *sipTransactionManager) (sippy_types.ServerTransaction, error) {
	needack := false
	var r487 sippy_types.SipResponse
	var branch string
	var expires time.Duration = 0
	method := req.GetMethod()
	if method == "INVITE" {
		via0, err := req.GetVias()[0].GetBody()
		if err != nil {
			return nil, err
		}
		needack = true
		r487 = req.GenResponse(487, "Request Terminated" /*body*/, nil /*server*/, nil)
		branch = via0.GetBranch()
		expires = 300 * time.Second
		if req.GetExpires() != nil && req.GetExpires().Number > 0 {
			expires = time.Duration(req.GetExpires().Number) * time.Second
		}
	}
	s := &serverTransaction{
		method:   method,
		checksum: checksum,
		r487:     r487,
		branch:   branch,
		expires:  expires,
		pr_rel:   false,
		//prov_inflight   : nil,
		rseq: sippy_header.NewSipRSeq(),
	}
	s.baseTransaction = newBaseTransaction(s, tid, userv, sip_tm, nil, nil, needack)
	return s, nil
}

func (s *serverTransaction) StartTimers() {
	if s.expires > 0 {
		s.startTeE(s.expires)
	}
}

func (s *serverTransaction) Cleanup() {
	s.cleanup()
}

func (s *serverTransaction) cleanup() {
	s.baseTransaction.cleanup()
	s.r487 = nil
	s.cancel_cb = nil
	if s.teD != nil {
		s.teD.Cancel()
		s.teD = nil
	}
	if s.teE != nil {
		s.teE.Cancel()
		s.teE = nil
	}
	s.noack_cb = nil
	s.ack_cb = nil
	s.prack_cb = nil
	s.noprack_cb = nil
}

func (s *serverTransaction) startTeE(t time.Duration) {
	s.teE = StartTimeout(s.timerE, s, t, 1, s.logger)
}

func (s *serverTransaction) cancelTeE() {
	if s.teE != nil {
		s.teE.Cancel()
		s.teE = nil
	}
}

func (s *serverTransaction) cancelTeD() {
	if s.teD != nil {
		s.teD.Cancel()
		s.teD = nil
	}
}

func (s *serverTransaction) startTeD() {
	if s.teD != nil {
		s.teD.Cancel()
	}
	s.teD = StartTimeout(s.timerD, s, 32*time.Second, 1, s.logger)
}

func (s *serverTransaction) timerD() {
	sip_tm := s.sip_tm
	if sip_tm == nil {
		return
	}
	//print("timerD", t, t.GetTID())
	if s.noack_cb != nil && s.state != CONFIRMED {
		s.noack_cb(nil)
	}
	sip_tm.tserver_del(s.tid)
	s.cleanup()
}

func (s *serverTransaction) timerE() {
	if s.sip_tm == nil {
		return
	}
	//print("timerE", t.GetTID())
	s.cancelTeE()
	if s.state == TRYING || s.state == RINGING {
		if s.r487 != nil {
			s.r487.SetSCodeReason("Request Expired")
		}
		s.doCancel( /*rtime*/ nil /*req*/, nil)
	}
}

func (s *serverTransaction) SetCancelCB(cancel_cb func(*sippy_time.MonoTime, sippy_types.SipRequest)) {
	s.cancel_cb = cancel_cb
}

func (s *serverTransaction) SetNoackCB(noack_cb func(*sippy_time.MonoTime)) {
	s.noack_cb = noack_cb
}

func (s *serverTransaction) doCancel(rtime *sippy_time.MonoTime, req sippy_types.SipRequest) {
	if rtime == nil {
		rtime, _ = sippy_time.NewMonoTime()
	}
	if s.r487 != nil {
		s.SendResponse(s.r487, true, nil)
	}
	if s.cancel_cb != nil {
		s.cancel_cb(rtime, req)
	}
}

func (s *serverTransaction) IncomingRequest(req sippy_types.SipRequest, checksum string) {
	sip_tm := s.sip_tm
	if sip_tm == nil {
		return
	}
	//println("existing transaction")
	switch req.GetMethod() {
	case s.method:
		// Duplicate received, check that we have sent any response on this
		// request already
		if s.data != nil && len(s.data) > 0 {
			sip_tm.transmitData(s.userv, s.data, s.address, checksum, s.tid.CallId, 0)
		}
	case "CANCEL":
		// RFC3261 says that we have to reply 200 OK in all cases if
		// there is such transaction
		resp := req.GenResponse(200, "OK" /*body*/, nil, s.server)
		via0, err := resp.GetVias()[0].GetBody()
		if err != nil {
			s.logger.Debug("error parsing Via: " + err.Error())
			return
		}
		sip_tm.transmitMsg(s.userv, resp, via0.GetTAddr(sip_tm.config), checksum, s.tid.CallId)
		if s.state == TRYING || s.state == RINGING {
			s.doCancel(req.GetRtime(), req)
		}
	case "ACK":
		if s.state == COMPLETED {
			s.state = CONFIRMED
			s.cancelTeA()
			s.prov_inflight_lock.Lock()
			if s.prov_inflight == nil {
				// We have done with the transaction, no need to wait for timeout
				s.cancelTeD()
				sip_tm.tserver_del(s.tid)
			}
			if s.ack_cb != nil {
				s.ack_cb(req)
			}
			sip_tm.rcache_set_call_id(checksum, s.tid.CallId)
			if s.prov_inflight == nil {
				s.cleanup()
			}
			s.prov_inflight_lock.Unlock()
		}
	case "PRACK":
		var resp sippy_types.SipResponse
		rskey, err := req.GetRTId()
		if err != nil {
			s.logger.Debug("Cannot get rtid: " + err.Error())
			return
		}
		s.prov_inflight_lock.Lock()
		if s.prov_inflight != nil && *s.prov_inflight.rtid == *rskey {
			s.prov_inflight.t.Cancel()
			s.prov_inflight = nil
			s.prov_inflight_lock.Unlock()
			sip_tm.rtid_del(rskey)
			resp = req.GenResponse(200, "OK", nil /*body*/, nil /*server*/)
			if s.prack_cb != nil {
				s.prack_cb(req, resp)
			}
		} else {
			s.prov_inflight_lock.Unlock()
			//print('rskey: %s, prov_inflight: %s' % (str(rskey), str(s.prov_inflight)))
			resp = req.GenResponse(481, "Huh?", nil /*body*/, nil /*server*/)
		}
		via0, err := resp.GetVias()[0].GetBody()
		if err != nil {
			s.logger.Debug("error parsing Via: " + err.Error())
			return
		}
		sip_tm.transmitMsg(s.userv, resp, via0.GetTAddr(sip_tm.config), checksum, s.tid.CallId)
	}
}

func (s *serverTransaction) SendResponse(resp sippy_types.SipResponse, retrans bool, ack_cb func(sippy_types.SipRequest)) {
	s.SendResponseWithLossEmul(resp, retrans, ack_cb, 0)
}

func (s *serverTransaction) SendResponseWithLossEmul(resp sippy_types.SipResponse, retrans bool, ack_cb func(sippy_types.SipRequest), lossemul int) {
	var via0 *sippy_header.SipViaBody
	var err error

	sip_tm := s.sip_tm
	if sip_tm == nil {
		return
	}
	if s.state != TRYING && s.state != RINGING && !retrans {
		s.logger.Error("BUG: attempt to send reply on already finished transaction!!!")
	}
	scode := resp.GetSCodeNum()
	if scode > 100 {
		to, err := resp.GetTo().GetBody(sip_tm.config)
		if err != nil {
			s.logger.Debug("error parsing To: " + err.Error())
			return
		}
		if to.GetTag() == "" {
			to.GenTag()
		}
	}
	if s.pr_rel && scode > 100 && scode < 200 {
		rseq := s.rseq.GetCopy()
		rseq_body, err := s.rseq.GetBody()
		if err != nil {
			s.logger.Debug("error parsing RSeq: " + err.Error())
			return
		}
		rseq_body.Number++
		resp.AppendHeader(rseq)
		resp.AppendHeader(sippy_header.CreateSipRequire("100rel")[0])
		tid, err := resp.GetTId(false /*wCSM*/, true /*wBRN*/, false /*wTTG*/)
		if err != nil {
			s.logger.Debug("Cannot get tid: " + err.Error())
			return
		}
		rtid, err := resp.GetRTId()
		if err != nil {
			s.logger.Debug("Cannot get rtid: " + err.Error())
			return
		}
		if lossemul > 0 {
			lossemul -= 1
		}
		s.prov_inflight_lock.Lock()
		if s.prov_inflight == nil {
			timeout := 500 * time.Millisecond
			s.prov_inflight = &provInFlight{
				t:    StartTimeout(func() { s.retrUasResponse(timeout, lossemul) }, s, timeout, 1, s.logger),
				rtid: rtid,
			}
		} else {
			s.logger.Error("Attempt to start new PRACK timeout while the old one is still active")
		}
		s.prov_inflight_lock.Unlock()
		sip_tm.rtid_put(rtid, tid)
	}
	sip_tm.beforeResponseSent(resp)
	s.data = []byte(resp.LocalStr(s.userv.GetLAddress() /*compact*/, false))
	via0, err = resp.GetVias()[0].GetBody()
	if err != nil {
		s.logger.Debug("error parsing Via: " + err.Error())
		return
	}
	s.address = via0.GetTAddr(sip_tm.config)
	need_cleanup := false
	if resp.GetSCodeNum() < 200 {
		s.state = RINGING
	} else {
		s.state = COMPLETED
		s.cancelTeE()
		if s.needack {
			// Schedule removal of the transaction
			s.ack_cb = ack_cb
			s.startTeD()
			if resp.GetSCodeNum() >= 200 {
				// Black magick to allow proxy send us another INVITE
				// same branch and From tag. Use To tag to match
				// ACK transaction after this point. Branch tag in ACK
				// could differ as well.
				tid := s.tid
				if tid != nil {
					to, err := resp.GetTo().GetBody(sip_tm.config)
					if err != nil {
						s.logger.Debug("error parsing To: " + err.Error())
						return
					}
					old_tid := *tid // copy
					tid.Branch = ""
					tid.ToTag = to.GetTag()
					s.prov_inflight_lock.Lock()
					if s.prov_inflight != nil {
						sip_tm.rtid_replace(s.prov_inflight.rtid, &old_tid, tid)
					}
					sip_tm.tserver_replace(&old_tid, tid, s)
					s.prov_inflight_lock.Unlock()
				}
			}
			// Install retransmit timer if necessary
			s.tout = time.Duration(0.5 * float64(time.Second))
			s.startTeA()
		} else {
			// We have done with the transaction
			sip_tm.tserver_del(s.tid)
			need_cleanup = true
		}
	}
	if s.before_response_sent != nil {
		s.before_response_sent(resp)
	}
	sip_tm.transmitData(s.userv, s.data, s.address, s.checksum, s.tid.CallId, lossemul)
	if need_cleanup {
		s.cleanup()
	}
}

func (s *serverTransaction) TimersAreActive() bool {
	return s.teA != nil || s.teD != nil || s.teE != nil
}

func (s *serverTransaction) Lock() {
	s.lock.Lock()
	if s.session_lock != nil {
		s.session_lock.Lock()
		s.lock.Unlock()
	}
}

func (s *serverTransaction) Unlock() {
	if s.session_lock != nil {
		s.session_lock.Unlock()
	} else {
		s.lock.Unlock()
	}
}

func (s *serverTransaction) UpgradeToSessionLock(session_lock sync.Locker) {
	// Must be called with the s.lock already locked!
	// Must be called once only!
	s.session_lock = session_lock
	session_lock.Lock()
	s.lock.Unlock()
}

func (s *serverTransaction) SetServer(server *sippy_header.SipServer) {
	s.server = server
}

func (s *serverTransaction) SetBeforeResponseSent(cb func(resp sippy_types.SipResponse)) {
	s.before_response_sent = cb
}

func (s *serverTransaction) retrUasResponse(last_timeout time.Duration, lossemul int) {
	if last_timeout > 16*time.Second {
		s.prov_inflight_lock.Lock()
		prov_inflight := s.prov_inflight
		s.prov_inflight = nil
		s.prov_inflight_lock.Unlock()
		if sip_tm := s.sip_tm; sip_tm != nil && prov_inflight != nil {
			sip_tm.rtid_del(prov_inflight.rtid)
		}
		if s.noprack_cb != nil && s.state == RINGING {
			s.noprack_cb(nil)
		}
		return
	}
	if sip_tm := s.sip_tm; sip_tm != nil {
		if lossemul == 0 {
			sip_tm.transmitData(s.userv, s.data, s.address, "" /*checksum*/, s.tid.CallId, 0 /*lossemul*/)
		} else {
			lossemul -= 1
		}
	}
	last_timeout *= 2
	rert_t := StartTimeout(func() { s.retrUasResponse(last_timeout, lossemul) }, s, last_timeout, 1, s.logger)
	s.prov_inflight_lock.Lock()
	s.prov_inflight.t = rert_t
	s.prov_inflight_lock.Unlock()
}

func (s *serverTransaction) SetPrackCBs(prack_cb func(sippy_types.SipRequest, sippy_types.SipResponse), noprack_cb func(*sippy_time.MonoTime)) {
	s.prack_cb = prack_cb
	s.noprack_cb = noprack_cb
}

func (s *serverTransaction) Setup100rel(req sippy_types.SipRequest) {
	for _, require := range req.GetSipRequire() {
		if require.HasTag("100rel") {
			s.pr_rel = true
			return
		}
	}
	for _, supported := range req.GetSipSupported() {
		if supported.HasTag("100rel") {
			s.pr_rel = true
			return
		}
	}
}

func (s *serverTransaction) PrRel() bool {
	return s.pr_rel
}
