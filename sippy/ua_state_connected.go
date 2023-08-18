package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UaStateConnected struct {
	*uaStateGeneric
	ka_controller *keepaliveController
}

func NewUaStateConnected(ua sippy_types.UA, config sippy_conf.Config) *UaStateConnected {
	ua.SetBranch("")
	s := &UaStateConnected{
		uaStateGeneric: newUaStateGeneric(ua, config),
		ka_controller:  newKeepaliveController(ua, config.ErrorLogger()),
	}
	s.connected = true
	return s
}

func (s *UaStateConnected) OnActivation() {
	if s.ka_controller != nil {
		s.ka_controller.Start()
	}
}

func (s *UaStateConnected) String() string {
	return "Connected"
}

func (s *UaStateConnected) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	if req.GetMethod() == "REFER" {
		if req.GetReferTo() == nil {
			t.SendResponse(req.GenResponse(400, "Bad Request", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
			return nil, nil
		}
		t.SendResponse(req.GenResponse(202, "Accepted", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
		refer_to, err := req.GetReferTo().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UaStateConnected::RecvRequest: #1: " + err.Error())
			return nil, nil
		}
		s.ua.Enqueue(NewCCEventDisconnect(refer_to.GetCopy(), req.GetRtime(), s.ua.GetOrigin()))
		s.ua.RecvEvent(NewCCEventDisconnect(nil, req.GetRtime(), s.ua.GetOrigin()))
		return nil, nil
	}
	if req.GetMethod() == "INVITE" {
		s.ua.SetUasResp(req.GenResponse(100, "Trying", nil, s.ua.GetLocalUA().AsSipServer()))
		t.SendResponse(s.ua.GetUasResp(), false, nil)
		body := req.GetBody()
		rsdp := s.ua.GetRSDP()
		if body != nil && rsdp != nil && rsdp.String() == body.String() {
			s.ua.SendUasResponse(t, 200, "OK", s.ua.GetLSDP(), s.ua.GetLContacts(), false /*ack_wait*/)
			return nil, nil
		}
		event := NewCCEventUpdate(req.GetRtime(), s.ua.GetOrigin(), req.GetReason(), req.GetMaxForwards(), body)
		s.ua.OnReinvite(req, event)
		if body != nil {
			if s.ua.HasOnRemoteSdpChange() {
				s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(event, x) })
				return NewUasStateUpdating(s.ua, s.config), nil
			} else {
				s.ua.SetRSDP(body.GetCopy())
			}
		} else {
			s.ua.SetRSDP(nil)
		}
		s.ua.Enqueue(event)
		return NewUasStateUpdating(s.ua, s.config), nil
	}
	if req.GetMethod() == "BYE" {
		t.SendResponse(req.GenResponse(200, "OK", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
		//print "BYE received in the Connected state, going to the Disconnected state"
		var also *sippy_header.SipAddress
		if len(req.GetAlso()) > 0 {
			also_body, err := req.GetAlso()[0].GetBody(s.config)
			if err != nil {
				s.config.ErrorLogger().Error("UaStateConnected::RecvRequest: #3: " + err.Error())
				return nil, nil
			}
			also = also_body.GetCopy()
		}
		event := NewCCEventDisconnect(also, req.GetRtime(), s.ua.GetOrigin())
		event.SetReason(req.GetReason())
		s.ua.Enqueue(event)
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(req.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(req.GetRtime(), s.ua.GetOrigin(), 0, req) }
	}
	if req.GetMethod() == "INFO" {
		t.SendResponse(req.GenResponse(200, "OK", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
		event := NewCCEventInfo(req.GetRtime(), s.ua.GetOrigin(), req.GetBody())
		event.SetReason(req.GetReason())
		s.ua.Enqueue(event)
		return nil, nil
	}
	if req.GetMethod() == "OPTIONS" || req.GetMethod() == "UPDATE" {
		t.SendResponse(req.GenResponse(200, "OK", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
		return nil, nil
	}
	//print "wrong request %s in the state Connected" % req.GetMethod()
	return nil, nil
}

func (s *UaStateConnected) RecvEvent(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	var err error
	var req sippy_types.SipRequest

	eh := event.GetExtraHeaders()
	ok := false
	var redirect *sippy_header.SipAddress = nil

	switch ev := event.(type) {
	case *CCEventDisconnect:
		redirect = ev.GetRedirectURL()
		ok = true
	case *CCEventRedirect:
		redirect = ev.GetRedirectURL()
		ok = true
	case *CCEventFail:
		ok = true
	}
	if ok {
		//println("event", event.String(), "received in the Connected state sending BYE")
		if redirect != nil && s.ua.ShouldUseRefer() {
			var lUri *sippy_header.SipAddress

			req, err = s.ua.GenRequest("REFER", nil, nil, eh...)
			if err != nil {
				return nil, nil, err
			}
			also := sippy_header.NewSipReferTo(redirect)
			req.AppendHeader(also)
			lUri, err = s.ua.GetLUri().GetBody(s.config)
			if err != nil {
				return nil, nil, err
			}
			rby := sippy_header.NewSipReferredBy(sippy_header.NewSipAddress("", lUri.GetUrl()))
			req.AppendHeader(rby)
			s.ua.BeginNewClientTransaction(req, newRedirectController(s.ua))
		} else {
			req, err = s.ua.GenRequest("BYE", nil, nil, eh...)
			if err != nil {
				return nil, nil, err
			}
			if redirect != nil {
				also := sippy_header.NewSipAlso(redirect)
				req.AppendHeader(also)
			}
			s.ua.BeginNewClientTransaction(req, nil)
		}
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), 0, nil) }, nil
	}
	if _event, ok := event.(*CCEventUpdate); ok {
		var tr sippy_types.ClientTransaction

		body := _event.GetBody()
		if s.ua.GetLSDP() != nil && body != nil && s.ua.GetLSDP().String() == body.String() {
			if s.ua.GetRSDP() != nil {
				s.ua.Enqueue(NewCCEventConnect(200, "OK", s.ua.GetRSDP().GetCopy(), event.GetRtime(), event.GetOrigin()))
			} else {
				s.ua.Enqueue(NewCCEventConnect(200, "OK", nil, event.GetRtime(), event.GetOrigin()))
			}
			return nil, nil, nil
		}
		if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
			err := s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			if err != nil {
				ev := NewCCEventFail(400, "Malformed SDP Body", event.GetRtime(), "")
				ev.SetWarning(err.Error())
				s.ua.Enqueue(ev)
			}
			return nil, nil, nil
		}
		if body == nil {
			s.ua.SetLateMedia(true)
		}
		eh2 := eh
		if _event.GetMaxForwards() != nil {
			var max_forwards *sippy_header.SipNumericHF

			max_forwards, err = _event.GetMaxForwards().GetBody()
			if err != nil {
				return nil, nil, err
			}
			if max_forwards.Number <= 0 {
				s.ua.Enqueue(NewCCEventFail(483, "Too Many Hops", event.GetRtime(), ""))
				return nil, nil, nil
			}
			eh2 = append(eh2, sippy_header.NewSipMaxForwards(max_forwards.Number-1))
		}
		req, err = s.ua.GenRequest("INVITE", body, nil, eh2...)
		if err != nil {
			return nil, nil, err
		}
		s.ua.SetLSDP(body)
		tr, err = s.ua.PrepTr(req, nil)
		if err != nil {
			return nil, nil, err
		}
		s.ua.SetClientTransaction(tr)
		s.ua.BeginClientTransaction(req, tr)
		return NewUacStateUpdating(s.ua, s.config), nil, nil
	}
	if _event, ok := event.(*CCEventInfo); ok {
		body := _event.GetBody()
		req, err = s.ua.GenRequest("INFO", nil, nil, eh...)
		if err != nil {
			return nil, nil, err
		}
		req.SetBody(body)
		s.ua.BeginNewClientTransaction(req, nil)
		return nil, nil, nil
	}
	if _event, ok := event.(*CCEventConnect); ok && s.ua.GetPendingTr() != nil {
		s.ua.CancelExpireTimer()
		body := _event.GetBody()
		if body != nil && s.ua.HasOnLocalSdpChange() && body.NeedsUpdate() {
			s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
			return nil, nil, nil
		}
		s.ua.CancelCreditTimer() // prevent timer leak
		s.ua.StartCreditTimer(event.GetRtime())
		s.ua.SetConnectTs(event.GetRtime())
		s.ua.SetLSDP(body)
		s.ua.GetPendingTr().GetACK().SetBody(body)
		s.ua.GetPendingTr().SendACK()
		s.ua.SetPendingTr(nil)
		s.ua.ConnCb(event.GetRtime(), s.ua.GetOrigin())
	}
	//print "wrong event %s in the Connected state" % event
	return nil, nil, nil
}

func (s *UaStateConnected) OnDeactivate() {
	if s.ka_controller != nil {
		s.ka_controller.Stop()
	}
	if s.ua.GetPendingTr() != nil {
		s.ua.GetPendingTr().SendACK()
		s.ua.SetPendingTr(nil)
	}
	s.ua.CancelExpireTimer()
}

func (s *UaStateConnected) ID() sippy_types.UaStateID {
	return sippy_types.UA_STATE_CONNECTED
}
