package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipAlso struct {
	normalName
	*sipAddressHF
}

var _sip_also_name normalName = newNormalName("Also")

func CreateSipAlso(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipAlso{
			normalName:   _sip_also_name,
			sipAddressHF: addr,
		}
	}
	return rval
}

func NewSipAlso(addr *SipAddress) *SipAlso {
	return &SipAlso{
		normalName:   _sip_also_name,
		sipAddressHF: newSipAddressHF(addr),
	}
}

func (a *SipAlso) String() string {
	return a.LocalStr(nil, false)
}

func (a *SipAlso) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return a.Name() + ": " + a.LocalStringBody(hostPort)
}

func (a *SipAlso) GetCopy() *SipAlso {
	if a == nil {
		return nil
	}
	return &SipAlso{
		normalName:   _sip_also_name,
		sipAddressHF: a.sipAddressHF.getCopy(),
	}
}

func (a *SipAlso) GetCopyAsIface() SipHeader {
	return a.GetCopy()
}
