package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type sipGenericHF struct {
	name string
	body string
}

func NewSipGenericHF(name, body string) *sipGenericHF {
	return &sipGenericHF{
		name: name,
		body: body,
	}
}

func (s *sipGenericHF) StringBody() string {
	return s.body
}

func (s *sipGenericHF) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.String()
}

func (s *sipGenericHF) String() string {
	return s.name + ": " + s.StringBody()
}

func (s *sipGenericHF) GetCopy() *sipGenericHF {
	ret := *s
	return &ret
}

func (s *sipGenericHF) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *sipGenericHF) Name() string {
	return s.name
}

func (s *sipGenericHF) CompactName() string {
	return s.name
}
