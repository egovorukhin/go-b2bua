package sippy

import (
	"strings"

	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
)

type test_sip_transport_factory struct {
	recv_cb  sippy_net.DataPacketReceiver
	laddress *sippy_net.HostPort
	data_ch  chan []byte
}

func NewTestSipTransportFactory() *test_sip_transport_factory {
	return &test_sip_transport_factory{
		laddress: sippy_net.NewHostPort("0.0.0.0", "5060"),
		data_ch:  make(chan []byte, 100),
	}
}

func (s *test_sip_transport_factory) NewSipTransport(addr *sippy_net.HostPort, recv_cb sippy_net.DataPacketReceiver) (sippy_net.Transport, error) {
	s.recv_cb = recv_cb
	return s, nil
}

func (s *test_sip_transport_factory) GetLAddress() *sippy_net.HostPort {
	return s.laddress
}

func (s *test_sip_transport_factory) SendTo(data []byte, dest *sippy_net.HostPort) {
	s.data_ch <- data
}

func (s *test_sip_transport_factory) SendToWithCb(data []byte, dest *sippy_net.HostPort, cb func()) {
	s.SendTo(data, dest)
	if cb != nil {
		cb()
	}
}

func (s *test_sip_transport_factory) Shutdown() {
}

func (s *test_sip_transport_factory) feed(inp []string) {
	s := strings.Join(inp, "\r\n")
	rtime, _ := sippy_time.NewMonoTime()
	s.recv_cb([]byte(s), sippy_net.NewHostPort("1.1.1.1", "5060"), s, rtime)
}

func (s *test_sip_transport_factory) get() []byte {
	return <-s.data_ch
}
