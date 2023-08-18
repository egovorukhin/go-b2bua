package sippy

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type UacStateRinging struct {
	*uaStateGeneric
}

func NewUacStateRinging(ua sippy_types.UA, config sippy_conf.Config) sippy_types.UaState {
	return &UacStateRinging{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
}

func (s *UacStateRinging) String() string {
	return "Ringing(UAC)"
}

func (s *UacStateRinging) RecvResponse(resp sippy_types.SipResponse, tr sippy_types.ClientTransaction) (sippy_types.UaState, func()) {
	var err error
	var event sippy_types.CCEvent

	body := resp.GetBody()
	code, reason := resp.GetSCode()
	if code > 180 {
		// the 100 Trying can be processed later than 180 Ringing
		s.ua.SetLastScode(code)
	}
	if code > 100 && code < 300 {
		// the route set must be ready for sending the PRACK
		s.ua.UpdateRouting(resp, true, true)
	}
	if code < 200 {
		if rseq := resp.GetRSeq(); rseq != nil {
			if !tr.CheckRSeq(rseq) {
				// bad RSeq number - ignore the response
				return nil, nil
			}
			to_body, err := resp.GetTo().GetBody(s.config)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #6: " + err.Error())
				return nil, nil
			}
			tag := to_body.GetTag()
			rUri, err := s.ua.GetRUri().GetBody(s.config)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #7: " + err.Error())
				return nil, nil
			}
			rUri.SetTag(tag)
			cseq, err := resp.GetCSeq().GetBody()
			if err != nil {
				s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #8: " + err.Error())
				return nil, nil
			}
			req, err := s.ua.GenRequest("PRACK", nil, nil)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #9: " + err.Error())
				return nil, nil
			}
			rack := sippy_header.NewSipRAck(rseq.Number, cseq.CSeq, cseq.Method)
			req.AppendHeader(rack)
			s.ua.BeginNewClientTransaction(req, nil)
		}
		if s.ua.GetP1xxTs() == nil {
			s.ua.SetP1xxTs(resp.GetRtime())
		}
		event := NewCCEventRing(code, reason, body, resp.GetRtime(), s.ua.GetOrigin())
		s.ua.RingCb(resp.GetRtime(), s.ua.GetOrigin(), code)
		if body != nil {
			if s.ua.HasOnRemoteSdpChange() {
				s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(event, x) })
				return nil, nil
			} else {
				s.ua.SetRSDP(body.GetCopy())
			}
		} else {
			s.ua.SetRSDP(nil)
		}
		s.ua.Enqueue(event)
		return nil, nil
	}
	s.ua.CancelExpireTimer()
	if code >= 200 && code < 300 {
		var to_body *sippy_header.SipAddress
		var rUri *sippy_header.SipAddress
		var cb func()

		to_body, err = resp.GetTo().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #1: " + err.Error())
			return nil, nil
		}
		tag := to_body.GetTag()
		if tag == "" {
			var req sippy_types.SipRequest

			//print "tag-less 200 OK, disconnecting"
			event := NewCCEventFail(502, "Bad Gateway", resp.GetRtime(), s.ua.GetOrigin())
			s.ua.Enqueue(event)
			req, err = s.ua.GenRequest("BYE", nil, nil)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #2: " + err.Error())
				return nil, nil
			}
			s.ua.BeginNewClientTransaction(req, nil)
			if s.ua.GetSetupTs() != nil && !s.ua.GetSetupTs().After(resp.GetRtime()) {
				s.ua.SetDisconnectTs(resp.GetRtime())
			} else {
				now, _ := sippy_time.NewMonoTime()
				s.ua.SetDisconnectTs(now)
			}
			return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(resp.GetRtime(), s.ua.GetOrigin(), 502) }
		}
		rUri, err = s.ua.GetRUri().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #3: " + err.Error())
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
			s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #4: " + err.Error())
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
				s.config.ErrorLogger().Error("UacStateRinging::RecvResponse: #5: " + err.Error())
				return nil, nil
			}
			urls = append(urls, cbody.GetCopy())
		}
		event = NewCCEventRedirect(code, reason, body, urls, resp.GetRtime(), s.ua.GetOrigin())
	} else {
		event = NewCCEventFail(code, reason, resp.GetRtime(), s.ua.GetOrigin())
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

func (s *UacStateRinging) RecvEvent(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	switch event.(type) {
	case *CCEventFail:
	case *CCEventRedirect:
	case *CCEventDisconnect:
	default:
		//return nil, fmt.Errorf("wrong event %s in the Ringing state", event.String())
		return nil, nil, nil
	}
	s.ua.GetClientTransaction().Cancel()
	s.ua.CancelExpireTimer()
	if s.ua.GetSetupTs() != nil && !s.ua.GetSetupTs().After(event.GetRtime()) {
		s.ua.SetDisconnectTs(event.GetRtime())
	} else {
		now, _ := sippy_time.NewMonoTime()
		s.ua.SetDisconnectTs(now)
	}
	return NewUacStateCancelling(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), s.ua.GetLastScode(), nil) }, nil
}

func (s *UacStateRinging) ID() sippy_types.UaStateID {
	return sippy_types.UAC_STATE_RINGING
}
