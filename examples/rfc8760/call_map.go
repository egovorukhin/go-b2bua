//
// Copyright (c) 2003-2005 Maxim Sobolev. All rights reserved.
// Copyright (c) 2006-2021 Sippy Software, Inc. All rights reserved.
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

	"github.com/sippy/go-b2bua/sippy/log"
	"github.com/sippy/go-b2bua/sippy/types"
)

type callMap struct {
	config     *myconfig
	logger     sippy_log.ErrorLogger
	Sip_tm     sippy_types.SipTransactionManager
	Proxy      sippy_types.StatefulProxy
	ccmap      map[int64]*callController
	ccmap_lock sync.Mutex
}

func NewCallMap(config *myconfig, logger sippy_log.ErrorLogger) (*callMap, error) {
	var err error

	ret := &callMap{
		logger: logger,
		config: config,
		ccmap:  make(map[int64]*callController),
	}
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *callMap) OnNewDialog(req sippy_types.SipRequest, tr sippy_types.ServerTransaction) (sippy_types.UA, sippy_types.RequestReceiver, sippy_types.SipResponse) {
	to_body, err := req.GetTo().GetBody(s.config)
	if err != nil {
		s.logger.Error("CallMap::OnNewDialog: #1: " + err.Error())
		return nil, nil, req.GenResponse(500, "Internal Server Error", nil, nil)
	}
	if to_body.GetTag() != "" {
		// Request within dialog, but no such dialog
		return nil, nil, req.GenResponse(481, "Call Leg/Transaction Does Not Exist", nil, nil)
	}
	if req.GetMethod() == "INVITE" {
		// New dialog
		identity_hf := req.GetFirstHF("identity")
		date_hf := req.GetSipDate()
		cc := NewCallController(s, identity_hf, date_hf)
		s.ccmap_lock.Lock()
		s.ccmap[cc.id] = cc
		s.ccmap_lock.Unlock()
		return cc.uaA, cc.uaA, nil
	}
	if req.GetMethod() == "REGISTER" {
		// Registration
		return nil, s.Proxy, nil
	}
	if req.GetMethod() == "NOTIFY" || req.GetMethod() == "PING" {
		// Whynot?
		return nil, nil, req.GenResponse(200, "OK", nil, nil)
	}
	return nil, nil, req.GenResponse(501, "Not Implemented", nil, nil)
}

func (s *callMap) Remove(ccid int64) {
	s.ccmap_lock.Lock()
	defer s.ccmap_lock.Unlock()
	delete(s.ccmap, ccid)
}

func (s *callMap) Shutdown() {
	acalls := []*callController{}
	s.ccmap_lock.Lock()
	for _, cc := range s.ccmap {
		acalls = append(acalls, cc)
	}
	s.ccmap_lock.Unlock()
	for _, cc := range acalls {
		//println(cc.String())
		cc.Shutdown()
	}
}
