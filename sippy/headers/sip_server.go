package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipServer struct {
	normalName
	Server string
}

var sipServerName = newNormalName("Server")

func CreateSipServer(body string) []SipHeader {
	return []SipHeader{
		&SipServer{
			normalName: sipServerName,
			Server:     body,
		},
	}
}

func NewSipServer(body string) *SipServer {
	return &SipServer{
		normalName: sipServerName,
		Server:     body,
	}
}

func (s *SipServer) StringBody() string {
	return s.Server
}

func (s *SipServer) String() string {
	return s.Name() + ": " + s.Server
}

func (s *SipServer) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipServer) GetCopy() *SipServer {
	tmp := *s
	return &tmp
}

func (s *SipServer) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipServer) AsSipUserAgent() *SipUserAgent {
	if s == nil {
		return nil
	}
	return NewSipUserAgent(s.Server)
}
