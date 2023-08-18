// Copyright (c) 2003-2005 Maxim Sobolev. All rights reserved.
// Copyright (c) 2006-2020 Sippy Software, Inc. All rights reserved.
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
package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UasStateRingingRel struct {
	*UasStateRinging
	prack_received     bool
	prack_wait         bool
	pending_ev_ring    sippy_types.CCEvent
	pending_ev_connect *CCEventConnect
	pending_ev_update  *CCEventUpdate
}

func NewUasStateRingingRel(ua sippy_types.UA, config sippy_conf.Config) *UasStateRingingRel {
	s := &UasStateRingingRel{
		UasStateRinging: NewUasStateRinging(ua, config),
		prack_wait:      false,
	}
	if ua.GetLSDP() != nil {
		s.prack_wait = true
	}
	return s
}

func (s *UasStateRingingRel) RecvEvent(_event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	switch event := _event.(type) {
	case *CCEventRing:
		if !s.prack_received {
			// There is no PRACK for the previous response yet.
			if event.scode > 100 {
				// Memorize the last event
				s.pending_ev_ring = _event
			}
			return nil, nil, nil
		} else {
			s.prack_wait = event.body != nil
			s.prack_received = false
		}
	case *CCEventConnect:
		if s.prack_wait && !s.prack_received {
			// 200 OK received but the last reliable provisional
			// response has not yet been aknowledged. Memorize the event
			// until PRACK is received.
			s.pending_ev_connect = event
			return nil, nil, nil
		}
	case *CCEventUpdate:
		// 200 OK's been received and re-INVITE has arrived but the last
		// reliable provisional response is still not aknowledged.
		// Memorize the event until PRACK is received.
		s.pending_ev_update = event
		return nil, nil, nil
	}
	return s.UasStateRinging.RecvEvent(_event)
}

func (s *UasStateRingingRel) RecvPRACK(req sippy_types.SipRequest, resp sippy_types.SipResponse) {
	var state sippy_types.UaState
	var cb func()
	var err error

	s.prack_received = true
	if s.pending_ev_connect != nil {
		state, cb, err = s.RecvEvent(s.pending_ev_connect)
	} else if s.pending_ev_ring != nil {
		state, cb, err = s.RecvEvent(s.pending_ev_ring)
	}
	if err != nil {
		s.config.ErrorLogger().Error("RecvPRACK: " + err.Error())
	}
	if state != nil {
		s.ua.ChangeState(state, cb)
	}
	if s.pending_ev_update != nil {
		s.ua.RecvEvent(s.pending_ev_update)
	}
	s.pending_ev_ring = nil
	s.pending_ev_connect = nil
	s.pending_ev_update = nil
	body := req.GetBody()
	if body != nil {
		s.ua.SetRSDP(body.GetCopy())
	}
}
