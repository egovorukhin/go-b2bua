package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipRoute struct {
	normalName
	*sipAddressHF
}

var sipRouteName normalName = newNormalName("Route")

func NewSipRoute(addr *SipAddress) *SipRoute {
	return &SipRoute{
		normalName:   sipRouteName,
		sipAddressHF: newSipAddressHF(addr),
	}
}

func CreateSipRoute(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipRoute{
			normalName:   sipRouteName,
			sipAddressHF: addr,
		}
	}
	return rval
}

func (s *SipRoute) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipRoute) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipRoute) GetCopy() *SipRoute {
	return &SipRoute{
		normalName:   sipRouteName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipRoute) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
