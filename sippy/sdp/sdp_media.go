package sippy_sdp

import (
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SdpMedia struct {
	stype     string
	port      string
	transport string
	formats   []string
}

func ParseSdpMedia(body string) *SdpMedia {
	if body == "" {
		return nil
	}
	params := strings.Fields(body)
	if len(params) < 3 {
		return nil
	}
	return &SdpMedia{
		stype:     params[0],
		port:      params[1],
		transport: params[2],
		formats:   params[3:],
	}
}

func (s *SdpMedia) String() string {
	rval := s.stype + " " + s.port + " " + s.transport
	for _, format := range s.formats {
		rval += " " + format
	}
	return rval
}

func (s *SdpMedia) LocalStr(hostPort *sippy_net.HostPort) string {
	return s.String()
}

func (s *SdpMedia) GetCopy() *SdpMedia {
	if s == nil {
		return nil
	}
	formats := make([]string, len(s.formats))
	copy(formats, s.formats)
	return &SdpMedia{
		stype:     s.stype,
		port:      s.port,
		transport: s.transport,
		formats:   formats,
	}
}

func (s *SdpMedia) GetTransport() string {
	return s.transport
}

func (s *SdpMedia) GetPort() string {
	return s.port
}

func (s *SdpMedia) SetPort(port string) {
	s.port = port
}

func (s *SdpMedia) HasFormat(format string) bool {
	for _, f := range s.formats {
		if f == format {
			return true
		}
	}
	return false
}

func (s *SdpMedia) GetFormats() []string {
	return s.formats
}

// WARNING! Use this function only if know what you do!
// Otherwise consider using the sdpMediaDescription.SetFormats() instead.
func (s *SdpMedia) SetFormats(formats []string) {
	s.formats = formats
}
