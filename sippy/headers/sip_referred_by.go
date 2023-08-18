package sippy_header

import (
	"github.com/sippy/go-b2bua/sippy/net"
)

type SipReferredBy struct {
	normalName
	*sipAddressHF
}

var sipReferredByName normalName = newNormalName("Referred-By")

func CreateSipReferredBy(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipReferredBy{
			normalName:   sipReferredByName,
			sipAddressHF: addr,
		}
	}
	return rval
}

func NewSipReferredBy(addr *SipAddress) *SipReferredBy {
	return &SipReferredBy{
		normalName:   sipReferredByName,
		sipAddressHF: newSipAddressHF(addr),
	}
}

func (s *SipReferredBy) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipReferredBy) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipReferredBy) GetCopy() *SipReferredBy {
	return &SipReferredBy{
		normalName:   sipReferredByName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipReferredBy) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
