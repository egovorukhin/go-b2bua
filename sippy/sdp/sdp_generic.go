package sippy_sdp

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SdpGeneric struct {
	body string
}

func ParseSdpGeneric(body string) *SdpGeneric {
	return &SdpGeneric{body}
}

func (s *SdpGeneric) String() string {
	return s.body
}

func (s *SdpGeneric) LocalStr(hostPort *sippy_net.HostPort) string {
	return s.String()
}

func (s *SdpGeneric) GetCopy() *SdpGeneric {
	if s == nil {
		return nil
	}
	return &SdpGeneric{s.body}
}
