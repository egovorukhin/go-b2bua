package sippy_header

import (
	"errors"
	"strconv"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

const (
	Rfc3261UserUnreserved = "&=+$,;?/#"
	Rfc3261Unreserved     = "-_.!~*'()"
)

var userEnc *sippy_utils.UrlEncode
var passwEnc *sippy_utils.UrlEncode
var hnvEnc *sippy_utils.UrlEncode

func init() {
	userEnc = sippy_utils.NewUrlEncode([]byte(Rfc3261UserUnreserved + Rfc3261Unreserved))
	passwEnc = sippy_utils.NewUrlEncode([]byte(Rfc3261Unreserved + "&=+$,"))
	//param_enc = sippy_utils.NewUrlEncode([]byte(RFC3261_UNRESERVED + "[]/:&+$"))
	hnvEnc = sippy_utils.NewUrlEncode([]byte(Rfc3261Unreserved + "[]/?:+$"))
}

type SipURL struct {
	Username   string
	password   string
	ttl        int
	Host       *sippy_net.MyAddress
	Port       *sippy_net.MyPort
	usertype   string
	transport  string
	mAddr      string
	method     string
	tag        string
	Lr         bool
	other      []string
	userParams []string
	headers    map[string]string
	scheme     string
}

func NewSipURL(username string, host *sippy_net.MyAddress, port *sippy_net.MyPort, lr bool /* false */) *SipURL {
	s := &SipURL{
		scheme:     "sip",
		other:      make([]string, 0),
		userParams: make([]string, 0),
		Username:   username,
		headers:    make(map[string]string),
		Lr:         lr,
		ttl:        -1,
		Host:       host,
		Port:       port,
	}
	return s
}

func ParseURL(url string, relaxedParser bool) (*SipURL, error) {
	parts := strings.SplitN(url, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("scheme is not present")
	}
	s := NewSipURL("", nil, nil, false)
	s.scheme = strings.ToLower(parts[0])
	switch s.scheme {
	case "sip":
		fallthrough
	case "sips":
		return s, s.parseSipURL(parts[1], relaxedParser)
	case "tel":
		s.parseTelUrl(parts[1])
		return s, nil
	}
	return nil, errors.New("unsupported scheme: " + s.scheme + ":")
}

func ParseSipURL(url string, relaxedParser bool, config sippy_conf.Config) (*SipURL, error) {
	s, err := ParseURL(url, relaxedParser)
	if err != nil {
		return nil, err
	}
	if s.scheme == "tel" {
		if config.AutoConvertTelUrl() {
			s.convertTelUrl(relaxedParser, config)
		} else {
			return nil, errors.New("unsupported scheme: " + s.scheme + ":")
		}
	}
	return s, nil
}

func (s *SipURL) convertTelUrl(relaxedParser bool, config sippy_conf.Config) {
	s.scheme = "sip"
	if relaxedParser {
		s.Host = sippy_net.NewMyAddress("")
	} else {
		s.Host = config.GetMyAddress()
		s.Port = config.DefaultPort()
	}
}

func (s *SipURL) parseTelUrl(url string) {
	parts := strings.Split(url, ";")
	s.Username, _ = userEnc.Unescape(parts[0])
	if len(parts) > 1 {
		// parse userParams
		for _, part := range parts[1:] {
			// The RFC-3261 suggests the user parameter keys should
			// be converted to lower case.
			arr := strings.SplitN(part, "=", 2)
			if len(arr) == 2 {
				s.userParams = append(s.userParams, strings.ToLower(arr[0])+"="+arr[1])
			} else {
				s.userParams = append(s.userParams, part)
			}
		}
	}
}

func (s *SipURL) parseSipURL(url string, relaxedparser bool) error {
	var params []string
	var hostPort string

	ear := strings.Index(url, "@") + 1
	parts := strings.Split(url[ear:], ";")
	userDomain := url[0:ear] + parts[0]
	if len(parts) > 1 {
		params = parts[1:]
	} else {
		params = make([]string, 0)
	}
	if len(params) == 0 && strings.Contains(userDomain[ear:], "?") {
		arr := strings.SplitN(userDomain[ear:], "?", 2)
		userDomainSuff := arr[0]
		headers := arr[1]
		userDomain = userDomain[:ear] + userDomainSuff
		for _, header := range strings.Split(headers, "&") {
			arr = strings.SplitN(header, "=", 2)
			if len(arr) == 2 {
				s.headers[strings.ToLower(arr[0])], _ = hnvEnc.Unescape(arr[1])
			}
		}
	}
	if ear > 0 {
		userPass := userDomain[:ear-1]
		hostPort = userDomain[ear:]
		upParts := strings.SplitN(userPass, ":", 2)
		if len(upParts) > 1 {
			s.password, _ = passwEnc.Unescape(upParts[1])
		}
		uParts := strings.Split(upParts[0], ";")
		if len(uParts) > 1 {
			s.userParams = uParts[1:]
		}
		s.Username, _ = userEnc.Unescape(uParts[0])
	} else {
		hostPort = userDomain
	}
	var parsePort *string = nil
	if relaxedparser && len(hostPort) == 0 {
		s.Host = sippy_net.NewMyAddress("")
	} else if hostPort[0] == '[' {
		// IPv6 host
		hpParts := strings.SplitN(hostPort, "]", 2)
		s.Host = sippy_net.NewMyAddress(hpParts[0] + "]")
		if len(hpParts[1]) > 0 {
			hpParts = strings.SplitN(hpParts[1], ":", 2)
			if len(hpParts) > 1 {
				parsePort = &hpParts[1]
			}
		}
	} else {
		// IPv4 host
		hpParts := strings.SplitN(hostPort, ":", 2)
		s.Host = sippy_net.NewMyAddress(hpParts[0])
		if len(hpParts) == 2 {
			parsePort = &hpParts[1]
		}
	}
	if parsePort != nil {
		port := strings.TrimSpace(*parsePort)
		if port == "" {
			// Bug on the other side, work around it
			//print 'WARNING: non-compliant URI detected, empty port number, ' \
			//  'assuming default: "%s"' % str(original_uri)
		} else {
			_, err := strconv.Atoi(port)
			if err != nil {
				if strings.Contains(port, ":") {
					// Can't parse port number, check why
					pParts := strings.SplitN(port, ":", 2)
					if pParts[0] == pParts[1] {
						// Bug on the other side, work around it
						//print 'WARNING: non-compliant URI detected, duplicate port number, ' \
						//  'taking "%s": %s' % (pparts[0], str(original_uri))
						if _, err = strconv.Atoi(pParts[0]); err != nil {
							return err
						}
						s.Port = sippy_net.NewMyPort(pParts[0])
					} else {
						return err
					}
				} else {
					return err
				}
			} else {
				s.Port = sippy_net.NewMyPort(port)
			}
		}
	}
	if len(params) > 0 {
		lastParam := params[len(params)-1]
		arr := strings.SplitN(lastParam, "?", 2)
		params[len(params)-1] = arr[0]
		s.SetParams(params)
		if len(arr) == 2 {
			s.headers = make(map[string]string)
			headers := arr[1]
			for _, header := range strings.Split(headers, "&") {
				if arr := strings.SplitN(header, "=", 2); len(arr) == 2 {
					s.headers[strings.ToLower(arr[0])], _ = hnvEnc.Unescape(arr[1])
				}
			}
		}
	}
	return nil
}

func (s *SipURL) SetParams(params []string) {
	s.usertype = ""
	s.transport = ""
	s.mAddr = ""
	s.method = ""
	s.tag = ""
	s.ttl = -1
	s.other = []string{}
	s.Lr = false

	for _, p := range params {
		nv := strings.SplitN(p, "=", 2)
		if len(nv) == 1 {
			if p == "lr" {
				s.Lr = true
			} else {
				s.other = append(s.other, p)
			}
			continue
		}
		name := nv[0]
		value := nv[1]
		switch name {
		case "user":
			s.usertype = value
		case "transport":
			s.transport = value
		case "ttl":
			if v, err := strconv.Atoi(value); err == nil {
				s.ttl = v
			}
		case "mAddr":
			s.mAddr = value
		case "method":
			s.method = value
		case "tag":
			s.tag = value
		case "lr":
			// RFC 3261 doesn't allow lr parameter to have a value,
			// but many stupid implementation do it anyway
			s.Lr = true
		default:
			s.other = append(s.other, p)
		}
	}
}

func (s *SipURL) String() string {
	return s.LocalStr(nil)
}

func (s *SipURL) LocalStr(hostPort *sippy_net.HostPort) string {
	l := s.scheme + ":"
	if s.Username != "" {
		username := userEnc.Escape(s.Username)
		l += username
		for _, v := range s.userParams {
			l += ";" + v
		}
		if s.password != "" {
			l += ":" + passwEnc.Escape(s.password)
		}
		l += "@"
	}
	if hostPort != nil && s.Host.IsSystemDefault() {
		l += hostPort.Host.String()
	} else {
		l += s.Host.String()
	}
	if s.Port != nil {
		if hostPort != nil && s.Port.IsSystemDefault() {
			l += ":" + hostPort.Port.String()
		} else {
			l += ":" + s.Port.String()
		}
	}
	for _, p := range s.GetParams() {
		l += ";" + p
	}
	if len(s.headers) > 0 {
		l += "?"
		arr := []string{}
		for k, v := range s.headers {
			arr = append(arr, strings.Title(k)+"="+hnvEnc.Escape(v))
		}
		l += strings.Join(arr, "&")
	}
	return l
}

func (s *SipURL) GetParams() []string {
	var ret []string
	if s.usertype != "" {
		ret = append(ret, "user="+s.usertype)
	}
	if s.transport != "" {
		ret = append(ret, "transport="+s.transport)
	}
	if s.mAddr != "" {
		ret = append(ret, "mAddr="+s.mAddr)
	}
	if s.method != "" {
		ret = append(ret, "method="+s.method)
	}
	if s.tag != "" {
		ret = append(ret, "tag="+s.tag)
	}
	if s.ttl != -1 {
		ret = append(ret, "ttl="+strconv.Itoa(s.ttl))
	}
	ret = append(ret, s.other...)
	if s.Lr {
		ret = append(ret, "lr")
	}
	return ret
}

func (s *SipURL) GetCopy() *SipURL {
	ret := *s
	return &ret
}

func (s *SipURL) GetAddr(config sippy_conf.Config) *sippy_net.HostPort {
	if s.Port != nil {
		return sippy_net.NewHostPort(s.Host.String(), s.Port.String())
	}
	return sippy_net.NewHostPort(s.Host.String(), config.DefaultPort().String())
}

func (s *SipURL) SetUserParams(userParams []string) {
	s.userParams = userParams
}

func (s *SipURL) GetUserParams() []string {
	return s.userParams
}
