package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipContentType struct {
	compactName
	body string
}

var sipContentTypeName compactName = newCompactName("Content-Type", "c")

func CreateSipContentType(body string) []SipHeader {
	return []SipHeader{
		&SipContentType{
			compactName: sipContentTypeName,
			body:        body,
		},
	}
}

func (s *SipContentType) StringBody() string {
	return s.body
}

func (s *SipContentType) String() string {
	return s.Name() + ": " + s.body
}

func (s *SipContentType) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	if compact {
		return s.CompactName() + ": " + s.body
	}
	return s.String()
}

func (s *SipContentType) GetCopy() *SipContentType {
	tmp := *s
	return &tmp
}

func (s *SipContentType) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
