package main

import (
	"sync"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type CallMap struct {
	config     *Myconfig
	logger     sippy_log.ErrorLogger
	Sip_tm     sippy_types.SipTransactionManager
	Proxy      sippy_types.StatefulProxy
	ccmap      map[int64]*callController
	ccmap_lock sync.Mutex
}

func NewCallMap(config *Myconfig, logger sippy_log.ErrorLogger) *CallMap {
	return &CallMap{
		logger: logger,
		config: config,
		ccmap:  make(map[int64]*callController),
	}
}

func (s *CallMap) OnNewDialog(req sippy_types.SipRequest, tr sippy_types.ServerTransaction) (sippy_types.UA, sippy_types.RequestReceiver, sippy_types.SipResponse) {
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
		cc := NewCallController(s)
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

func (s *CallMap) Remove(ccid int64) {
	s.ccmap_lock.Lock()
	defer s.ccmap_lock.Unlock()
	delete(s.ccmap, ccid)
}

func (s *CallMap) Shutdown() {
	acalls := []*callController{}
	s.ccmap_lock.Lock()
	for _, cc := range s.ccmap {
		//println(cc.String())
		acalls = append(acalls, cc)
	}
	s.ccmap_lock.Unlock()
	for _, cc := range acalls {
		cc.Shutdown()
	}
}

type Myconfig struct {
	sippy_conf.Config

	Nh_addr *sippy_net.HostPort
}
