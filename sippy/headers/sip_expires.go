package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipExpires struct {
	normalName
	SipNumericHF
}

var sipExpiresName normalName = newNormalName("Expires")

func NewSipExpires() *SipExpires {
	return &SipExpires{
		normalName:   sipExpiresName,
		SipNumericHF: newSipNumericHF(300),
	}
}

func CreateSipExpires(body string) []SipHeader {
	return []SipHeader{&SipExpires{
		normalName:   sipExpiresName,
		SipNumericHF: createSipNumericHF(body),
	}}
}

func (s *SipExpires) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipExpires) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.String()
}

func (s *SipExpires) GetCopy() *SipExpires {
	tmp := *s
	return &tmp
}

func (s *SipExpires) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
