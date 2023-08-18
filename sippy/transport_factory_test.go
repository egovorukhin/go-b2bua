package sippy

import (
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/time"
)

type TestSipTransportFactory struct {
	recvCb   sippy_net.DataPacketReceiver
	lAddress *sippy_net.HostPort
	dataCh   chan []byte
}

func NewTestSipTransportFactory() *TestSipTransportFactory {
	return &TestSipTransportFactory{
		lAddress: sippy_net.NewHostPort("0.0.0.0", "5060"),
		dataCh:   make(chan []byte, 100),
	}
}

func (f *TestSipTransportFactory) NewSipTransport(addr *sippy_net.HostPort, recvCb sippy_net.DataPacketReceiver) (sippy_net.Transport, error) {
	f.recvCb = recvCb
	return f, nil
}

func (f *TestSipTransportFactory) GetLAddress() *sippy_net.HostPort {
	return f.lAddress
}

func (f *TestSipTransportFactory) SendTo(data []byte, dest *sippy_net.HostPort) {
	f.dataCh <- data
}

func (f *TestSipTransportFactory) SendToWithCb(data []byte, dest *sippy_net.HostPort, cb func()) {
	f.SendTo(data, dest)
	if cb != nil {
		cb()
	}
}

func (f *TestSipTransportFactory) Shutdown() {
}

func (f *TestSipTransportFactory) feed(inp []string) {
	s := strings.Join(inp, "\r\n")
	rtime, _ := sippy_time.NewMonoTime()
	f.recvCb([]byte(s), sippy_net.NewHostPort("1.1.1.1", "5060"), f, rtime)
}

func (f *TestSipTransportFactory) get() []byte {
	return <-f.dataCh
}
