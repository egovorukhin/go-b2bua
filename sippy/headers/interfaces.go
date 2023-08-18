package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipHeader interface {
	Name() string
	CompactName() string
	String() string
	StringBody() string
	LocalStr(hostPort *sippy_net.HostPort, compact bool) string
	GetCopyAsIface() SipHeader
}

type SipAuthorizationHeader interface {
	SipHeader
	GetBody() (*SipAuthorizationBody, error)
}
