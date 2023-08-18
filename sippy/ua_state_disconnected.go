package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UaStateDisconnected struct {
	*uaStateGeneric
}

func NewUaStateDisconnected(ua sippy_types.UA, config sippy_conf.Config) *UaStateDisconnected {
	s := &UaStateDisconnected{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
	ua.ResetOnLocalSdpChange()
	ua.ResetOnRemoteSdpChange()
	return s
}

func (s *UaStateDisconnected) OnActivation() {
	StartTimeout(s.goDead, s.ua.GetSessionLock(), s.ua.GetGoDeadTimeout(), 1, s.config.ErrorLogger())
}

func (s *UaStateDisconnected) String() string {
	return "Disconnected"
}

func (s *UaStateDisconnected) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	if req.GetMethod() == "BYE" {
		//print "BYE received in the Disconnected state"
		t.SendResponse(req.GenResponse(200, "OK", nil /*server*/, s.ua.GetLocalUA().AsSipServer()), false, nil)
	} else {
		t.SendResponse(req.GenResponse(500, "Disconnected", nil /*server*/, s.ua.GetLocalUA().AsSipServer()), false, nil)
	}
	return nil, nil
}

func (s *UaStateDisconnected) goDead() {
	//print "Time in Disconnected state expired, going to the Dead state"
	s.ua.ChangeState(NewUaStateDead(s.ua, s.config), nil)
}

func (s *UaStateDisconnected) ID() sippy_types.UaStateID {
	return sippy_types.UA_STATE_DISCONNECTED
}
