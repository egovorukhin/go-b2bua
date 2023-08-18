package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UacStateCancelling struct {
	*uaStateGeneric
	te *Timeout
}

func NewUacStateCancelling(ua sippy_types.UA, config sippy_conf.Config) *UacStateCancelling {
	s := &UacStateCancelling{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
	ua.ResetOnLocalSdpChange()
	ua.ResetOnRemoteSdpChange()
	// 300 provides good estimate on the amount of time during which
	// we can wait for receiving non-negative response to CANCELled
	// INVITE transaction.
	return s
}

func (s *UacStateCancelling) OnActivation() {
	s.te = StartTimeout(s.goIdle, s.ua.GetSessionLock(), 300.0, 1, s.config.ErrorLogger())
}

func (s *UacStateCancelling) String() string {
	return "Cancelling(UAC)"
}

func (s *UacStateCancelling) goIdle() {
	//print "Time in Cancelling state expired, going to the Dead state"
	s.te = nil
	s.ua.ChangeState(NewUaStateDead(s.ua, s.config), nil)
}

func (s *UacStateCancelling) RecvResponse(resp sippy_types.SipResponse, tr sippy_types.ClientTransaction) (sippy_types.UaState, func()) {
	code, _ := resp.GetSCode()
	if code < 200 {
		return nil, nil
	}
	if s.te != nil {
		s.te.Cancel()
		s.te = nil
	}
	// When the final response arrives make sure to send BYE
	// if response is positive 200 OK and move into
	// UaStateDisconnected to catch any in-flight BYE from the
	// called party.
	//
	// If the response is negative or redirect go to the UaStateDead
	// immediately, since this means that we won"t receive any more
	// requests from the calling party. XXX: redirects should probably
	// somehow reported to the upper level, but it will create
	// significant additional complexity there, since after signalling
	// Failure/Disconnect calling party don"t expect any more
	// events to be delivered from the called one. In any case,
	// this should be fine, since we are in this state only when
	// caller already has declared his wilingless to end the session,
	// so that he is probably isn"t interested in redirects anymore.
	if code >= 200 && code < 300 {
		var err error
		var rUri *sippy_header.SipAddress
		var to_body *sippy_header.SipAddress
		var req sippy_types.SipRequest

		s.ua.UpdateRouting(resp, true, true)
		rUri, err = s.ua.GetRUri().GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateCancelling::RecvResponse: #1: " + err.Error())
			return nil, nil
		}
		to_body, err = resp.GetTo().GetBody(s.config)
		rUri.SetTag(to_body.GetTag())
		req, err = s.ua.GenRequest("BYE", nil, nil)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateCancelling::RecvResponse: #2: " + err.Error())
			return nil, nil
		}
		s.ua.BeginNewClientTransaction(req, nil)
		return NewUaStateDisconnected(s.ua, s.config), nil
	}
	return NewUaStateDead(s.ua, s.config), nil
}

func (s *UacStateCancelling) ID() sippy_types.UaStateID {
	return sippy_types.UAC_STATE_CANCELLING
}
