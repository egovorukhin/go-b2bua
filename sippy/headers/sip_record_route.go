package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipRecordRoute struct {
	normalName
	*sipAddressHF
}

var sipRecordRouteName normalName = newNormalName("Record-Route")

func CreateSipRecordRoute(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, address := range addresses {
		rval[i] = &SipRecordRoute{
			normalName:   sipRecordRouteName,
			sipAddressHF: address,
		}
	}
	return rval
}

func (s *SipRecordRoute) GetCopy() *SipRecordRoute {
	return &SipRecordRoute{
		normalName:   sipRecordRouteName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipRecordRoute) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipRecordRoute) String() string {
	return s.Name() + ": " + s.LocalStringBody(nil)
}

func (s *SipRecordRoute) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipRecordRoute) AsSipRoute() *SipRoute {
	return &SipRoute{
		normalName:   sipRouteName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}
