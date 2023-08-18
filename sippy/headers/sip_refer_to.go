package sippy_header

import (
	"github.com/sippy/go-b2bua/sippy/net"
)

type SipReferTo struct {
	compactName
	*sipAddressHF
}

var sipReferToName compactName = newCompactName("Refer-To", "r")

func CreateSipReferTo(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipReferTo{
			compactName:  sipReferToName,
			sipAddressHF: addr,
		}
	}
	return rval
}

func NewSipReferTo(addr *SipAddress) *SipReferTo {
	return &SipReferTo{
		compactName:  sipReferToName,
		sipAddressHF: newSipAddressHF(addr),
	}
}

func (s *SipReferTo) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipReferTo) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	prefix := s.Name()
	if compact {
		prefix = s.CompactName()
	}
	return prefix + ": " + s.LocalStringBody(hostPort)
}

func (s *SipReferTo) AsSipAlso() *SipAlso {
	return &SipAlso{
		normalName:   _sip_also_name,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipReferTo) GetCopy() *SipReferTo {
	return &SipReferTo{
		compactName:  sipReferToName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipReferTo) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
