package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipFrom struct {
	compactName
	*sipAddressHF
}

var sipFromName compactName = newCompactName("From", "f")

func CreateSipFrom(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, address := range addresses {
		rval[i] = &SipFrom{
			compactName:  sipFromName,
			sipAddressHF: address,
		}
	}
	return rval
}

func NewSipFrom(address *SipAddress, config sippy_conf.Config) *SipFrom {
	if address == nil {
		address = NewSipAddress("Anonymous", NewSipURL("", /* username */
			config.GetMyAddress(),
			config.GetMyPort(),
			false))
	}
	return &SipFrom{
		compactName:  sipFromName,
		sipAddressHF: newSipAddressHF(address),
	}
}

func (s *SipFrom) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipFrom) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	if compact {
		return s.CompactName() + ": " + s.LocalStringBody(hostPort)
	}
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipFrom) GetCopy() *SipFrom {
	return &SipFrom{
		compactName:  sipFromName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipFrom) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
