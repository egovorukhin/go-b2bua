package sippy

import (
	"fmt"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/sdp"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type sdpBody struct {
	sections []*sippy_sdp.SdpMediaDescription
	vHeader  *sippy_sdp.SdpGeneric
	oHeader  *sippy_sdp.SdpOrigin
	sHeader  *sippy_sdp.SdpGeneric
	iHeader  *sippy_sdp.SdpGeneric
	uHeader  *sippy_sdp.SdpGeneric
	eHeader  *sippy_sdp.SdpGeneric
	pHeader  *sippy_sdp.SdpGeneric
	bHeader  *sippy_sdp.SdpGeneric
	tHeader  *sippy_sdp.SdpGeneric
	rHeader  *sippy_sdp.SdpGeneric
	zHeader  *sippy_sdp.SdpGeneric
	kHeader  *sippy_sdp.SdpGeneric
	aHeaders []string
	cHeader  *sippy_sdp.SdpConnecton
}

func ParseSdpBody(body string) (*sdpBody, error) {
	var err error
	s := &sdpBody{
		aHeaders: make([]string, 0),
		sections: make([]*sippy_sdp.SdpMediaDescription, 0),
	}
	currentSnum := 0
	var cHeader *sippy_sdp.SdpConnecton
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
			currentSnum += 1
			s.sections = append(s.sections, sippy_sdp.NewSdpMediaDescription())
		}
		if currentSnum == 0 {
			if name == "c" {
				cHeader = sippy_sdp.ParseSdpConnecton(v)
			} else if name == "a" {
				s.aHeaders = append(s.aHeaders, v)
			} else {
				switch name {
				case "v":
					s.vHeader = sippy_sdp.ParseSdpGeneric(v)
				case "o":
					s.oHeader, err = sippy_sdp.ParseSdpOrigin(v)
					if err != nil {
						return nil, err
					}
				case "s":
					s.sHeader = sippy_sdp.ParseSdpGeneric(v)
				case "i":
					s.iHeader = sippy_sdp.ParseSdpGeneric(v)
				case "u":
					s.uHeader = sippy_sdp.ParseSdpGeneric(v)
				case "e":
					s.eHeader = sippy_sdp.ParseSdpGeneric(v)
				case "p":
					s.pHeader = sippy_sdp.ParseSdpGeneric(v)
				case "b":
					s.bHeader = sippy_sdp.ParseSdpGeneric(v)
				case "t":
					s.tHeader = sippy_sdp.ParseSdpGeneric(v)
				case "r":
					s.rHeader = sippy_sdp.ParseSdpGeneric(v)
				case "z":
					s.zHeader = sippy_sdp.ParseSdpGeneric(v)
				case "k":
					s.kHeader = sippy_sdp.ParseSdpGeneric(v)
				}
			}
		} else {
			s.sections[len(s.sections)-1].AddHeader(name, v)
		}
	}
	if cHeader != nil {
		for _, section := range s.sections {
			if section.GetCHeader() == nil {
				section.SetCHeader(cHeader)
			}
		}
		if len(s.sections) == 0 {
			s.cHeader = cHeader
		}
	}
	// Do some sanity checking, RFC4566
	switch {
	case s.vHeader == nil:
		return nil, fmt.Errorf("Mandatory \"v=\" SDP header is missing")
	case s.oHeader == nil:
		return nil, fmt.Errorf("Mandatory \"o=\" SDP header is missing")
	case s.sHeader == nil:
		return nil, fmt.Errorf("Mandatory \"s=\" SDP header is missing")
	case s.tHeader == nil:
		return nil, fmt.Errorf("Mandatory \"t=\" SDP header is missing")
	}
	for _, sect := range s.sections {
		if err := sect.SanityCheck(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *sdpBody) firstHalf() []*sippy_sdp.SdpHeaderAndName {
	var ret []*sippy_sdp.SdpHeaderAndName
	if s.vHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"v", s.vHeader})
	}
	if s.oHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"o", s.oHeader})
	}
	if s.sHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"s", s.sHeader})
	}
	if s.iHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"i", s.iHeader})
	}
	if s.uHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"u", s.uHeader})
	}
	if s.eHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"e", s.eHeader})
	}
	if s.pHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"p", s.pHeader})
	}
	return ret
}

func (s *sdpBody) secondHalf() []*sippy_sdp.SdpHeaderAndName {
	var ret []*sippy_sdp.SdpHeaderAndName
	if s.bHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"b", s.bHeader})
	}
	if s.tHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"t", s.tHeader})
	}
	if s.rHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"r", s.rHeader})
	}
	if s.zHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"z", s.zHeader})
	}
	if s.kHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"k", s.kHeader})
	}
	return ret
}

func (s *sdpBody) allHeaders() []*sippy_sdp.SdpHeaderAndName {
	ret := s.firstHalf()
	if s.cHeader != nil {
		ret = append(ret, &sippy_sdp.SdpHeaderAndName{"c", s.cHeader})
	}
	return append(ret, s.secondHalf()...)
}

func (b *sdpBody) String() (s string) {
	if len(b.sections) == 1 && b.sections[0].GetCHeader() != nil {
		for _, it := range b.firstHalf() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
		s += "c=" + b.sections[0].GetCHeader().String() + "\r\n"
		for _, it := range b.secondHalf() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
		for _, header := range b.aHeaders {
			s += "a=" + header + "\r\n"
		}
		s += b.sections[0].LocalStr(nil, true /* noC */)
		return s
	}
	// Special code to optimize for the cases when there are many media streams pointing to
	// the same IP. Only include c= header into the top section of the SDP and remove it from
	// the streams that match.
	optimizeCHeaders := false
	sections0Str := ""
	if len(b.sections) > 1 && b.cHeader == nil && b.sections[0].GetCHeader() != nil &&
		b.sections[0].GetCHeader().String() == b.sections[1].GetCHeader().String() {
		// Special code to optimize for the cases when there are many media streams pointing to
		// the same IP. Only include c= header into the top section of the SDP and remove it from
		// the streams that match.
		optimizeCHeaders = true
		sections0Str = b.sections[0].GetCHeader().String()
	}
	if optimizeCHeaders {
		for _, it := range b.firstHalf() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
		s += "c=" + sections0Str + "\r\n"
		for _, it := range b.secondHalf() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
	} else {
		for _, it := range b.allHeaders() {
			s += it.Name + "=" + it.Header.String() + "\r\n"
		}
	}
	for _, header := range b.aHeaders {
		s += "a=" + header + "\r\n"
	}
	for _, section := range b.sections {
		if optimizeCHeaders && section.GetCHeader() != nil && section.GetCHeader().String() == sections0Str {
			s += section.LocalStr(nil, true /* noC */)
		} else {
			s += section.String()
		}
	}
	return s
}

func (b *sdpBody) LocalStr(hostPort *sippy_net.HostPort) (s string) {
	if len(b.sections) == 1 && b.sections[0].GetCHeader() != nil {
		for _, it := range b.firstHalf() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
		s += "c=" + b.sections[0].GetCHeader().String() + "\r\n"
		for _, it := range b.secondHalf() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
		for _, header := range b.aHeaders {
			s += "a=" + header + "\r\n"
		}
		s += b.sections[0].LocalStr(hostPort, true /* noC */)
		return s
	}
	// Special code to optimize for the cases when there are many media streams pointing to
	// the same IP. Only include c= header into the top section of the SDP and remove it from
	// the streams that match.
	optimizeCHeaders := false
	sections0Str := ""
	if len(b.sections) > 1 && b.cHeader == nil && b.sections[0].GetCHeader() != nil &&
		b.sections[0].GetCHeader().String() == b.sections[1].GetCHeader().String() {
		// Special code to optimize for the cases when there are many media streams pointing to
		// the same IP. Only include c= header into the top section of the SDP and remove it from
		// the streams that match.
		optimizeCHeaders = true
		sections0Str = b.sections[0].GetCHeader().String()
	}
	if optimizeCHeaders {
		for _, it := range b.firstHalf() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
		s += "c=" + sections0Str + "\r\n"
		for _, it := range b.secondHalf() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
	} else {
		for _, it := range b.allHeaders() {
			s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
		}
	}
	for _, header := range b.aHeaders {
		s += "a=" + header + "\r\n"
	}
	for _, section := range b.sections {
		if optimizeCHeaders && section.GetCHeader() != nil &&
			section.GetCHeader().String() == sections0Str {
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
	a_headers := make([]string, len(s.aHeaders))
	copy(a_headers, s.aHeaders)
	return &sdpBody{
		sections: sections,
		vHeader:  s.vHeader.GetCopy(),
		oHeader:  s.oHeader.GetCopy(),
		sHeader:  s.sHeader.GetCopy(),
		iHeader:  s.iHeader.GetCopy(),
		uHeader:  s.uHeader.GetCopy(),
		eHeader:  s.eHeader.GetCopy(),
		pHeader:  s.pHeader.GetCopy(),
		bHeader:  s.bHeader.GetCopy(),
		tHeader:  s.tHeader.GetCopy(),
		rHeader:  s.rHeader.GetCopy(),
		zHeader:  s.zHeader.GetCopy(),
		kHeader:  s.kHeader.GetCopy(),
		aHeaders: a_headers,
		cHeader:  s.cHeader.GetCopy(),
	}
}

func (s *sdpBody) GetCHeader() *sippy_sdp.SdpConnecton {
	return s.cHeader
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
	return s.oHeader
}

func (s *sdpBody) SetOHeader(o_header *sippy_sdp.SdpOrigin) {
	s.oHeader = o_header
}

func (s *sdpBody) AppendAHeader(hdr string) {
	s.aHeaders = append(s.aHeaders, hdr)
}
