package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UaStateFailed struct {
	*uaStateGeneric
}

func NewUaStateFailed(ua sippy_types.UA, config sippy_conf.Config) *UaStateFailed {
	s := &UaStateFailed{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
	ua.ResetOnLocalSdpChange()
	ua.ResetOnRemoteSdpChange()
	return s
}

func (s *UaStateFailed) OnActivation() {
	StartTimeout(s.goDead, s.ua.GetSessionLock(), s.ua.GetGoDeadTimeout(), 1, s.config.ErrorLogger())
}

func (s *UaStateFailed) String() string {
	return "Failed"
}

func (s *UaStateFailed) goDead() {
	//print 'Time in Failed state expired, going to the Dead state'
	s.ua.ChangeState(NewUaStateDead(s.ua, s.config), nil)
}

func (s *UaStateFailed) ID() sippy_types.UaStateID {
	return sippy_types.UA_STATE_FAILED
}
