package sippy

import (
	"time"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UacStateIdle struct {
	*uaStateGeneric
	config sippy_conf.Config
}

func NewUacStateIdle(ua sippy_types.UA, config sippy_conf.Config) *UacStateIdle {
	return &UacStateIdle{
		uaStateGeneric: newUaStateGeneric(ua, config),
		config:         config,
	}
}

func (s *UacStateIdle) String() string {
	return "Idle(UAC)"
}

func (s *UacStateIdle) RecvEvent(_event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	var err error
	var rUri *sippy_header.SipAddress
	var lUri *sippy_header.SipAddress
	var contact *sippy_header.SipAddress
	var req sippy_types.SipRequest
	var tr sippy_types.ClientTransaction

	switch event := _event.(type) {
	case *CCEventTry:
		if s.ua.GetSetupTs() == nil {
			s.ua.SetSetupTs(event.rtime)
		}
		s.ua.SetOrigin("callee")
		body := event.GetBody()
		if body != nil {
			if body.NeedsUpdate() && s.ua.HasOnLocalSdpChange() {
				s.ua.OnLocalSdpChange(body, func(sippy_types.MsgBody) { s.ua.RecvEvent(event) })
				return nil, nil, nil
			}
		} else {
			s.ua.SetLateMedia(true)
		}
		if event.GetSipCallId() == nil {
			s.ua.SetCallId(sippy_header.GenerateSipCallId(s.config))
		} else {
			s.ua.SetCallId(event.GetSipCallId().GetCopy())
		}
		s.ua.SetRTarget(sippy_header.NewSipURL(event.GetCLD(), s.ua.GetRAddr0().Host, s.ua.GetRAddr0().Port, false))
		s.ua.SetRUri(sippy_header.NewSipTo(sippy_header.NewSipAddress("", s.ua.GetRTarget().GetCopy()), s.config))
		rUri, err = s.ua.GetRUri().GetBody(s.config)
		if err != nil {
			return nil, nil, err
		}
		rUri.GetUrl().Port = nil
		s.ua.SetLUri(sippy_header.NewSipFrom(sippy_header.NewSipAddress(event.GetCallerName(), sippy_header.NewSipURL(event.GetCLI(), s.config.GetMyAddress(), s.config.GetMyPort(), false)), s.config))
		s.ua.RegConsumer(s.ua, s.ua.GetCallId().CallId)
		lUri, err = s.ua.GetLUri().GetBody(s.config)
		if err != nil {
			return nil, nil, err
		}
		lUri.GetUrl().Port = nil
		lUri.SetTag(s.ua.GetLTag())
		s.ua.SetLCSeq(200)
		if s.ua.GetLContact() == nil {
			if src_addr := s.ua.GetSourceAddress(); src_addr != nil {
				s.ua.SetLContact(sippy_header.NewSipContactFromHostPort(src_addr.Host, src_addr.Port))
			} else {
				s.ua.SetLContact(sippy_header.NewSipContact(s.config))
			}
		}
		contact, err = s.ua.GetLContact().GetBody(s.config)
		if err != nil {
			return nil, nil, err
		}
		contact.GetUrl().Username = event.GetCLI()
		s.ua.SetRoutes(event.routes)
		s.ua.SetLSDP(body)
		eh := event.GetExtraHeaders()
		if event.GetMaxForwards() != nil {
			eh = append(eh, event.GetMaxForwards())
		}
		s.ua.OnUacSetupComplete()
		req, err = s.ua.GenRequest("INVITE", body /*Challenge*/, nil, eh...)
		if err != nil {
			return nil, nil, err
		}
		tr, err = s.ua.PrepTr(req, eh)
		if err != nil {
			return nil, nil, err
		}
		s.ua.SetClientTransaction(tr)
		s.ua.BeginClientTransaction(req, tr)
		if s.ua.PassAuth() && event.GetSipAuthorizationHF() != nil {
			req.AppendHeader(event.GetSipAuthorizationHF())
		}
		s.ua.SetAuth(nil)

		if s.ua.GetExpireTime() > 0 {
			s.ua.SetExMtime(event.GetRtime().Add(s.ua.GetExpireTime()))
		}
		if s.ua.GetNoProgressTime() > 0 && (s.ua.GetExpireTime() <= 0 || s.ua.GetNoProgressTime() < s.ua.GetExpireTime()) {
			s.ua.SetNpMtime(event.GetRtime().Add(s.ua.GetNoProgressTime()))
		}
		if (s.ua.GetNoReplyTime() > 0 && s.ua.GetNoReplyTime() < time.Duration(32*time.Second)) &&
			(s.ua.GetExpireTime() <= 0 || s.ua.GetNoReplyTime() < s.ua.GetExpireTime()) &&
			(s.ua.GetNoProgressTime() <= 0 || s.ua.GetNoReplyTime() < s.ua.GetNoProgressTime()) {
			s.ua.SetNrMtime(event.GetRtime().Add(s.ua.GetNoReplyTime()))
		}
		if s.ua.GetNrMtime() != nil {
			s.ua.StartNoReplyTimer()
		} else if s.ua.GetNpMtime() != nil {
			s.ua.StartNoProgressTimer()
		} else if s.ua.GetExMtime() != nil {
			s.ua.StartExpireTimer(event.GetRtime())
		}
		return NewUacStateTrying(s.ua, s.config), nil, nil
	case *CCEventFail:
	case *CCEventRedirect:
	case *CCEventDisconnect:
	default:
		return nil, nil, nil
	}
	if s.ua.GetSetupTs() != nil && !_event.GetRtime().Before(s.ua.GetSetupTs()) {
		s.ua.SetDisconnectTs(_event.GetRtime())
	} else {
		disconnect_ts, _ := sippy_time.NewMonoTime()
		s.ua.SetDisconnectTs(disconnect_ts)
	}
	return NewUaStateDead(s.ua, s.config), func() { s.ua.DiscCb(_event.GetRtime(), _event.GetOrigin(), 0, nil) }, nil
}

func (s *UacStateIdle) ID() sippy_types.UaStateID {
	return sippy_types.UAC_STATE_IDLE
}
