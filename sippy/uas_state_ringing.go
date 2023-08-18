package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UasStateRinging struct {
	*uaStateGeneric
}

func NewUasStateRinging(ua sippy_types.UA, config sippy_conf.Config) *UasStateRinging {
	return &UasStateRinging{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
}

func (s *UasStateRinging) String() string {
	return "Ringing(UAS)"
}

func (s *UasStateRinging) RecvEvent(_event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	eh := _event.GetExtraHeaders()
	switch event := _event.(type) {
	case *CCEventRing:
		code, reason, body := event.scode, event.scode_reason, event.body
		if code == 0 {
			code, reason, body = 180, "Ringing", nil
		} else {
			if code == 100 {
				return nil, nil, nil
			}
			if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
				s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
				return nil, nil, nil
			}
		}
		s.ua.SetLSDP(body)
		if s.ua.GetP1xxTs() == nil {
			s.ua.SetP1xxTs(event.GetRtime())
		}
		s.ua.SendUasResponse(nil, code, reason, body, s.ua.GetLContacts(), false, eh...)
		s.ua.RingCb(event.rtime, event.origin, code)
		return nil, nil, nil
	case *CCEventConnect:
		body := event.body
		if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.SendUasResponse(nil, event.scode, event.scode_reason, body, s.ua.GetLContacts() /*ack_wait*/, true, eh...)
		s.ua.CancelExpireTimer()
		s.ua.StartCreditTimer(event.GetRtime())
		s.ua.SetConnectTs(event.GetRtime())
		return NewUasStatePreConnect(s.ua, s.config /*confirm_connect*/, false), func() { s.ua.ConnCb(event.GetRtime(), event.GetOrigin()) }, nil
	case *CCEventPreConnect:
		body := event.body
		if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.SendUasResponse(nil, event.scode, event.scode_reason, body, s.ua.GetLContacts() /*ack_wait*/, true, eh...)
		return NewUasStatePreConnect(s.ua, s.config /*confirm_connect*/, true), nil, nil
	case *CCEventRedirect:
		s.ua.SendUasResponse(nil, event.scode, event.scode_reason, event.body, event.GetContacts(), false, eh...)
		s.ua.CancelExpireTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(event.GetRtime(), event.GetOrigin(), event.scode) }, nil
	case *CCEventFail:
		code, reason := event.scode, event.scode_reason
		if code == 0 {
			code, reason = 500, "Failed"
		}
		s.ua.SendUasResponse(nil, code, reason, nil, nil, false, eh...)
		s.ua.CancelExpireTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(event.GetRtime(), event.GetOrigin(), code) }, nil
	case *CCEventDisconnect:
		code, reason := s.ua.OnEarlyUasDisconnect(event)
		eh = event.GetExtraHeaders()
		s.ua.SendUasResponse(nil, code, reason, nil, nil, false, eh...)
		s.ua.CancelExpireTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), s.ua.GetLastScode(), nil) }, nil
	}
	//return nil, fmt.Errorf("wrong event %s in the Ringing state", _event.String())
	return nil, nil, nil
}

func (s *UasStateRinging) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	if req.GetMethod() == "BYE" {
		s.ua.SendUasResponse(t, 487, "Request Terminated", nil, nil, false)
		t.SendResponseWithLossEmul(req.GenResponse(200, "OK", nil, s.ua.GetLocalUA().AsSipServer()), false, nil, s.ua.UasLossEmul())
		//print "BYE received in the Ringing state, going to the Disconnected state"
		var also *sippy_header.SipAddress = nil
		if len(req.GetAlso()) > 0 {
			also_body, err := req.GetAlso()[0].GetBody(s.config)
			if err != nil {
				s.config.ErrorLogger().Error("UasStateRinging::RecvRequest: #1: " + err.Error())
				return nil, nil
			}
			also = also_body.GetCopy()
		}
		event := NewCCEventDisconnect(also, req.GetRtime(), s.ua.GetOrigin())
		event.SetReason(req.GetReason())
		s.ua.Enqueue(event)
		s.ua.CancelExpireTimer()
		s.ua.SetDisconnectTs(req.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(req.GetRtime(), s.ua.GetOrigin(), 0, req) }
	}
	return nil, nil
}

func (s *UasStateRinging) RecvCancel(rtime *sippy_time.MonoTime, req sippy_types.SipRequest) {
	event := NewCCEventDisconnect(nil, rtime, s.ua.GetOrigin())
	if req != nil {
		event.SetReason(req.GetReason())
	}
	s.ua.SetDisconnectTs(rtime)
	s.ua.ChangeState(NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(rtime, s.ua.GetOrigin(), 0, req) })
	s.ua.EmitEvent(event)
}

func (s *UasStateRinging) ID() sippy_types.UaStateID {
	return sippy_types.UAS_STATE_RINGING
}
