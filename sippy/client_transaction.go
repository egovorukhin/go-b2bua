package sippy

import (
	"sync"
	"time"

	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
)

type clientTransaction struct {
	*baseTransaction
	teB                 *Timeout
	teC                 *Timeout
	teG                 *Timeout
	r408                sippy_types.SipResponse
	resp_receiver       sippy_types.ResponseReceiver
	expires             time.Duration
	ack                 sippy_types.SipRequest
	outbound_proxy      *sippy_net.HostPort
	cancel              sippy_types.SipRequest
	cancelPending       bool
	uack                bool
	ack_rAddr           *sippy_net.HostPort
	ack_checksum        string
	before_request_sent func(sippy_types.SipRequest)
	ack_rparams_present bool
	ack_rTarget         *sippy_header.SipURL
	ack_routes          []*sippy_header.SipRoute
	txn_headers         []sippy_header.SipHeader
	req_extra_hdrs      []sippy_header.SipHeader
	on_send_complete    func()
	seen_rseqs          map[sippy_header.RTID]bool
	last_rseq           int
}

func NewClientTransactionObj(req sippy_types.SipRequest, tid *sippy_header.TID,
	userv sippy_net.Transport, data []byte, sip_tm *sipTransactionManager,
	resp_receiver sippy_types.ResponseReceiver, session_lock sync.Locker,
	address *sippy_net.HostPort, eh []sippy_header.SipHeader,
	req_out_cb func(sippy_types.SipRequest)) (*clientTransaction, error) {
	var r408 sippy_types.SipResponse = nil
	var err error

	if resp_receiver != nil {
		r408 = req.GenResponse(408, "Request Timeout" /*body*/, nil /*server*/, nil)
	}
	expires := 32 * time.Second
	needack := false
	var ack, cancel sippy_types.SipRequest
	if req.GetMethod() == "INVITE" {
		expires = 300 * time.Second
		if req.GetExpires() != nil {
			exp, err := req.GetExpires().GetBody()
			if err == nil && exp.Number > 0 {
				expires = time.Duration(exp.Number) * time.Second
			}
		}
		needack = true
		if ack, err = req.GenACK(nil); err != nil {
			return nil, err
		}
		if cancel, err = req.GenCANCEL(); err != nil {
			return nil, err
		}
	}
	s := &clientTransaction{
		resp_receiver:       resp_receiver,
		cancelPending:       false,
		r408:                r408,
		expires:             expires,
		ack:                 ack,
		cancel:              cancel,
		uack:                false,
		before_request_sent: req_out_cb,
		ack_rparams_present: false,
		seen_rseqs:          make(map[sippy_header.RTID]bool),
		last_rseq:           0,
		req_extra_hdrs:      eh,
	}
	s.baseTransaction = newBaseTransaction(session_lock, tid, userv, sip_tm, address, data, needack)
	return s, nil
}

func (s *clientTransaction) SetOnSendComplete(fn func()) {
	s.on_send_complete = fn
}

func (s *clientTransaction) StartTimers() {
	s.startTeA()
	s.startTeB(32 * time.Second)
}

func (s *clientTransaction) cleanup() {
	s.baseTransaction.cleanup()
	s.ack = nil
	s.resp_receiver = nil
	if teB := s.teB; teB != nil {
		teB.Cancel()
		s.teB = nil
	}
	if teC := s.teC; teC != nil {
		teC.Cancel()
		s.teC = nil
	}
	if teG := s.teG; teG != nil {
		teG.Cancel()
		s.teG = nil
	}
	s.r408 = nil
	s.cancel = nil
}

func (s *clientTransaction) SetOutboundProxy(outbound_proxy *sippy_net.HostPort) {
	s.outbound_proxy = outbound_proxy
}

func (s *clientTransaction) startTeC() {
	if teC := s.teC; teC != nil {
		teC.Cancel()
	}
	s.teC = StartTimeout(s.timerC, s.lock, 32*time.Second, 1, s.logger)
}

func (s *clientTransaction) timerB() {
	if s.sip_tm == nil {
		return
	}
	//println("timerB", s.tid.String())
	s.cancelTeA()
	s.cancelTeB()
	s.state = TERMINATED
	s.startTeC()
	rtime, _ := sippy_time.NewMonoTime()
	if s.r408 != nil {
		s.r408.SetRtime(rtime)
	}
	if s.resp_receiver != nil {
		s.resp_receiver.RecvResponse(s.r408, s)
	}
}

func (s *clientTransaction) timerC() {
	if sip_tm := s.sip_tm; sip_tm != nil {
		sip_tm.tclient_del(s.tid)
		s.cleanup()
	}
}

func (s *clientTransaction) timerG() {
	if s.sip_tm == nil {
		return
	}
	s.teG = nil
	if s.state == UACK {
		s.logger.Error("INVITE transaction stuck in the UACK state, possible UAC bug")
	}
}

func (s *clientTransaction) cancelTeB() {
	if teB := s.teB; teB != nil {
		teB.Cancel()
		s.teB = nil
	}
}

func (s *clientTransaction) startTeB(timeout time.Duration) {
	if teB := s.teB; teB != nil {
		teB.Cancel()
	}
	s.teB = StartTimeout(s.timerB, s.lock, timeout, 1, s.logger)
}

func (s *clientTransaction) IncomingResponse(resp sippy_types.SipResponse, checksum string) {
	sip_tm := s.sip_tm
	if sip_tm == nil {
		return
	}
	// In those two states upper level already notified, only do ACK retransmit
	// if needed
	if s.state == TERMINATED {
		return
	}
	code := resp.GetSCodeNum()
	if code > 100 && code < 200 && resp.GetRSeq() != nil {
		rskey, err := resp.GetRTId()
		if err != nil {
			s.logger.Error("Cannot get rtid: " + err.Error())
			return
		}
		if _, ok := s.seen_rseqs[*rskey]; ok {
			sip_tm.rcache_put(checksum, &sipTMRetransmitO{
				userv:   nil,
				data:    nil,
				address: nil,
			})
			return
		}
		s.seen_rseqs[*rskey] = true
	}
	if s.state == TRYING {
		// Stop timers
		s.cancelTeA()
	}
	s.cancelTeB()
	if code < 200 {
		s.process_provisional_response(checksum, resp, sip_tm)
	} else {
		s.process_final_response(checksum, resp, sip_tm)
	}
}

func (s *clientTransaction) process_provisional_response(checksum string, resp sippy_types.SipResponse, sip_tm *sipTransactionManager) {
	// Privisional response - leave everything as is, except that
	// change state and reload timeout timer
	if s.state == TRYING {
		s.state = RINGING
		if s.cancelPending {
			sip_tm.BeginNewClientTransaction(s.cancel, nil, s.lock, nil, s.userv, s.before_request_sent)
			s.cancelPending = false
		}
	}
	s.startTeB(s.expires)
	sip_tm.rcache_set_call_id(checksum, s.tid.CallId)
	if s.resp_receiver != nil {
		s.resp_receiver.RecvResponse(resp, s)
	}
}

func (s *clientTransaction) process_final_response(checksum string, resp sippy_types.SipResponse, sip_tm *sipTransactionManager) {
	// Final response - notify upper layer and remove transaction
	if s.resp_receiver != nil {
		s.resp_receiver.RecvResponse(resp, s)
	}
	if s.needack {
		// Prepare and send ACK if necessary
		code := resp.GetSCodeNum()
		to_body, err := resp.GetTo().GetBody(sip_tm.config)
		if err != nil {
			s.logger.Debug(err.Error())
			return
		}
		tag := to_body.GetTag()
		if tag != "" {
			to_body, err = s.ack.GetTo().GetBody(sip_tm.config)
			if err != nil {
				s.logger.Debug(err.Error())
				return
			}
			to_body.SetTag(tag)
		}
		var rAddr *sippy_net.HostPort
		var rTarget *sippy_header.SipURL
		if code >= 200 && code < 300 {
			// Some hairy code ahead
			if len(resp.GetContacts()) > 0 {
				var contact *sippy_header.SipAddress
				contact, err = resp.GetContacts()[0].GetBody(sip_tm.config)
				if err != nil {
					s.logger.Debug(err.Error())
					return
				}
				rTarget = contact.GetUrl().GetCopy()
			} else {
				rTarget = nil
			}
			var routes []*sippy_header.SipRoute
			if !s.ack_rparams_present {
				routes = make([]*sippy_header.SipRoute, len(resp.GetRecordRoutes()))
				for idx, r := range resp.GetRecordRoutes() {
					r2 := r.AsSipRoute()                           // r.getCopy()
					routes[len(resp.GetRecordRoutes())-1-idx] = r2 // reverse order
				}
				if s.outbound_proxy != nil {
					routes = append([]*sippy_header.SipRoute{sippy_header.NewSipRoute(sippy_header.NewSipAddress("", sippy_header.NewSipURL("", s.outbound_proxy.Host, s.outbound_proxy.Port, true)))}, routes...)
				}
				if len(routes) > 0 {
					var r0 *sippy_header.SipAddress
					r0, err = routes[0].GetBody(sip_tm.config)
					if err != nil {
						s.logger.Debug(err.Error())
						return
					}
					if !r0.GetUrl().Lr {
						if rTarget != nil {
							routes = append(routes, sippy_header.NewSipRoute(sippy_header.NewSipAddress("", rTarget)))
						}
						rTarget = r0.GetUrl()
						routes = routes[1:]
						rAddr = rTarget.GetAddr(sip_tm.config)
					} else {
						rAddr = r0.GetUrl().GetAddr(sip_tm.config)
					}
				} else if rTarget != nil {

					rAddr = rTarget.GetAddr(sip_tm.config)
				}
				if rTarget != nil {
					s.ack.SetRURI(rTarget)
				}
			} else {
				rAddr, rTarget, routes = s.ack_rAddr, s.ack_rTarget, s.ack_routes
			}
			for _, h := range s.txn_headers {
				s.ack.AppendHeader(h)
			}
			s.ack.SetRoutes(routes)
		}
		if code >= 200 && code < 300 {
			var via0 *sippy_header.SipViaBody
			if via0, err = s.ack.GetVias()[0].GetBody(); err != nil {
				s.logger.Debug("error parsing via: " + err.Error())
				return
			}
			via0.GenBranch()
		}
		if rAddr == nil {
			rAddr = s.address
		}
		if !s.uack {
			s.BeforeRequestSent(s.ack)
			sip_tm.transmitMsg(s.userv, s.ack, rAddr, checksum, s.tid.CallId)
		} else {
			s.state = UACK
			s.ack_rAddr = rAddr
			s.ack_checksum = checksum
			sip_tm.rcache_set_call_id(checksum, s.tid.CallId)
			s.teG = StartTimeout(s.timerG, s.lock, 64*time.Second, 1, s.logger)
			return
		}
	} else {
		sip_tm.rcache_set_call_id(checksum, s.tid.CallId)
	}
	sip_tm.tclient_del(s.tid)
	s.cleanup()
}

func (s *clientTransaction) Cancel(extra_headers ...sippy_header.SipHeader) {
	sip_tm := s.sip_tm
	if sip_tm == nil {
		return
	}
	// If we got at least one provisional reply then (state == RINGING)
	// then start CANCEL transaction, otherwise deffer it
	if s.state != RINGING {
		s.cancelPending = true
	} else {
		for _, h := range extra_headers {
			s.cancel.AppendHeader(h)
		}
		for _, h := range s.txn_headers {
			s.cancel.AppendHeader(h)
		}
		sip_tm.BeginNewClientTransaction(s.cancel, nil, s.lock, nil, s.userv, s.before_request_sent)
	}
}

func (s *clientTransaction) Lock() {
	s.lock.Lock()
}

func (s *clientTransaction) Unlock() {
	s.lock.Unlock()
}

func (s *clientTransaction) SendACK() {
	if teG := s.teG; teG != nil {
		teG.Cancel()
		s.teG = nil
	}
	s.BeforeRequestSent(s.ack)
	if sip_tm := s.sip_tm; sip_tm != nil {
		sip_tm.transmitMsg(s.userv, s.ack, s.ack_rAddr, s.ack_checksum, s.tid.CallId)
		sip_tm.tclient_del(s.tid)
	}
	s.cleanup()
}

func (s *clientTransaction) GetACK() sippy_types.SipRequest {
	return s.ack
}

func (s *clientTransaction) SetUAck(uack bool) {
	s.uack = uack
}

func (s *clientTransaction) BeforeRequestSent(req sippy_types.SipRequest) {
	if s.before_request_sent != nil {
		s.before_request_sent(req)
	}
}

func (s *clientTransaction) TransmitData() {
	if sip_tm := s.sip_tm; sip_tm != nil {
		sip_tm.transmitDataWithCb(s.userv, s.data, s.address /*cachesum*/, "" /*call_id =*/, s.tid.CallId, 0, s.on_send_complete)
	}
}

func (s *clientTransaction) SetAckRparams(rAddr *sippy_net.HostPort, rTarget *sippy_header.SipURL, routes []*sippy_header.SipRoute) {
	s.ack_rparams_present = true
	s.ack_rAddr = rAddr
	s.ack_rTarget = rTarget
	s.ack_routes = routes
}

func (s *clientTransaction) CheckRSeq(rseq *sippy_header.SipRSeq) bool {
	if s.last_rseq != 0 && s.last_rseq+1 != rseq.Number {
		return false
	}
	s.last_rseq = rseq.Number
	return true
}

func (s *clientTransaction) SetTxnHeaders(hdrs []sippy_header.SipHeader) {
	s.txn_headers = hdrs
}

func (s *clientTransaction) GetReqExtraHeaders() []sippy_header.SipHeader {
	return s.req_extra_hdrs
}
