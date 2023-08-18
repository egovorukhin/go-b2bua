package sippy

import (
	"sync"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/types"
)

type statefulProxy struct {
	sip_tm      sippy_types.SipTransactionManager
	destination *sippy_net.HostPort
	config      sippy_conf.Config
}

func NewStatefulProxy(sip_tm sippy_types.SipTransactionManager, destination *sippy_net.HostPort, config sippy_conf.Config) *statefulProxy {
	return &statefulProxy{
		sip_tm:      sip_tm,
		destination: destination,
		config:      config,
	}
}

func (s *statefulProxy) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) *sippy_types.UaContext {
	via0 := sippy_header.NewSipVia(s.config)
	via0_body, _ := via0.GetBody()
	via0_body.GenBranch()
	req.InsertFirstVia(via0)
	req.SetTarget(s.destination)
	//print req
	s.sip_tm.BeginNewClientTransaction(req, s, new(sync.Mutex), nil, nil, nil)
	return &sippy_types.UaContext{}
}

func (s *statefulProxy) RecvResponse(resp sippy_types.SipResponse, t sippy_types.ClientTransaction) {
	resp.RemoveFirstVia()
	s.sip_tm.SendResponse(resp /*lock*/, true, nil)
}
