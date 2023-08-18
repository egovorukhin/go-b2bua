package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipDiversion struct {
	normalName
	*sipAddressHF
}

var sipDiversionName normalName = newNormalName("Diversion")

func NewSipDiversion(addr *SipAddress) *SipDiversion {
	return &SipDiversion{
		normalName:   sipDiversionName,
		sipAddressHF: newSipAddressHF(addr),
	}
}

func CreateSipDiversion(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipDiversion{
			normalName:   sipDiversionName,
			sipAddressHF: addr,
		}
	}
	return rval
}

func (s *SipDiversion) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipDiversion) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipDiversion) GetCopy() *SipDiversion {
	return &SipDiversion{
		normalName:   sipDiversionName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipDiversion) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
