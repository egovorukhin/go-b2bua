package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipRequire struct {
	normalName
	tagListHF
}

var sipRequireName normalName = newNormalName("Require")

func CreateSipRequire(body string) []SipHeader {
	return []SipHeader{
		&SipRequire{
			normalName: sipRequireName,
			tagListHF:  *createTagListHF(body),
		},
	}
}

func (s *SipRequire) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipRequire) GetCopy() *SipRequire {
	tmp := *s
	tmp.tagListHF = *s.tagListHF.getCopy()
	return &tmp
}

func (s *SipRequire) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipRequire) String() string {
	return s.Name() + ": " + s.StringBody()
}
