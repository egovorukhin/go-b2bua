package sippy_header

import (
	"github.com/sippy/go-b2bua/sippy/net"
)

type SipCCDiversion struct {
	normalName
	*sipAddressHF
}

var sipCcDiversionName normalName = newNormalName("CC-Diversion")

func CreateSipCCDiversion(body string) []SipHeader {
	addresses := CreateSipAddressHFs(body)
	rval := make([]SipHeader, len(addresses))
	for i, addr := range addresses {
		rval[i] = &SipCCDiversion{
			normalName:   sipCcDiversionName,
			sipAddressHF: addr,
		}
	}
	return rval
}

func (d *SipCCDiversion) String() string {
	return d.LocalStr(nil, false)
}

func (d *SipCCDiversion) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return d.Name() + ": " + d.LocalStringBody(hostPort)
}

func (d *SipCCDiversion) GetCopy() *SipCCDiversion {
	rval := &SipCCDiversion{
		normalName:   sipCcDiversionName,
		sipAddressHF: d.sipAddressHF.getCopy(),
	}
	return rval
}

func (d *SipCCDiversion) GetCopyAsIface() SipHeader {
	return d.GetCopy()
}
