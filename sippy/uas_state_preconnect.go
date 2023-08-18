package sippy

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type UasStatePreConnect struct {
	*uaStateGeneric
	pending_ev_update *CCEventUpdate
	confirm_connect   bool
}

func NewUasStatePreConnect(ua sippy_types.UA, config sippy_conf.Config, confirm_connect bool) *UasStatePreConnect {
	ua.SetBranch("")
	s := &UasStatePreConnect{
		uaStateGeneric:  newUaStateGeneric(ua, config),
		confirm_connect: confirm_connect,
	}
	s.connected = true
	return s
}

func (s *UasStatePreConnect) String() string {
	return "PreConnect(UAS)"
}

func (s *UasStatePreConnect) try_other_events(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	var redirect *sippy_header.SipAddress = nil
	switch ev := event.(type) {
	case *CCEventDisconnect:
		redirect = ev.GetRedirectURL()
	case *CCEventRedirect:
		redirect = ev.GetRedirectURL()
	case *CCEventFail:
	default:
		//fmt.Printf("wrong event %s in the %s state", event.String(), s.ID().String())
		return nil, nil, nil
	}
	//println("event", event.String(), "received in the Connected state sending BYE")
	eh := event.GetExtraHeaders()
	if redirect != nil && s.ua.ShouldUseRefer() {
		var lUri *sippy_header.SipAddress

		req, err := s.ua.GenRequest("REFER", nil, nil, eh...)
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
		req, err := s.ua.GenRequest("BYE", nil, nil, eh...)
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

func (s *UasStatePreConnect) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	if req.GetMethod() == "INVITE" {
		t.SendResponseWithLossEmul(req.GenResponse(491, "Request Pending", nil, s.ua.GetLocalUA().AsSipServer()), false, nil, s.ua.UasLossEmul())
		return nil, nil
	} else if req.GetMethod() == "BYE" {
		s.ua.SendUasResponse(t, 200, "OK", nil, nil, false)
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
			s.config.ErrorLogger().Error("UasStatePreConnect::RecvRequest: #1: " + err.Error())
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

func (s *UasStatePreConnect) RecvEvent(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	switch ev := event.(type) {
	case *CCEventUpdate:
		s.pending_ev_update = ev
		return nil, nil, nil
	case *CCEventInfo:
		body := ev.GetBody()
		req, err := s.ua.GenRequest("INFO", nil, nil, event.GetExtraHeaders()...)
		if err != nil {
			return nil, nil, err
		}
		req.SetBody(body)
		s.ua.BeginNewClientTransaction(req, nil)
		return nil, nil, nil
	default:
		return s.try_other_events(event)
	}
}

func (s *UasStatePreConnect) OnDeactivate() {
	s.ua.CancelExpireTimer()
}

func (s *UasStatePreConnect) RecvACK(req sippy_types.SipRequest) {
	var event *CCEventConnect
	var cb func()
	rtime := req.GetRtime()
	origin := s.ua.GetOrigin()
	if s.confirm_connect {
		body := req.GetBody()
		event = NewCCEventConnect(0, "ACK", body, rtime, origin)
		s.ua.CancelExpireTimer()
		s.ua.CancelCreditTimer() // prevent timer leak
		s.ua.StartCreditTimer(rtime)
		s.ua.SetConnectTs(rtime)
		s.ua.ConnCb(rtime, origin)
		if body != nil {
			if s.ua.HasOnRemoteSdpChange() {
				ev := event
				event = nil // do not send this event via EmitEvent below
				s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(ev, x) })
			} else {
				s.ua.SetRSDP(body.GetCopy())
			}
		} else {
			s.ua.SetRSDP(nil)
		}
		cb = func() { s.ua.ConnCb(rtime, origin) }
	}
	s.ua.ChangeState(NewUaStateConnected(s.ua, s.config), cb)
	if event != nil {
		s.ua.EmitEvent(event)
	}
	if s.pending_ev_update != nil {
		s.ua.RecvEvent(s.pending_ev_update)
		s.pending_ev_update = nil
	}
}

func (s *UasStatePreConnect) ID() sippy_types.UaStateID {
	return sippy_types.UAS_STATE_PRE_CONNECT
}
