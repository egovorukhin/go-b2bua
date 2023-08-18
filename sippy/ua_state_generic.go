package sippy

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type uaStateGeneric struct {
	ua        sippy_types.UA
	connected bool
	config    sippy_conf.Config
}

func newUaStateGeneric(ua sippy_types.UA, config sippy_conf.Config) *uaStateGeneric {
	return &uaStateGeneric{
		ua:        ua,
		connected: false,
		config:    config,
	}
}

func (s *uaStateGeneric) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	return nil, nil
}

func (s *uaStateGeneric) RecvResponse(resp sippy_types.SipResponse, t sippy_types.ClientTransaction) (sippy_types.UaState, func()) {
	return nil, nil
}

func (s *uaStateGeneric) RecvEvent(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	return nil, nil, nil
}

func (s *uaStateGeneric) RecvCancel(rtime *sippy_time.MonoTime, req sippy_types.SipRequest) {
}

func (*uaStateGeneric) OnDeactivate() {
}

func (*uaStateGeneric) OnActivation() {
}

func (*uaStateGeneric) RecvACK(sippy_types.SipRequest) {
}

func (s *uaStateGeneric) IsConnected() bool {
	return s.connected
}

func (s *uaStateGeneric) RecvPRACK(req sippy_types.SipRequest, resp sippy_types.SipResponse) {
}
