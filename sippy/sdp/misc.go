package sippy_sdp

import net "github.com/egovorukhin/go-b2bua/sippy/net"

type SdpHeader interface {
	String() string
	LocalStr(hostPort *net.HostPort) string
}

type SdpHeaderAndName struct {
	Name   string
	Header SdpHeader
}
