//
// Copyright (c) 2003-2005 Maxim Sobolev. All rights reserved.
// Copyright (c) 2019 Sippy Software, Inc. All rights reserved.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"sync"

	"github.com/egovorukhin/go-b2bua/sippy"
	//"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

var Next_cc_id chan int64

type callController struct {
	uaA                     sippy_types.UA
	uaO                     sippy_types.UA
	lock                    *sync.Mutex // this must be a reference to prevent memory leak
	id                      int64
	cmap                    *CallMap
	evTry                   *sippy.CCEventTry
	transfer_is_in_progress bool
}

func NewCallController(cmap *CallMap) *callController {
	s := &callController{
		id:                      <-Next_cc_id,
		uaO:                     nil,
		lock:                    new(sync.Mutex),
		cmap:                    cmap,
		transfer_is_in_progress: false,
	}
	s.uaA = sippy.NewUA(cmap.Sip_tm, cmap.config, cmap.config.Nh_addr, s, s.lock, nil)
	s.uaA.SetDeadCb(s.aDead)
	//s.uaA.SetCreditTime(5 * time.Second)
	return s
}

func (s *callController) handle_transfer(event sippy_types.CCEvent, ua sippy_types.UA) {
	switch ua {
	case s.uaA:
		if _, ok := event.(*sippy.CCEventConnect); ok {
			// Transfer is completed.
			s.transfer_is_in_progress = false
		}
		s.uaO.RecvEvent(event)
	case s.uaO:
		if _, ok := event.(*sippy.CCEventPreConnect); ok {
			//
			// Convert into CCEventUpdate.
			//
			// Here 200 OK response from the new callee has been received
			// and now re-INVITE will be sent to the caller.
			//
			// The CCEventPreConnect is here because the outgoing call to the
			// new destination has been sent using the late offer model, i.e.
			// the outgoing INVITE was body-less.
			//
			event = sippy.NewCCEventUpdate(event.GetRtime(), event.GetOrigin(), event.GetReason(),
				event.GetMaxForwards(), event.GetBody().GetCopy())
		}
		s.uaA.RecvEvent(event)
	}
}

func (s *callController) RecvEvent(event sippy_types.CCEvent, ua sippy_types.UA) {
	if s.transfer_is_in_progress {
		s.handle_transfer(event, ua)
		return
	}
	if ua == s.uaA {
		if s.uaO == nil {
			ev_try, ok := event.(*sippy.CCEventTry)
			if !ok {
				// Some weird event received
				s.uaA.RecvEvent(sippy.NewCCEventDisconnect(nil, event.GetRtime(), ""))
				return
			}
			s.uaO = sippy.NewUA(s.cmap.Sip_tm, s.cmap.config, s.cmap.config.Nh_addr, s, s.lock, nil)
			s.uaO.SetRAddr(s.cmap.config.Nh_addr)
			s.evTry = ev_try
		}
		s.uaO.RecvEvent(event)
	} else {
		if ev_disc, ok := event.(*sippy.CCEventDisconnect); ok {
			redirect_url := ev_disc.GetRedirectURL()
			if redirect_url != nil {
				//
				// Either REFER or a BYE with Also: has been received from the callee.
				//
				// Do not interrupt the caller call leg and create a new call leg
				// to the new destination.
				//
				cld := redirect_url.GetUrl().Username

				//nh_addr := &sippy_net.HostPort{ redirect_url.GetUrl().Host, redirect_url.GetUrl().Port }
				nh_addr := s.cmap.config.Nh_addr

				s.uaO = sippy.NewUA(s.cmap.Sip_tm, s.cmap.config, nh_addr, s, s.lock, nil)
				ev_try, _ := sippy.NewCCEventTry(s.evTry.GetSipCallId(),
					s.evTry.GetCLI(), cld, nil /*body*/, nil /*auth*/, s.evTry.GetCallerName(),
					ev_disc.GetRtime(), s.evTry.GetOrigin())
				s.transfer_is_in_progress = true
				s.uaO.RecvEvent(ev_try)
				return
			}
		}
		s.uaA.RecvEvent(event)
	}
}

func (s *callController) aDead() {
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
