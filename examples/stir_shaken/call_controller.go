package main

import (
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

var Next_cc_id chan int64

type CallController struct {
	uaA         sippy_types.UA
	uaO         sippy_types.UA
	lock        *sync.Mutex // this must be a reference to prevent memory leak
	id          int64
	cmap        *callMap
	identity_hf sippy_header.SipHeader
	date_hf     *sippy_header.SipDate
	call_id     string
}

func NewCallController(cmap *callMap, identity_hf sippy_header.SipHeader, date_hf *sippy_header.SipDate) *CallController {
	s := &CallController{
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

func (s *CallController) Error(msg string) {
	s.cmap.logger.Error(s.call_id + ": " + msg)
}

func (s *CallController) RecvEvent(event sippy_types.CCEvent, ua sippy_types.UA) {
	if ua == s.uaA {
		if ev_try, ok := event.(*sippy.CCEventTry); ok {
			s.call_id = ev_try.GetSipCallId().StringBody()
			if s.cmap.config.Verify && !s.SshakenVerify(ev_try) {
				s.uaA.RecvEvent(sippy.NewCCEventFail(438, "Invalid Identity Header", event.GetRtime(), ""))
				return
			}
		}
		if s.uaO == nil {
			ev_try, ok := event.(*sippy.CCEventTry)
			if !ok {
				s.uaA.RecvEvent(sippy.NewCCEventDisconnect(nil, event.GetRtime(), ""))
				return
			}
			s.uaO = sippy.NewUA(s.cmap.Sip_tm, s.cmap.config, s.cmap.config.Nh_addr, s, s.lock, nil)
			identity, date, err := s.SshakenAuth(ev_try.GetCLI(), ev_try.GetCLD())
			if err == nil {
				extra_headers := []sippy_header.SipHeader{
					sippy_header.NewSipDate(date),
					sippy_header.NewSipGenericHF("Identity", identity),
				}
				s.uaO.SetExtraHeaders(extra_headers)
			}
			s.uaO.SetDeadCb(s.oDead)
			s.uaO.SetRAddr(s.cmap.config.Nh_addr)
		}
		s.uaO.RecvEvent(event)
	} else {
		s.uaA.RecvEvent(event)
	}
}

func (s *CallController) SshakenVerify(ev_try *sippy.CCEventTry) bool {
	if s.identity_hf == nil || s.date_hf == nil {
		s.Error("Verification failure: no identity provided")
		return false
	}
	identity := s.identity_hf.StringBody()
	date_ts, err := s.date_hf.GetTime()
	if err != nil {
		s.Error("Error parsing Date: header: " + err.Error())
		return false
	}
	orig_tn := ev_try.GetCLI()
	dest_tn := ev_try.GetCLD()
	err = s.cmap.sshaken.Verify(identity, orig_tn, dest_tn, date_ts)
	if err != nil {
		s.Error("Verification failure: " + err.Error())
	}
	return err == nil
}

func (s *CallController) SshakenAuth(cli, cld string) (string, time.Time, error) {
	date_ts := time.Now()
	identity, err := s.cmap.sshaken.Authenticate(date_ts, cli, cld)
	return identity, date_ts, err
}

func (s *CallController) aDead() {
	s.cmap.Remove(s.id)
}

func (s *CallController) oDead() {
	s.cmap.Remove(s.id)
}

func (s *CallController) Shutdown() {
	s.uaA.Disconnect(nil, "")
}

func (s *CallController) String() string {
	res := "uaA:" + s.uaA.String() + ", uaO: "
	if s.uaO == nil {
		res += "nil"
	} else {
		res += s.uaO.String()
	}
	return res
}
