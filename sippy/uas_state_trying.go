package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UasStateTrying struct {
	*uaStateGeneric
}

func NewUasStateTrying(ua sippy_types.UA, config sippy_conf.Config) *UasStateTrying {
	return &UasStateTrying{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
}

func (s *UasStateTrying) String() string {
	return "Trying(UAS)"
}

func (s *UasStateTrying) RecvEvent(_event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
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
		s.ua.SendUasResponse(nil, code, reason, body, s.ua.GetLContacts(), false, eh...)
		if s.ua.HasNoProgressTimer() {
			s.ua.CancelNoProgressTimer()
			if s.ua.GetExMtime() != nil {
				s.ua.StartExpireTimer(event.GetRtime())
			}
		}
		if s.ua.GetP1xxTs() == nil {
			s.ua.SetP1xxTs(event.GetRtime())
		}
		if s.ua.PrRel() {
			return NewUasStateRingingRel(s.ua, s.config), func() { s.ua.RingCb(event.GetRtime(), event.GetOrigin(), code) }, nil
		}
		return NewUasStateRinging(s.ua, s.config), func() { s.ua.RingCb(event.GetRtime(), event.GetOrigin(), code) }, nil
	case *CCEventPreConnect:
		code, reason, body := event.scode, event.scode_reason, event.body
		if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.CancelNoProgressTimer()
		s.ua.SendUasResponse(nil, code, reason, body, s.ua.GetLContacts() /*ack_wait*/, true, eh...)
		return NewUasStatePreConnect(s.ua, s.config, true /*confirm_connect*/), nil, nil
	case *CCEventConnect:
		code, reason, body := event.scode, event.scode_reason, event.body
		if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.SendUasResponse(nil, code, reason, body, s.ua.GetLContacts() /*ack_wait*/, true, eh...)
		s.ua.CancelExpireTimer()
		s.ua.CancelNoProgressTimer()
		s.ua.StartCreditTimer(event.GetRtime())
		s.ua.SetConnectTs(event.GetRtime())
		return NewUasStatePreConnect(s.ua, s.config, false /*confirm_connect*/), func() { s.ua.ConnCb(event.GetRtime(), event.GetOrigin()) }, nil
	case *CCEventRedirect:
		s.ua.SendUasResponse(nil, event.scode, event.scode_reason, event.body, event.GetContacts(), false, eh...)
		s.ua.CancelExpireTimer()
		s.ua.CancelNoProgressTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(event.GetRtime(), event.GetOrigin(), event.scode) }, nil
	case *CCEventFail:
		code, reason := event.scode, event.scode_reason
		if code == 0 {
			code, reason = 500, "Failed"
		}
		s.ua.SendUasResponse(nil, code, reason, nil, nil, false, eh...)
		s.ua.CancelExpireTimer()
		s.ua.CancelNoProgressTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateFailed(s.ua, s.config), func() { s.ua.FailCb(event.GetRtime(), event.GetOrigin(), code) }, nil
	case *CCEventDisconnect:
		code, reason := s.ua.OnEarlyUasDisconnect(event)
		eh = event.GetExtraHeaders()
		s.ua.SendUasResponse(nil, code, reason, nil, nil, false, eh...)
		s.ua.CancelExpireTimer()
		s.ua.CancelNoProgressTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), s.ua.GetLastScode(), nil) }, nil
	}
	//return nil, fmt.Errorf("uas-trying: wrong event %s in the Trying state", _event.String())
	return nil, nil, nil
}

func (s *UasStateTrying) RecvCancel(rtime *sippy_time.MonoTime, req sippy_types.SipRequest) {
	event := NewCCEventDisconnect(nil, rtime, s.ua.GetOrigin())
	if req != nil {
		event.SetReason(req.GetReason())
	}
	s.ua.SetDisconnectTs(rtime)
	s.ua.ChangeState(NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(rtime, s.ua.GetOrigin(), 0, req) })
	s.ua.EmitEvent(event)
}

func (s *UasStateTrying) ID() sippy_types.UaStateID {
	return sippy_types.UAS_STATE_TRYING
}
