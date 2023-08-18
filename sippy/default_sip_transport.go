package sippy

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type defaultSipTransportFactory struct {
	config sippy_conf.Config
}

func NewDefaultSipTransportFactory(config sippy_conf.Config) *defaultSipTransportFactory {
	return &defaultSipTransportFactory{
		config: config,
	}
}

func (s *defaultSipTransportFactory) NewSipTransport(laddress *sippy_net.HostPort, handler sippy_net.DataPacketReceiver) (sippy_net.Transport, error) {
	sopts := NewUdpServerOpts(laddress, handler)
	return NewUdpServer(s.config, sopts)
}
