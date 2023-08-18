package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

var sipContentLengthName compactName = newCompactName("Content-Length", "l")

type SipContentLength struct {
	compactName
	SipNumericHF
}

func CreateSipContentLength(body string) []SipHeader {
	return []SipHeader{&SipContentLength{
		compactName:  sipContentLengthName,
		SipNumericHF: createSipNumericHF(body),
	}}
}

func (s *SipContentLength) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipContentLength) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	if compact {
		return s.CompactName() + ": " + s.StringBody()
	}
	return s.String()
}

func (s *SipContentLength) GetCopy() *SipContentLength {
	tmp := *s
	return &tmp
}

func (s *SipContentLength) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
