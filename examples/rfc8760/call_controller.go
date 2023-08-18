package main

import (
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type callController struct {
	uaA         sippy_types.UA
	uaO         sippy_types.UA
	lock        *sync.Mutex // this must be a reference to prevent memory leak
	id          int64
	cmap        *callMap
	identity_hf sippy_header.SipHeader
	date_hf     *sippy_header.SipDate
	call_id     string
}

var Next_cc_id chan int64

func NewCallController(cmap *callMap, identity_hf sippy_header.SipHeader, date_hf *sippy_header.SipDate) *callController {
	s := &callController{
		id:          <-Next_cc_id,
		uaO:         nil,
		lock:        new(sync.Mutex),
		cmap:        cmap,
		identity_hf: identity_hf,
		date_hf:     date_hf,
	}
	s.uaA = sippy.NewUA(cmap.Sip_tm, cmap.config, cmap.config.Nh_addr, s, s.lock, nil)
	s.uaA.SetDeadCb(s.aDead)
	//s.uaA.SetCreditTime(5 * time.Second)
	return s
}

func (s *callController) Error(msg string) {
	s.cmap.logger.Error(s.call_id + ": " + msg)
}

func (s *callController) RecvEvent(event sippy_types.CCEvent, ua sippy_types.UA) {
	if ua == s.uaA {
		if s.uaO == nil {
			ev_try, ok := event.(*sippy.CCEventTry)
			if !ok {
				s.uaA.RecvEvent(sippy.NewCCEventDisconnect(nil, event.GetRtime(), ""))
				return
			}
			s.uaO = sippy.NewUA(s.cmap.Sip_tm, s.cmap.config, s.cmap.config.Nh_addr, s, s.lock, nil)
			s.uaO.SetDeadCb(s.oDead)
			s.uaO.SetRAddr(s.cmap.config.Nh_addr)
			if s.cmap.config.Authname_out != "" {
				s.uaO.SetUsername(s.cmap.config.Authname_out)
				s.uaO.SetPassword(s.cmap.config.Passwd_out)
			}
			if s.cmap.config.Authname_in != "" {
				entity_body := ""
				if body := ev_try.GetBody(); body != nil {
					entity_body = body.String()
				}
				sip_auth := ev_try.GetSipAuthorizationBody()
				if sip_auth == nil {
					www_auth := sippy_header.NewSipWWWAuthenticateWithRealm("myrealm", s.cmap.config.Hash_alg, time.Now())
					s.uaA.RecvEvent(sippy.NewCCEventFail(401, "Unauthorized", nil, "", www_auth))
					return
				} else if sip_auth.GetUsername() == "" || !sip_auth.Verify(s.cmap.config.Passwd_in, "INVITE", entity_body) {
					s.uaA.RecvEvent(sippy.NewCCEventFail(401, "Unauthorized", nil, ""))
					return
				}
			}
		}
		s.uaO.RecvEvent(event)
	} else {
		s.uaA.RecvEvent(event)
	}
}

func (s *callController) aDead() {
	s.cmap.Remove(s.id)
}

func (s *callController) oDead() {
	s.cmap.Remove(s.id)
}

func (s *callController) Shutdown() {
	s.uaA.Disconnect(nil, "")
}

func (s *callController) String() string {
	res := "uaA:" + s.uaA.String() + ", uaO: "
	if s.uaO == nil {
		res += "nil"
	} else {
		res += s.uaO.String()
	}
	return res
}
