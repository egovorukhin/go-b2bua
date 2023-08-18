package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UasStateIdle struct {
	*uaStateGeneric
	config sippy_conf.Config
}

func NewUasStateIdle(ua sippy_types.UA, config sippy_conf.Config) *UasStateIdle {
	return &UasStateIdle{
		uaStateGeneric: newUaStateGeneric(ua, config),
		config:         config,
	}
}

func (s *UasStateIdle) String() string {
	return "Idle(UAS)"
}

func (s *UasStateIdle) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	var err error
	var contact *sippy_header.SipAddress
	var to_body *sippy_header.SipAddress
	var from_body *sippy_header.SipAddress
	var via0 *sippy_header.SipViaBody

	if req.GetMethod() != "INVITE" {
		//print "wrong request %s in the Trying state" % req.getMethod()
		return nil, nil
	}
	s.ua.SetOrigin("caller")
	//print "INVITE received in the Idle state, going to the Trying state"
	s.ua.SetUasResp(req.GenResponse(100, "Trying", nil, s.ua.GetLocalUA().AsSipServer()))
	s.ua.SetLCSeq(100) // XXX: 100 for debugging so that incorrect CSeq generation will be easily spotted
	if s.ua.GetLContact() == nil {
		if src_addr := s.ua.GetSourceAddress(); src_addr != nil {
			s.ua.SetLContact(sippy_header.NewSipContactFromHostPort(src_addr.Host, src_addr.Port))
		} else {
			s.ua.SetLContact(sippy_header.NewSipContact(s.config))
		}
	}
	contact, err = req.GetContacts()[0].GetBody(s.config)
	if err != nil {
		s.config.ErrorLogger().Error("UasStateIdle::RecvRequest: #1: " + err.Error())
		return nil, nil
	}
	s.ua.SetRTarget(contact.GetUrl().GetCopy())
	s.ua.UpdateRouting(s.ua.GetUasResp() /*update_rtarget*/, false /*reverse_routes*/, false)
	s.ua.SetRAddr0(s.ua.GetRAddr())
	t.SendResponseWithLossEmul(s.ua.GetUasResp(), false, nil, s.ua.GetUasLossEmul())
	to_body, err = s.ua.GetUasResp().GetTo().GetBody(s.config)
	if err != nil {
		s.config.ErrorLogger().Error("UasStateIdle::RecvRequest: #2: " + err.Error())
		return nil, nil
	}
	to_body.SetTag(s.ua.GetLTag())
	from_body, err = s.ua.GetUasResp().GetFrom().GetBody(s.config)
	if err != nil {
		s.config.ErrorLogger().Error("UasStateIdle::RecvRequest: #3: " + err.Error())
		return nil, nil
	}
	s.ua.SetLUri(sippy_header.NewSipFrom(to_body, s.config))
	s.ua.SetRUri(sippy_header.NewSipTo(from_body, s.config))
	s.ua.SetCallId(s.ua.GetUasResp().GetCallId())
	s.ua.RegConsumer(s.ua, s.ua.GetCallId().CallId)
	auth_hf := req.GetSipAuthorizationHF()
	body := req.GetBody()
	via0, err = req.GetVias()[0].GetBody()
	if err != nil {
		s.config.ErrorLogger().Error("UasStateIdle::RecvRequest: #4: " + err.Error())
		return nil, nil
	}
	s.ua.SetBranch(via0.GetBranch())
	event, err := NewCCEventTry(s.ua.GetCallId(), from_body.GetUrl().Username,
		req.GetRURI().Username, body, auth_hf, from_body.GetName(), req.GetRtime(), s.ua.GetOrigin())
	if err != nil {
		s.config.ErrorLogger().Error("UasStateIdle::RecvRequest: #5: " + err.Error())
		return nil, nil
	}
	event.SetReason(req.GetReason())
	event.SetMaxForwards(req.GetMaxForwards())
	if s.ua.GetExpireTime() > 0 {
		s.ua.SetExMtime(event.GetRtime().Add(s.ua.GetExpireTime()))
	}
	if s.ua.GetNoProgressTime() > 0 && (s.ua.GetExpireTime() <= 0 || s.ua.GetNoProgressTime() < s.ua.GetExpireTime()) {
		s.ua.SetNpMtime(event.GetRtime().Add(s.ua.GetNoProgressTime()))
	}
	if s.ua.GetNpMtime() != nil {
		s.ua.StartNoProgressTimer()
	} else if s.ua.GetExMtime() != nil {
		s.ua.StartExpireTimer(req.GetRtime())
	}
	if body != nil {
		if s.ua.HasOnRemoteSdpChange() {
			s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(event, x) })
			s.ua.SetSetupTs(req.GetRtime())
			return NewUasStateTrying(s.ua, s.config), nil
		} else {
			s.ua.SetRSDP(body.GetCopy())
		}
	} else {
		s.ua.SetRSDP(nil)
	}
	s.ua.Enqueue(event)
	s.ua.SetSetupTs(req.GetRtime())
	return NewUasStateTrying(s.ua, s.config), nil
}

func (s *UasStateIdle) ID() sippy_types.UaStateID {
	return sippy_types.UAS_STATE_IDLE
}
