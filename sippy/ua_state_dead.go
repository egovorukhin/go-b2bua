package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UaStateDead struct {
	*uaStateGeneric
}

func NewUaStateDead(ua sippy_types.UA, config sippy_conf.Config) *UaStateDead {
	return &UaStateDead{
		uaStateGeneric: newUaStateGeneric(ua, config),
	}
}

func (s *UaStateDead) OnActivation() {
	s.ua.OnDead()
	// Break cross-ref chain
	s.ua.Cleanup()
	s.ua = nil
}

func (s *UaStateDead) String() string {
	return "Dead"
}

func (s *UaStateDead) ID() sippy_types.UaStateID {
	return sippy_types.UA_STATE_DEAD
}
