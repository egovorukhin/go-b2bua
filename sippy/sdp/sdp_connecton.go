package sippy_sdp

import (
	"strings"

	"github.com/sippy/go-b2bua/sippy/net"
)

type SdpConnecton struct {
	ntype string
	atype string
	addr  string
}

func ParseSdpConnecton(body string) *SdpConnecton {
	arr := strings.Fields(body)
	if len(arr) != 3 {
		return nil
	}
	return &SdpConnecton{
		ntype: arr[0],
		atype: arr[1],
		addr:  arr[2],
	}
}

func (s *SdpConnecton) String() string {
	return s.ntype + " " + s.atype + " " + s.addr
}

func (s *SdpConnecton) LocalStr(hostPort *sippy_net.HostPort) string {
	return s.String()
}

func (s *SdpConnecton) GetCopy() *SdpConnecton {
	if s == nil {
		return nil
	}
	var ret SdpConnecton = *s
	return &ret
}

func (s *SdpConnecton) GetAddr() string {
	return s.addr
}

func (s *SdpConnecton) SetAddr(addr string) {
	s.addr = addr
}

func (s *SdpConnecton) GetAType() string {
	return s.atype
}

func (s *SdpConnecton) SetAType(atype string) {
	s.atype = atype
}
