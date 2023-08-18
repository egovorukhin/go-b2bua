package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipUserAgent struct {
	normalName
	UserAgent string
}

var sipUserAgent normalName = newNormalName("User-Agent")

func NewSipUserAgent(name string) *SipUserAgent {
	return &SipUserAgent{
		normalName: sipUserAgent,
		UserAgent:  name,
	}
}

func CreateSipUserAgent(body string) []SipHeader {
	return []SipHeader{
		&SipUserAgent{
			normalName: sipUserAgent,
			UserAgent:  body,
		},
	}
}

func (s *SipUserAgent) StringBody() string {
	return s.UserAgent
}

func (s *SipUserAgent) String() string {
	return s.Name() + ": " + s.UserAgent
}

func (s *SipUserAgent) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipUserAgent) AsSipServer() *SipServer {
	if s == nil {
		return nil
	}
	return NewSipServer(s.UserAgent)
}

func (s *SipUserAgent) GetCopy() *SipUserAgent {
	tmp := *s
	return &tmp
}

func (s *SipUserAgent) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
