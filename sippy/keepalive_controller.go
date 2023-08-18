package sippy

import (
	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type keepaliveController struct {
	ua         sippy_types.UA
	triedauth  bool
	ka_tr      sippy_types.ClientTransaction
	keepalives int
	logger     sippy_log.ErrorLogger
}

func newKeepaliveController(ua sippy_types.UA, logger sippy_log.ErrorLogger) *keepaliveController {
	if ua.GetKaInterval() <= 0 {
		return nil
	}
	s := &keepaliveController{
		ua:         ua,
		triedauth:  false,
		keepalives: 0,
		logger:     logger,
	}
	return s
}

func (s *keepaliveController) Start() {
	StartTimeout(s.keepAlive, s.ua.GetSessionLock(), s.ua.GetKaInterval(), 1, s.logger)
}

func (s *keepaliveController) RecvResponse(resp sippy_types.SipResponse, tr sippy_types.ClientTransaction) {
	var err error
	var challenge sippy_types.Challenge
	var req sippy_types.SipRequest

	if s.ua.GetState() != sippy_types.UA_STATE_CONNECTED {
		return
	}
	code, _ := resp.GetSCode()
	if s.ua.GetUsername() != "" && s.ua.GetPassword() != "" && !s.triedauth {
		if code == 401 && len(resp.GetSipWWWAuthenticates()) > 0 {
			challenge = resp.GetSipWWWAuthenticates()[0]
			if err != nil {
				s.logger.Error("error parsing 401 auth: " + err.Error())
				return
			}
		} else if code == 407 && len(resp.GetSipProxyAuthenticates()) > 0 {
			challenge = resp.GetSipProxyAuthenticates()[0]
			if err != nil {
				s.logger.Error("error parsing 407 auth: " + err.Error())
				return
			}
		}
		if challenge != nil {
			req, err = s.ua.GenRequest("INVITE", s.ua.GetLSDP(), challenge)
			if err != nil {
				s.logger.Error("Cannot create INVITE: " + err.Error())
				return
			}
			s.ka_tr, err = s.ua.PrepTr(req, nil)
			if err == nil {
				s.triedauth = true
			}
			s.ua.BeginClientTransaction(req, s.ka_tr)
			return
		}
	}
	if code < 200 {
		return
	}
	s.ka_tr = nil
	s.keepalives += 1
	if code == 408 || code == 481 || code == 486 {
		if s.keepalives == 1 {
			//print "%s: Remote UAS at %s:%d does not support re-INVITES, disabling keep alives" % (s.ua.cId, s.ua.rAddr[0], s.ua.rAddr[1])
			StartTimeout(func() { s.ua.Disconnect(nil, "") }, s.ua.GetSessionLock(), 600, 1, s.logger)
			return
		}
		//print "%s: Received %d response to keep alive from %s:%d, disconnecting the call" % (s.ua.cId, code, s.ua.rAddr[0], s.ua.rAddr[1])
		s.ua.Disconnect(nil, "")
		return
	}
	StartTimeout(s.keepAlive, s.ua.GetSessionLock(), s.ua.GetKaInterval(), 1, s.logger)
}

func (s *keepaliveController) keepAlive() {
	var err error
	var req sippy_types.SipRequest

	if s.ua.GetState() != sippy_types.UA_STATE_CONNECTED {
		return
	}
	req, err = s.ua.GenRequest("INVITE", s.ua.GetLSDP(), nil)
	if err != nil {
		s.logger.Error("Cannot create INVITE: " + err.Error())
		return
	}
	s.triedauth = false
	s.ka_tr, err = s.ua.PrepTr(req, nil)
	if err == nil {
		s.ua.BeginClientTransaction(req, s.ka_tr)
	}
}

func (s *keepaliveController) Stop() {
	if ka_tr := s.ka_tr; ka_tr != nil {
		ka_tr.Cancel()
		s.ka_tr = nil
	}
}
