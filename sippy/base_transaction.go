package sippy

import (
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type sip_transaction_state int

const (
	TRYING = sip_transaction_state(iota)
	RINGING
	COMPLETED
	CONFIRMED
	TERMINATED
	UACK
)

func (s sip_transaction_state) String() string {
	switch s {
	case TRYING:
		return "TRYING"
	case RINGING:
		return "RINGING"
	case COMPLETED:
		return "COMPLETED"
	case CONFIRMED:
		return "CONFIRMED"
	case TERMINATED:
		return "TERMINATED"
	default:
		return "UNKNOWN"
	}
}

type baseTransaction struct {
	lock    sync.Locker
	userv   sippy_net.Transport
	sip_tm  *sipTransactionManager
	state   sip_transaction_state
	tid     *sippy_header.TID
	teA     *Timeout
	address *sippy_net.HostPort
	needack bool
	tout    time.Duration
	data    []byte
	logger  sippy_log.ErrorLogger
}

func newBaseTransaction(lock sync.Locker, tid *sippy_header.TID, userv sippy_net.Transport, sip_tm *sipTransactionManager, address *sippy_net.HostPort, data []byte, needack bool) *baseTransaction {
	return &baseTransaction{
		tout:    time.Duration(0.5 * float64(time.Second)),
		userv:   userv,
		tid:     tid,
		state:   TRYING,
		sip_tm:  sip_tm,
		address: address,
		data:    data,
		needack: needack,
		lock:    lock,
		logger:  sip_tm.config.ErrorLogger(),
	}
}

func (s *baseTransaction) cleanup() {
	s.sip_tm = nil
	s.userv = nil
	s.tid = nil
	s.address = nil
	if s.teA != nil {
		s.teA.Cancel()
		s.teA = nil
	}
}

func (s *baseTransaction) cancelTeA() {
	if s.teA != nil {
		s.teA.Cancel()
		s.teA = nil
	}
}

func (s *baseTransaction) startTeA() {
	if s.teA != nil {
		s.teA.Cancel()
	}
	s.teA = StartTimeout(s.timerA, s.lock, s.tout, 1, s.logger)
}

func (s *baseTransaction) timerA() {
	//print("timerA", t.GetTID())
	if sip_tm := s.sip_tm; sip_tm != nil {
		sip_tm.transmitData(s.userv, s.data, s.address /*cachesum*/, "" /*call_id*/, s.tid.CallId, 0)
		s.tout *= 2
		s.teA = StartTimeout(s.timerA, s.lock, s.tout, 1, s.logger)
	}
}

func (s *baseTransaction) GetHost() string {
	return s.address.Host.String()
}
