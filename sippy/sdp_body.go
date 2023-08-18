package sippy

import (
	"fmt"
	"strings"

	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/sdp"
	"github.com/sippy/go-b2bua/sippy/types"
)

type sdpBody struct {
	sections  []*sippy_sdp.SdpMediaDescription
	v_header  *sippy_sdp.SdpGeneric
	o_header  *sippy_sdp.SdpOrigin
	s_header  *sippy_sdp.SdpGeneric
	i_header  *sippy_sdp.SdpGeneric
	u_header  *sippy_sdp.SdpGeneric
	e_header  *sippy_sdp.SdpGeneric
	p_header  *sippy_sdp.SdpGeneric
	b_header  *sippy_sdp.SdpGeneric
	t_header  *sippy_sdp.SdpGeneric
	r_header  *sippy_sdp.SdpGeneric
	z_header  *sippy_sdp.SdpGeneric
	k_header  *sippy_sdp.SdpGeneric
	a_headers []string
	c_header  *sippy_sdp.SdpConnecton
}

func ParseSdpBody(body string) (*sdpBody, error) {
	var err error
	s := &sdpBody{
		a_headers: make([]string, 0),
		sections:  make([]*sippy_sdp.SdpMediaDescription, 0),
	}
	current_snum := 0
	var c_header *sippy_sdp.SdpConnecton
	for _, line := range strings.FieldsFunc(strings.TrimSpace(body), func(c rune) bool { return c == '\n' || c == '\r' }) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		arr := strings.SplitN(line, "=", 2)
		if len(arr) != 2 {
			continue
		}
		name, v := strings.ToLower(arr[0]), arr[1]
		if name == "m" {
			current_snum += 1
			s.sections = append(s.sections, sippy_sdp.NewSdpMediaDescription())
		}
		if current_snum == 0 {
			if name == "c" {
				c_header = sippy_sdp.ParseSdpConnecton(v)
			} else if name == "a" {
				s.a_headers = append(s.a_headers, v)
			} else {
				switch name {
				case "v":
					s.v_header = sippy_sdp.ParseSdpGeneric(v)
				case "o":
					s.o_header, err = sippy_sdp.ParseSdpOrigin(v)
					if err != nil {
						return nil, err
					}
				case "s":
					s.s_header = sippy_sdp.ParseSdpGeneric(v)
				case "i":
					s.i_header = sippy_sdp.ParseSdpGeneric(v)
				case "u":
					s.u_header = sippy_sdp.ParseSdpGeneric(v)
				case "e":
					s.e_header = sippy_sdp.ParseSdpGeneric(v)
				case "p":
					s.p_header = sippy_sdp.ParseSdpGeneric(v)
				case "b":
					s.b_header = sippy_sdp.ParseSdpGeneric(v)
				case "t":
					s.t_header = sippy_sdp.ParseSdpGeneric(v)
				case "r":
					s.r_header = sippy_sdp.ParseSdpGeneric(v)
				case "z":
					s.z_header = sippy_sdp.ParseSdpGeneric(v)
				case "k":
					s.k_header = sippy_sdp.ParseSdpGeneric(v)
				}
			}
		} else {
			s.sections[len(s.sections)-1].AddHeader(name, v)
		}
	}
	if c_header != nil {
		for _, section := range s.sections {
			if section.GetCHeader() == nil {
				section.SetCHeader(c_header)
			}
		}
		if len(s.sections) == 0 {
			s.c_header = c_header
		}
	}
	// Do some sanity checking, RFC4566
	switch {
	case s.v_header == nil:
		return nil, fmt.Errorf("Mandatory \"v=\" SDP header is missing")
	case s.o_header == nil:
		return nil, fmt.Errorf("Mandatory \"o=\" SDP header is missing")
	case s.s_header == nil:
		return nil, fmt.Errorf("Mandatory \"s=\" SDP header is missing")
	case s.t_header == nil:
		return nil, fmt.Errorf("Mandatory \"t=\" SDP header is missing")
	}
	for _, sect := range s.sections {
		if err := sect.SanityCheck(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *sdpBody) first_half() []*sippy_sdp.Sdp_header_and_name {
	ret := []*sippy_sdp.Sdp_header_and_name{}
	if s.v_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"v", s.v_header})
	}
	if s.o_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"o", s.o_header})
	}
	if s.s_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"s", s.s_header})
	}
	if s.i_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"i", s.i_header})
	}
	if s.u_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"u", s.u_header})
	}
	if s.e_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"e", s.e_header})
	}
	if s.p_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"p", s.p_header})
	}
	return ret
}

func (s *sdpBody) second_half() []*sippy_sdp.Sdp_header_and_name {
	ret := []*sippy_sdp.Sdp_header_and_name{}
	if s.b_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"b", s.b_header})
	}
	if s.t_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"t", s.t_header})
	}
	if s.r_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"r", s.r_header})
	}
	if s.z_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"z", s.z_header})
	}
	if s.k_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"k", s.k_header})
	}
	return ret
}

func (s *sdpBody) all_headers() []*sippy_sdp.Sdp_header_and_name {
	ret := s.first_half()
	if s.c_header != nil {
		ret = append(ret, &sippy_sdp.Sdp_header_and_name{"c", s.c_header})
	}
	return append(ret, s.second_half()...)
}

func (s *sdpBody) String() string {
	s := ""
	if len(s.sections) == 1 && s.sections[0].GetCHeader() != nil {
		for _, it := range s.first_half() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
		s += "c=" + s.sections[0].GetCHeader().String() + "\r\n"
		for _, it := range s.second_half() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
		for _, header := range s.a_headers {
			s += "a=" + header + "\r\n"
		}
		s += s.sections[0].LocalStr(nil, true /* noC */)
		return s
	}
	// Special code to optimize for the cases when there are many media streams pointing to
	// the same IP. Only include c= header into the top section of the SDP and remove it from
	// the streams that match.
	optimize_c_headers := false
	sections_0_str := ""
	if len(s.sections) > 1 && s.c_header == nil && s.sections[0].GetCHeader() != nil &&
		s.sections[0].GetCHeader().String() == s.sections[1].GetCHeader().String() {
		// Special code to optimize for the cases when there are many media streams pointing to
		// the same IP. Only include c= header into the top section of the SDP and remove it from
		// the streams that match.
		optimize_c_headers = true
		sections_0_str = s.sections[0].GetCHeader().String()
	}
	if optimize_c_headers {
		for _, it := range s.first_half() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
		s += "c=" + sections_0_str + "\r\n"
		for _, it := range s.second_half() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
	} else {
		for _, it := range s.all_headers() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
	}
	for _, header := range s.a_headers {
		s += "a=" + header + "\r\n"
	}
	for _, section := range s.sections {
		if optimize_c_headers && section.GetCHeader() != nil && section.GetCHeader().String() == sections_0_str {
			s += section.LocalStr(nil, true /* noC */)
		} else {
			s += section.String()
		}
	}
	return s
}

func (s *sdpBody) LocalStr(hostPort *sippy_net.HostPort) string {
	s := ""
	if len(s.sections) == 1 && s.sections[0].GetCHeader() != nil {
		for _, it := range s.first_half() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
		s += "c=" + s.sections[0].GetCHeader().String() + "\r\n"
		for _, it := range s.second_half() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
		for _, header := range s.a_headers {
			s += "a=" + header + "\r\n"
		}
		s += s.sections[0].LocalStr(hostPort, true /* noC */)
		return s
	}
	// Special code to optimize for the cases when there are many media streams pointing to
	// the same IP. Only include c= header into the top section of the SDP and remove it from
	// the streams that match.
	optimize_c_headers := false
	sections_0_str := ""
	if len(s.sections) > 1 && s.c_header == nil && s.sections[0].GetCHeader() != nil &&
		s.sections[0].GetCHeader().String() == s.sections[1].GetCHeader().String() {
		// Special code to optimize for the cases when there are many media streams pointing to
		// the same IP. Only include c= header into the top section of the SDP and remove it from
		// the streams that match.
		optimize_c_headers = true
		sections_0_str = s.sections[0].GetCHeader().String()
	}
	if optimize_c_headers {
		for _, it := range s.first_half() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
		s += "c=" + sections_0_str + "\r\n"
		for _, it := range s.second_half() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
	} else {
		for _, it := range s.all_headers() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
	}
	for _, header := range s.a_headers {
		s += "a=" + header + "\r\n"
	}
	for _, section := range s.sections {
		if optimize_c_headers && section.GetCHeader() != nil &&
			section.GetCHeader().String() == sections_0_str {
			s += section.LocalStr(hostPort /*noC =*/, true)
		} else {
			s += section.LocalStr(hostPort /*noC =*/, false)
		}
	}
	return s
}

func (s *sdpBody) GetCopy() sippy_types.Sdp {
	sections := make([]*sippy_sdp.SdpMediaDescription, len(s.sections))
	for i, s := range s.sections {
		sections[i] = s.GetCopy()
	}
	a_headers := make([]string, len(s.a_headers))
	copy(a_headers, s.a_headers)
	return &sdpBody{
		sections:  sections,
		v_header:  s.v_header.GetCopy(),
		o_header:  s.o_header.GetCopy(),
		s_header:  s.s_header.GetCopy(),
		i_header:  s.i_header.GetCopy(),
		u_header:  s.u_header.GetCopy(),
		e_header:  s.e_header.GetCopy(),
		p_header:  s.p_header.GetCopy(),
		b_header:  s.b_header.GetCopy(),
		t_header:  s.t_header.GetCopy(),
		r_header:  s.r_header.GetCopy(),
		z_header:  s.z_header.GetCopy(),
		k_header:  s.k_header.GetCopy(),
		a_headers: a_headers,
		c_header:  s.c_header.GetCopy(),
	}
}

func (s *sdpBody) GetCHeader() *sippy_sdp.SdpConnecton {
	return s.c_header
}

func (s *sdpBody) SetCHeaderAddr(addr string) {
	for _, sect := range s.sections {
		sect.GetCHeader().SetAddr(addr)
	}
}

func (s *sdpBody) GetSections() []*sippy_sdp.SdpMediaDescription {
	return s.sections
}

func (s *sdpBody) SetSections(sections []*sippy_sdp.SdpMediaDescription) {
	s.sections = sections
}

func (s *sdpBody) RemoveSection(idx int) {
	if idx < 0 || idx >= len(s.sections) {
		return
	}
	s.sections = append(s.sections[:idx], s.sections[idx+1:]...)
}

func (s *sdpBody) GetOHeader() *sippy_sdp.SdpOrigin {
	return s.o_header
}

func (s *sdpBody) SetOHeader(o_header *sippy_sdp.SdpOrigin) {
	s.o_header = o_header
}

func (s *sdpBody) AppendAHeader(hdr string) {
	s.a_headers = append(s.a_headers, hdr)
}
