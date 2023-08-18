package sippy

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type UasStateUpdating struct {
	*uaStateGeneric
}

func NewUasStateUpdating(ua sippy_types.UA, config sippy_conf.Config) *UasStateUpdating {
	s := &UasStateUpdating{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
	s.connected = true
	return s
}

func (s *UasStateUpdating) String() string {
	return "Updating(UAS)"
}

func (s *UasStateUpdating) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	if req.GetMethod() == "INVITE" {
		t.SendResponseWithLossEmul(req.GenResponse(491, "Request Pending", nil, s.ua.GetLocalUA().AsSipServer()), false, nil, s.ua.UasLossEmul())
		return nil, nil
	} else if req.GetMethod() == "BYE" {
		s.ua.SendUasResponse(t, 487, "Request Terminated", nil, nil, false)
		t.SendResponseWithLossEmul(req.GenResponse(200, "OK", nil, s.ua.GetLocalUA().AsSipServer()), false, nil, s.ua.UasLossEmul())
		//print "BYE received in the Updating state, going to the Disconnected state"
		event := NewCCEventDisconnect(nil, req.GetRtime(), s.ua.GetOrigin())
		event.SetReason(req.GetReason())
		s.ua.Enqueue(event)
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(req.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(req.GetRtime(), s.ua.GetOrigin(), 0, req) }
	} else if req.GetMethod() == "REFER" {
		if req.GetReferTo() == nil {
			t.SendResponseWithLossEmul(req.GenResponse(400, "Bad Request", nil, s.ua.GetLocalUA().AsSipServer()), false, nil, s.ua.UasLossEmul())
			return nil, nil
		}
		s.ua.SendUasResponse(t, 487, "Request Terminated", nil, nil, false)
		t.SendResponseWithLossEmul(req.GenResponse(202, "Accepted", nil, s.ua.GetLocalUA().AsSipServer()), false, nil, s.ua.UasLossEmul())
		refer_to, err := req.GetReferTo().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UasStateUpdating::RecvRequest: #1: " + err.Error())
			return nil, nil
		}
		s.ua.Enqueue(NewCCEventDisconnect(refer_to.GetCopy(), req.GetRtime(), s.ua.GetOrigin()))
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(req.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(req.GetRtime(), s.ua.GetOrigin(), 0, req) }
	}
	//print "wrong request %s in the state Updating" % req.getMethod()
	return nil, nil
}

func (s *UasStateUpdating) RecvEvent(_event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	eh := _event.GetExtraHeaders()
	switch event := _event.(type) {
	case *CCEventRing:
		code, reason, body := event.scode, event.scode_reason, event.body
		if code == 0 {
			code, reason, body = 180, "Ringing", nil
		}
		if body != nil && body.NeedsUpdate() && s.ua.HasOnLocalSdpChange() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.SendUasResponse(nil, code, reason, body, nil, false, eh...)
		return nil, nil, nil
	case *CCEventPreConnect:
		code, reason, body := event.scode, event.scode_reason, event.body
		if body != nil && body.NeedsUpdate() && s.ua.HasOnLocalSdpChange() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.SendUasResponse(nil, code, reason, body, s.ua.GetLContacts(), true /*ack_wait*/, eh...)
		return NewUasStatePreConnect(s.ua, s.config, true /*confirm_connect*/), nil, nil
	case *CCEventConnect:
		code, reason, body := event.scode, event.scode_reason, event.body
		if body != nil && body.NeedsUpdate() && s.ua.HasOnLocalSdpChange() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.SetLSDP(body)
		s.ua.SendUasResponse(nil, code, reason, body, s.ua.GetLContacts(), true, eh...)
		return NewUasStatePreConnect(s.ua, s.config, false /*confirm_connect*/), nil, nil
	case *CCEventRedirect:
		s.ua.SendUasResponse(nil, event.scode, event.scode_reason, event.body, event.GetContacts(), true /*ack_wait*/, eh...)
		return NewUasStatePreConnect(s.ua, s.config, false /*confirm_connect*/), nil, nil
	case *CCEventFail:
		code, reason := event.scode, event.scode_reason
		if code == 0 {
			code, reason = 500, "Failed"
		}
		if event.warning != nil {
			eh = append(eh, event.warning)
		}
		s.ua.SetRSDP(nil)
		s.ua.SendUasResponse(nil, code, reason, nil, nil, true /*ack_wait*/, eh...)
		return NewUasStatePreConnect(s.ua, s.config, false /*confirm_connect*/), nil, nil
	case *CCEventDisconnect:
		s.ua.SendUasResponse(nil, 487, "Request Terminated", nil, nil, false, eh...)
		req, err := s.ua.GenRequest("BYE", nil, nil, eh...)
		if err != nil {
			return nil, nil, err
		}
		s.ua.BeginNewClientTransaction(req, nil)
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), 0, nil) }, nil
	}
	//return nil, fmt.Errorf("wrong event %s in the Updating state", _event.String())
	return nil, nil, nil
}

func (s *UasStateUpdating) RecvCancel(rtime *sippy_time.MonoTime, inreq sippy_types.SipRequest) {
	req, err := s.ua.GenRequest("BYE", nil, nil)
	if err != nil {
		s.config.ErrorLogger().Error("UasStateUpdating::Cancel: #1: " + err.Error())
		return
	}
	s.ua.BeginNewClientTransaction(req, nil)
	s.ua.CancelCreditTimer()
	s.ua.SetDisconnectTs(rtime)
	s.ua.ChangeState(NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(rtime, s.ua.GetOrigin(), 0, inreq) })
	event := NewCCEventDisconnect(nil, rtime, s.ua.GetOrigin())
	if inreq != nil {
		event.SetReason(inreq.GetReason())
	}
	s.ua.EmitEvent(event)
}

func (s *UasStateUpdating) ID() sippy_types.UaStateID {
	return sippy_types.UAS_STATE_UPDATING
}
