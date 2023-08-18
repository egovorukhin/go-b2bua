package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UacStateTrying struct {
	*uaStateGeneric
}

func NewUacStateTrying(ua sippy_types.UA, config sippy_conf.Config) *UacStateTrying {
	return &UacStateTrying{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
}

func (s *UacStateTrying) String() string {
	return "Trying(UAC)"
}

func (s *UacStateTrying) RecvResponse(resp sippy_types.SipResponse, tr sippy_types.ClientTransaction) (sippy_types.UaState, func()) {
	var err error
	var event sippy_types.CCEvent

	body := resp.GetBody()
	code, reason := resp.GetSCode()
	s.ua.SetLastScode(code)

	if s.ua.HasNoReplyTimer() {
		s.ua.CancelNoReplyTimer()
		if code == 100 && s.ua.GetNpMtime() != nil {
			s.ua.StartNoProgressTimer()
		} else if code < 200 && s.ua.GetExMtime() != nil {
			s.ua.StartExpireTimer(resp.GetRtime())
		}
	}
	if code == 100 {
		s.ua.SetP100Ts(resp.GetRtime())
		s.ua.Enqueue(NewCCEventRing(code, reason, body, resp.GetRtime(), s.ua.GetOrigin()))
		return nil, nil
	}
	if s.ua.HasNoProgressTimer() {
		s.ua.CancelNoProgressTimer()
		if code < 200 && s.ua.GetExMtime() != nil {
			s.ua.StartExpireTimer(resp.GetRtime())
		}
	}
	if rseq := resp.GetRSeq(); rseq != nil {
		if !tr.CheckRSeq(rseq) {
			// bad RSeq number - ignore the response
			return nil, nil
		}
		to_body, err := resp.GetTo().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #7: " + err.Error())
			return nil, nil
		}
		tag := to_body.GetTag()
		rUri, err := s.ua.GetRUri().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #8: " + err.Error())
			return nil, nil
		}
		rUri.SetTag(tag)
		cseq, err := resp.GetCSeq().GetBody()
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #9: " + err.Error())
			return nil, nil
		}
		req, err := s.ua.GenRequest("PRACK", nil, nil)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #10: " + err.Error())
			return nil, nil
		}
		rack := sippy_header.NewSipRAck(rseq.Number, cseq.CSeq, cseq.Method)
		req.AppendHeader(rack)
		s.ua.BeginNewClientTransaction(req, nil)
	}
	if code > 100 && code < 300 {
		// the route set must be ready for sending the PRACK
		s.ua.UpdateRouting(resp, true, true)
	}
	if code < 200 {
		event := NewCCEventRing(code, reason, body, resp.GetRtime(), s.ua.GetOrigin())
		if body != nil {
			if s.ua.HasOnRemoteSdpChange() {
				s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(event, x) })
				s.ua.SetP1xxTs(resp.GetRtime())
				return NewUacStateRinging(s.ua, s.config), func() { s.ua.RingCb(resp.GetRtime(), s.ua.GetOrigin(), code) }
			} else {
				s.ua.SetRSDP(body.GetCopy())
			}
		} else {
			s.ua.SetRSDP(nil)
		}
		s.ua.Enqueue(event)
		s.ua.SetP1xxTs(resp.GetRtime())
		return NewUacStateRinging(s.ua, s.config), func() { s.ua.RingCb(resp.GetRtime(), s.ua.GetOrigin(), code) }
	}
	s.ua.CancelExpireTimer()
	if code >= 200 && code < 300 {
		var to_body *sippy_header.SipAddress
		var rUri *sippy_header.SipAddress
		var cb func()

		to_body, err = resp.GetTo().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #1: " + err.Error())
			return nil, nil
		}
		tag := to_body.GetTag()
		if tag == "" {
			var req sippy_types.SipRequest

			//logger.Debug("tag-less 200 OK, disconnecting")
			s.ua.Enqueue(NewCCEventFail(502, "Bad Gateway", resp.GetRtime(), s.ua.GetOrigin()))
			// Generate and send BYE
			req, err = s.ua.GenRequest("BYE", nil, nil)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #2: " + err.Error())
				return nil, nil
			}
			s.ua.BeginNewClientTransaction(req, nil)
			if s.ua.GetSetupTs() != nil && !s.ua.GetSetupTs().After(resp.GetRtime()) {
				s.ua.SetDisconnectTs(resp.GetRtime())
			} else {
				now, _ := sippy_time.NewMonoTime()
				s.ua.SetDisconnectTs(now)
			}
			return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(resp.GetRtime(), s.ua.GetOrigin(), code) }
		}
		rUri, err = s.ua.GetRUri().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #3: " + err.Error())
			return nil, nil
		}
		rUri.SetTag(tag)
		if !s.ua.GetLateMedia() || body == nil {
			s.ua.SetLateMedia(false)
			event = NewCCEventConnect(code, reason, body, resp.GetRtime(), s.ua.GetOrigin())
			s.ua.StartCreditTimer(resp.GetRtime())
			s.ua.SetConnectTs(resp.GetRtime())
			cb = func() { s.ua.ConnCb(resp.GetRtime(), s.ua.GetOrigin()) }
		} else {
			event = NewCCEventPreConnect(code, reason, body, resp.GetRtime(), s.ua.GetOrigin())
			tr.SetUAck(true)
			s.ua.SetPendingTr(tr)
		}
		newstate := NewUaStateConnected(s.ua, s.config)
		if body != nil {
			if s.ua.HasOnRemoteSdpChange() {
				s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(event, x) })
				s.ua.SetConnectTs(resp.GetRtime())
				return newstate, cb
			} else {
				s.ua.SetRSDP(body.GetCopy())
			}
		} else {
			s.ua.SetRSDP(nil)
		}
		s.ua.Enqueue(event)
		return newstate, cb
	}
	if (code == 301 || code == 302) && len(resp.GetContacts()) > 0 {
		var contact *sippy_header.SipAddress

		contact, err = resp.GetContacts()[0].GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #4: " + err.Error())
			return nil, nil
		}
		event = NewCCEventRedirect(code, reason, body,
			[]*sippy_header.SipAddress{contact.GetCopy()},
			resp.GetRtime(), s.ua.GetOrigin())
	} else if code == 300 && len(resp.GetContacts()) > 0 {
		urls := make([]*sippy_header.SipAddress, 0)
		for _, contact := range resp.GetContacts() {
			var cbody *sippy_header.SipAddress

			cbody, err = contact.GetBody(s.config)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateTrying::RecvResponse: #5: " + err.Error())
				return nil, nil
			}
			urls = append(urls, cbody.GetCopy())
		}
		event = NewCCEventRedirect(code, reason, body, urls, resp.GetRtime(), s.ua.GetOrigin())
	} else {
		event_fail := NewCCEventFail(code, reason, resp.GetRtime(), s.ua.GetOrigin())
		event = event_fail
		if s.ua.GetPassAuth() {
			if code == 401 && len(resp.GetSipWWWAuthenticates()) > 0 {
				event_fail.challenges = make([]sippy_header.SipHeader, len(resp.GetSipWWWAuthenticates()))
				for i, hdr := range resp.GetSipWWWAuthenticates() {
					event_fail.challenges[i] = hdr.GetCopy()
				}
			} else if code == 407 && len(resp.GetSipProxyAuthenticates()) > 0 {
				event_fail.challenges = make([]sippy_header.SipHeader, len(resp.GetSipProxyAuthenticates()))
				for i, hdr := range resp.GetSipProxyAuthenticates() {
					event_fail.challenges[i] = hdr.GetCopy()
				}
			}
		}
		if resp.GetReason() != nil {
			event_fail.sip_reason = resp.GetReason().GetCopy()
		}
		event.SetReason(resp.GetReason())
	}
	s.ua.Enqueue(event)
	if s.ua.GetSetupTs() != nil && !s.ua.GetSetupTs().After(resp.GetRtime()) {
		s.ua.SetDisconnectTs(resp.GetRtime())
	} else {
		now, _ := sippy_time.NewMonoTime()
		s.ua.SetDisconnectTs(now)
	}
	return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(resp.GetRtime(), s.ua.GetOrigin(), code) }
}

func (s *UacStateTrying) RecvEvent(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	cancel_transaction := false
	switch event.(type) {
	case *CCEventFail:
		cancel_transaction = true
	case *CCEventRedirect:
		cancel_transaction = true
	case *CCEventDisconnect:
		cancel_transaction = true
	}
	if cancel_transaction {
		s.ua.GetClientTransaction().Cancel(event.GetExtraHeaders()...)
		s.ua.CancelExpireTimer()
		s.ua.CancelNoProgressTimer()
		s.ua.CancelNoReplyTimer()
		if s.ua.GetSetupTs() != nil && !s.ua.GetSetupTs().After(event.GetRtime()) {
			s.ua.SetDisconnectTs(event.GetRtime())
		} else {
			now, _ := sippy_time.NewMonoTime()
			s.ua.SetDisconnectTs(now)
		}
		return NewUacStateCancelling(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), s.ua.GetLastScode(), nil) }, nil
	}
	//return nil, fmt.Errorf("uac-trying: wrong event %s in the Trying state", event.String())
	return nil, nil, nil
}

func (s *UacStateTrying) ID() sippy_types.UaStateID {
	return sippy_types.UAC_STATE_TRYING
}
