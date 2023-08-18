package sippy_net

import (
	"github.com/egovorukhin/go-b2bua/sippy/time"
)

type DataPacketReceiver func(data []byte, addr *HostPort, server Transport, rtime *sippy_time.MonoTime)

type SipTransportFactory interface {
	NewSipTransport(*HostPort, DataPacketReceiver) (Transport, error)
}

type Transport interface {
	Shutdown()
	GetLAddress() *HostPort
	SendTo([]byte, *HostPort)
	SendToWithCb([]byte, *HostPort, func())
}
