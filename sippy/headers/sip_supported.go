package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipSupported struct {
	normalName
	tagListHF
}

var sipSupportedName normalName = newNormalName("Supported")

func CreateSipSupported(body string) []SipHeader {
	return []SipHeader{
		&SipSupported{
			normalName: sipSupportedName,
			tagListHF:  *createTagListHF(body),
		},
	}
}

func (s *SipSupported) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipSupported) GetCopy() *SipSupported {
	tmp := *s
	tmp.tagListHF = *s.tagListHF.getCopy()
	return &tmp
}

func (s *SipSupported) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipSupported) String() string {
	return s.Name() + ": " + s.StringBody()
}
