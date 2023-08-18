package sippy_header

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type SipAddress struct {
	params   map[string]*string
	url      *SipURL
	hadbrace bool
	name     string
	q        float64
}

func ParseSipAddress(address string, relaxedparser bool, config sippy_conf.Config) (*SipAddress, error) {
	var err error
	var arr []string

	// simple 'sip:foo' case
	sipAddress := &SipAddress{
		params:   make(map[string]*string),
		hadbrace: true,
		q:        1.0,
	}

	if strings.HasPrefix(strings.ToLower(address), "sip:") && strings.Index(address, "<") == -1 {
		parts := strings.SplitN(address, ";", 2)
		sipAddress.url, err = ParseSipURL(parts[0], relaxedparser, config)
		if err != nil {
			return nil, err
		}
		if len(parts) == 2 {
			if err = sipAddress.parseParamString(parts[1]); err != nil {
				return nil, err
			}
		}
		sipAddress.hadbrace = false
		return sipAddress, nil
	}
	var url *string = nil
	if address[0] == '"' {
		equote := strings.Index(address[1:], "\"") + 1
		if equote != 0 {
			sbrace := strings.Index(address[equote:], "<")
			if sbrace != -1 {
				sipAddress.hadbrace = true
				sipAddress.name = strings.TrimSpace(address[1:equote])
				tmp := address[equote+sbrace+1:]
				url = &tmp
			}
		}
	}
	if url == nil {
		arr = strings.SplitN(address, "<", 2)
		if len(arr) != 2 {
			return nil, errors.New("ParseSipAddress #1")
		}
		sipAddress.name = strings.TrimSpace(arr[0])
		url = &arr[1]
		if len(sipAddress.name) > 0 && sipAddress.name[0] == '"' {
			sipAddress.name = sipAddress.name[1:]
		}
		if len(sipAddress.name) > 0 && sipAddress.name[len(sipAddress.name)-1] == '"' {
			sipAddress.name = sipAddress.name[:len(sipAddress.name)-1]
		}
	}
	arr = strings.SplitN(*url, ">", 2)
	if len(arr) != 2 {
		return nil, errors.New("ParseSipAddress #2")
	}
	paramstring := arr[1]
	if sipAddress.url, err = ParseSipURL(arr[0], relaxedparser, config); err != nil {
		return nil, err
	}
	paramstring = strings.TrimSpace(paramstring)
	if err = sipAddress.parseParamString(paramstring); err != nil {
		return nil, err
	}
	return sipAddress, nil
}

func (a *SipAddress) parseParamString(s string) error {
	for _, l := range strings.Split(s, ";") {
		var v *string

		if l == "" {
			continue
		}
		arr := strings.SplitN(l, "=", 2)
		k := arr[0]
		if len(arr) == 2 {
			tmp := arr[1]
			v = &tmp
		} else {
			v = nil
		}
		if _, ok := a.params[k]; ok {
			return errors.New("Duplicate parameter in SIP address: " + k)
		}
		if k == "q" {
			if v != nil {
				// ignore absense or possible errors in the q= value
				if q, err := strconv.ParseFloat(*v, 64); err == nil {
					a.q = q
				}
			}
		} else {
			a.params[k] = v
		}
	}
	return nil
}

func (a *SipAddress) String() string {
	return a.LocalStr(nil)
}

func (a *SipAddress) LocalStr(hostPort *sippy_net.HostPort) string {
	var od, cd, s string
	if a.hadbrace {
		od = "<"
		cd = ">"
	}
	if len(a.name) > 0 {
		needsQuote := false
		for _, r := range a.name {
			if unicode.IsLetter(r) || unicode.IsNumber(r) || strings.ContainsRune("-.!%*_+`'~", r) {
				continue
			}
			needsQuote = true
			break
		}
		if needsQuote {
			s += "\"" + a.name + "\" "
		} else {
			s += a.name + " "
		}
		od = "<"
		cd = ">"
	}
	s += od + a.url.LocalStr(hostPort) + cd
	for k, v := range a.params {
		if v == nil {
			s += ";" + k
		} else {
			s += ";" + k + "=" + *v
		}
	}
	if a.q != 1.0 {
		s += ";q=" + strconv.FormatFloat(a.q, 'g', -1, 64)
	}
	return s
}

func NewSipAddress(name string, url *SipURL) *SipAddress {
	return &SipAddress{
		name:     name,
		url:      url,
		hadbrace: true,
		params:   make(map[string]*string),
		q:        1.0,
	}
}

func (a *SipAddress) GetCopy() *SipAddress {
	ret := *a
	ret.params = make(map[string]*string)
	for k, v := range a.params {
		if v == nil {
			ret.params[k] = nil
		} else {
			s := *v
			ret.params[k] = &s
		}
	}
	ret.url = a.url.GetCopy()
	return &ret
}

func (a *SipAddress) GetParam(name string) string {
	ret, ok := a.params[name]
	if !ok || ret == nil {
		return ""
	}
	return *ret
}

func (a *SipAddress) SetParam(name, value string) {
	a.params[name] = &value
}

func (a *SipAddress) GetParams() map[string]*string {
	return a.params
}

func (a *SipAddress) SetParams(params map[string]*string) {
	a.params = params
}

func (a *SipAddress) GetName() string {
	return a.name
}

func (a *SipAddress) SetName(name string) {
	a.name = name
}

func (a *SipAddress) GetUrl() *SipURL {
	return a.url
}

func (a *SipAddress) GetTag() string {
	return a.GetParam("tag")
}

func (a *SipAddress) SetTag(tag string) {
	a.SetParam("tag", tag)
}

func (a *SipAddress) GenTag() {
	a.SetParam("tag", sippy_utils.GenTag())
}

func (a *SipAddress) GetQ() float64 {
	return a.q
}
