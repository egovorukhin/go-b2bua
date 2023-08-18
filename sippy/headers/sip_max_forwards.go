package sippy_header

import (
	"github.com/sippy/go-b2bua/sippy/net"
)

type SipMaxForwards struct {
	normalName
	SipNumericHF
}

var sipMaxForwardsName normalName = newNormalName("Max-Forwards")

func CreateSipMaxForwards(body string) []SipHeader {
	return []SipHeader{
		&SipMaxForwards{
			normalName:   sipMaxForwardsName,
			SipNumericHF: createSipNumericHF(body),
		},
	}
}

func NewSipMaxForwards(number int) *SipMaxForwards {
	return &SipMaxForwards{
		normalName:   sipMaxForwardsName,
		SipNumericHF: newSipNumericHF(number),
	}
}

func NewSipMaxForwardsDefault() *SipMaxForwards {
	return NewSipMaxForwards(70)
}

func (s *SipMaxForwards) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipMaxForwards) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipMaxForwards) GetCopy() *SipMaxForwards {
	tmp := *s
	return &tmp
}

func (s *SipMaxForwards) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
