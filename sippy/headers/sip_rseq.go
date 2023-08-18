package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipRSeq struct {
	normalName
	SipNumericHF
}

var sipRseqName = newNormalName("RSeq")

func NewSipRSeq() *SipRSeq {
	return &SipRSeq{
		normalName:   sipRseqName,
		SipNumericHF: newSipNumericHF(1),
	}
}

func CreateSipRSeq(body string) []SipHeader {
	return []SipHeader{&SipRSeq{
		normalName:   sipRseqName,
		SipNumericHF: createSipNumericHF(body),
	}}
}

func (s *SipRSeq) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipRSeq) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.String()
}

func (s *SipRSeq) GetCopy() *SipRSeq {
	tmp := *s
	return &tmp
}

func (s *SipRSeq) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
