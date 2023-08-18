package sippy_header

import (
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type sipReplacesBody struct {
	callId      string
	fromTag     string
	toTag       string
	earlyOnly   bool
	otherParams string
}

type SipReplaces struct {
	normalName
	stringBody string
	body       *sipReplacesBody
}

var sipReplacesName normalName = newNormalName("Replaces")

func CreateSipReplaces(body string) []SipHeader {
	return []SipHeader{
		&SipReplaces{
			normalName: sipReplacesName,
		},
	}
}

func (s *SipReplaces) parse() {
	params := strings.Split(s.stringBody, ";")
	body := &sipReplacesBody{
		callId: params[0],
	}
	for _, param := range params[1:] {
		kv := strings.SplitN(param, "=", 2)
		switch kv[0] {
		case "from-tag":
			if len(kv) == 2 {
				body.fromTag = kv[1]
			}
		case "to-tag":
			if len(kv) == 2 {
				body.toTag = kv[1]
			}
		case "early-only":
			body.earlyOnly = true
		default:
			body.otherParams += ";" + param
		}
	}
	s.body = body
}

func (s *SipReplaces) StringBody() string {
	if s.body != nil {
		return s.body.String()
	}
	return s.stringBody
}

func (s *sipReplacesBody) String() string {
	res := s.callId + ";from-tag=" + s.fromTag + ";to-tag=" + s.toTag
	if s.earlyOnly {
		res += ";early-only"
	}
	return res + s.otherParams
}

func (s *SipReplaces) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipReplaces) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipReplaces) GetCopy() *SipReplaces {
	tmp := *s
	return &tmp
}

func (s *SipReplaces) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
