package sippy_sdp

import (
	"fmt"
	"strings"

	"github.com/sippy/go-b2bua/sippy/net"
)

type SdpMediaDescription struct {
	m_header  *SdpMedia
	i_header  *SdpGeneric
	c_header  *SdpConnecton
	b_header  *SdpGeneric
	k_header  *SdpGeneric
	a_headers []string
}

func (s *SdpMediaDescription) GetCopy() *SdpMediaDescription {
	a_headers := make([]string, len(s.a_headers))
	copy(a_headers, s.a_headers)
	return &SdpMediaDescription{
		m_header:  s.m_header.GetCopy(),
		i_header:  s.i_header.GetCopy(),
		c_header:  s.c_header.GetCopy(),
		b_header:  s.b_header.GetCopy(),
		k_header:  s.k_header.GetCopy(),
		a_headers: a_headers,
	}
}

func NewSdpMediaDescription() *SdpMediaDescription {
	return &SdpMediaDescription{
		a_headers: make([]string, 0),
	}
}

func (s *SdpMediaDescription) all_headers() []*Sdp_header_and_name {
	ret := []*Sdp_header_and_name{}
	if s.m_header != nil {
		ret = append(ret, &Sdp_header_and_name{"m", s.m_header})
	}
	if s.i_header != nil {
		ret = append(ret, &Sdp_header_and_name{"i", s.i_header})
	}
	if s.c_header != nil {
		ret = append(ret, &Sdp_header_and_name{"c", s.c_header})
	}
	if s.b_header != nil {
		ret = append(ret, &Sdp_header_and_name{"b", s.b_header})
	}
	if s.k_header != nil {
		ret = append(ret, &Sdp_header_and_name{"k", s.k_header})
	}
	return ret
}

func (s *SdpMediaDescription) String() string {
	s := ""
	for _, it := range s.all_headers() {
		s += it.Name + "=" + it.Header.String() + "\r\n"
	}
	for _, header := range s.a_headers {
		s += "a=" + header + "\r\n"
	}
	return s
}

func (s *SdpMediaDescription) LocalStr(hostPort *sippy_net.HostPort, noC bool) string {
	s := ""
	for _, it := range s.all_headers() {
		if noC && it.Name == "c" {
			continue
		}
		s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
	}
	for _, header := range s.a_headers {
		s += "a=" + header + "\r\n"
	}
	return s
}

func (s *SdpMediaDescription) AddHeader(name, header string) {
	switch name {
	case "a":
		s.a_headers = append(s.a_headers, header)
	case "m":
		s.m_header = ParseSdpMedia(header)
	case "i":
		s.i_header = ParseSdpGeneric(header)
	case "c":
		s.c_header = ParseSdpConnecton(header)
	case "b":
		s.b_header = ParseSdpGeneric(header)
	case "k":
		s.k_header = ParseSdpGeneric(header)
	}
}

func (s *SdpMediaDescription) SetCHeaderAddr(addr string) {
	s.c_header.SetAddr(addr)
}

func (s *SdpMediaDescription) GetMHeader() *SdpMedia {
	if s.m_header == nil {
		return nil
	}
	return s.m_header
}

func (s *SdpMediaDescription) GetCHeader() *SdpConnecton {
	if s.c_header == nil {
		return nil
	}
	return s.c_header
}

func (s *SdpMediaDescription) SetCHeader(c_header *SdpConnecton) {
	s.c_header = c_header
}

func (s *SdpMediaDescription) HasAHeader(headers []string) bool {
	for _, hdr := range s.a_headers {
		for _, match := range headers {
			if hdr == match {
				return true
			}
		}
	}
	return false
}

func (s *SdpMediaDescription) RemoveAHeader(hdr string) {
	new_a_hdrs := []string{}
	for _, h := range s.a_headers {
		if strings.HasPrefix(h, hdr) {
			continue
		}
		new_a_hdrs = append(new_a_hdrs, h)
	}
	s.a_headers = new_a_hdrs
}

func (s *SdpMediaDescription) SetFormats(formats []string) {
	if s.m_header != nil {
		s.m_header.SetFormats(formats)
		s.optimize_a()
	}
}

func (s *SdpMediaDescription) optimize_a() {
	new_a_headers := []string{}
	for _, ah := range s.a_headers {
		pt := ""
		if strings.HasPrefix(ah, "rtpmap:") {
			pt = strings.Split(ah[7:], " ")[0]
		} else if strings.HasPrefix(ah, "fmtp:") {
			pt = strings.Split(ah[5:], " ")[0]
		}
		if pt != "" && !s.m_header.HasFormat(pt) {
			continue
		}
		new_a_headers = append(new_a_headers, ah)
	}
	s.a_headers = new_a_headers
}

func (s *SdpMediaDescription) GetAHeaders() []string {
	return s.a_headers
}

func (s *SdpMediaDescription) SetAHeaders(a_headers []string) {
	s.a_headers = a_headers
}

func (s *SdpMediaDescription) SanityCheck() error {
	switch {
	case s.m_header == nil:
		return fmt.Errorf("Mandatory \"m=\" SDP header is missing")
	case s.c_header == nil:
		return fmt.Errorf("Mandatory \"c=\" SDP header is missing")
	}
	return nil
}

func (s *SdpMediaDescription) IsOnHold() bool {
	if s.c_header.atype == "IP4" && s.c_header.addr == "0.0.0.0" {
		return true
	}
	if s.c_header.atype == "IP6" && s.c_header.addr == "::" {
		return true
	}
	for _, aname := range s.a_headers {
		if aname == "sendonly" || aname == "inactive" {
			return true
		}
	}
	return false
}
