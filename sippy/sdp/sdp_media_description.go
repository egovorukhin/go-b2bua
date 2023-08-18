package sippy_sdp

import (
	"fmt"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SdpMediaDescription struct {
	m_header  *SdpMedia
	i_header  *SdpGeneric
	c_header  *SdpConnecton
	b_header  *SdpGeneric
	k_header  *SdpGeneric
	a_headers []string
}

func (d *SdpMediaDescription) GetCopy() *SdpMediaDescription {
	a_headers := make([]string, len(d.a_headers))
	copy(a_headers, d.a_headers)
	return &SdpMediaDescription{
		m_header:  d.m_header.GetCopy(),
		i_header:  d.i_header.GetCopy(),
		c_header:  d.c_header.GetCopy(),
		b_header:  d.b_header.GetCopy(),
		k_header:  d.k_header.GetCopy(),
		a_headers: a_headers,
	}
}

func NewSdpMediaDescription() *SdpMediaDescription {
	return &SdpMediaDescription{
		a_headers: make([]string, 0),
	}
}

func (d *SdpMediaDescription) all_headers() []*SdpHeaderAndName {
	ret := []*SdpHeaderAndName{}
	if d.m_header != nil {
		ret = append(ret, &SdpHeaderAndName{"m", d.m_header})
	}
	if d.i_header != nil {
		ret = append(ret, &SdpHeaderAndName{"i", d.i_header})
	}
	if d.c_header != nil {
		ret = append(ret, &SdpHeaderAndName{"c", d.c_header})
	}
	if d.b_header != nil {
		ret = append(ret, &SdpHeaderAndName{"b", d.b_header})
	}
	if d.k_header != nil {
		ret = append(ret, &SdpHeaderAndName{"k", d.k_header})
	}
	return ret
}

func (d *SdpMediaDescription) String() (s string) {
	for _, it := range d.all_headers() {
		s += it.Name + "=" + it.Header.String() + "\r\n"
	}
	for _, header := range d.a_headers {
		s += "a=" + header + "\r\n"
	}
	return s
}

func (d *SdpMediaDescription) LocalStr(hostPort *sippy_net.HostPort, noC bool) (s string) {
	for _, it := range d.all_headers() {
		if noC && it.Name == "c" {
			continue
		}
		s += it.Name + "=" + it.Header.LocalStr(hostPort) + "\r\n"
	}
	for _, header := range d.a_headers {
		s += "a=" + header + "\r\n"
	}
	return
}

func (d *SdpMediaDescription) AddHeader(name, header string) {
	switch name {
	case "a":
		d.a_headers = append(d.a_headers, header)
	case "m":
		d.m_header = ParseSdpMedia(header)
	case "i":
		d.i_header = ParseSdpGeneric(header)
	case "c":
		d.c_header = ParseSdpConnecton(header)
	case "b":
		d.b_header = ParseSdpGeneric(header)
	case "k":
		d.k_header = ParseSdpGeneric(header)
	}
}

func (d *SdpMediaDescription) SetCHeaderAddr(addr string) {
	d.c_header.SetAddr(addr)
}

func (d *SdpMediaDescription) GetMHeader() *SdpMedia {
	if d.m_header == nil {
		return nil
	}
	return d.m_header
}

func (d *SdpMediaDescription) GetCHeader() *SdpConnecton {
	if d.c_header == nil {
		return nil
	}
	return d.c_header
}

func (d *SdpMediaDescription) SetCHeader(c_header *SdpConnecton) {
	d.c_header = c_header
}

func (d *SdpMediaDescription) HasAHeader(headers []string) bool {
	for _, hdr := range d.a_headers {
		for _, match := range headers {
			if hdr == match {
				return true
			}
		}
	}
	return false
}

func (d *SdpMediaDescription) RemoveAHeader(hdr string) {
	new_a_hdrs := []string{}
	for _, h := range d.a_headers {
		if strings.HasPrefix(h, hdr) {
			continue
		}
		new_a_hdrs = append(new_a_hdrs, h)
	}
	d.a_headers = new_a_hdrs
}

func (d *SdpMediaDescription) SetFormats(formats []string) {
	if d.m_header != nil {
		d.m_header.SetFormats(formats)
		d.optimize_a()
	}
}

func (d *SdpMediaDescription) optimize_a() {
	new_a_headers := []string{}
	for _, ah := range d.a_headers {
		pt := ""
		if strings.HasPrefix(ah, "rtpmap:") {
			pt = strings.Split(ah[7:], " ")[0]
		} else if strings.HasPrefix(ah, "fmtp:") {
			pt = strings.Split(ah[5:], " ")[0]
		}
		if pt != "" && !d.m_header.HasFormat(pt) {
			continue
		}
		new_a_headers = append(new_a_headers, ah)
	}
	d.a_headers = new_a_headers
}

func (d *SdpMediaDescription) GetAHeaders() []string {
	return d.a_headers
}

func (d *SdpMediaDescription) SetAHeaders(a_headers []string) {
	d.a_headers = a_headers
}

func (d *SdpMediaDescription) SanityCheck() error {
	switch {
	case d.m_header == nil:
		return fmt.Errorf("Mandatory \"m=\" SDP header is missing")
	case d.c_header == nil:
		return fmt.Errorf("Mandatory \"c=\" SDP header is missing")
	}
	return nil
}

func (d *SdpMediaDescription) IsOnHold() bool {
	if d.c_header.atype == "IP4" && d.c_header.addr == "0.0.0.0" {
		return true
	}
	if d.c_header.atype == "IP6" && d.c_header.addr == "::" {
		return true
	}
	for _, aname := range d.a_headers {
		if aname == "sendonly" || aname == "inactive" {
			return true
		}
	}
	return false
}
