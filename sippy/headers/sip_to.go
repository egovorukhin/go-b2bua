package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipTo struct {
	compactName
	*sipAddressHF
}

var sipToName = newCompactName("To", "t")

func NewSipTo(address *SipAddress, config sippy_conf.Config) *SipTo {
	if address == nil {
		address = NewSipAddress("Anonymous", NewSipURL("", /* username */
			config.GetMyAddress(),
			config.GetMyPort(),
			false))
	}
	return &SipTo{
		compactName:  sipToName,
		sipAddressHF: newSipAddressHF(address),
	}
}

func CreateSipTo(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipTo{
			compactName:  sipToName,
			sipAddressHF: addr,
		}
	}
	return rval
}

func (s *SipTo) GetCopy() *SipTo {
	return &SipTo{
		compactName:  sipToName,
		sipAddressHF: s.sipAddressHF.getCopy(),
	}
}

func (s *SipTo) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipTo) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipTo) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	if compact {
		return s.CompactName() + ": " + s.LocalStringBody(hostPort)
	}
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}
