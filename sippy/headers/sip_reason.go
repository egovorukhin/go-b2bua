package sippy_header

import (
	"errors"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type sipReasonBody struct {
	reason   string
	cause    string
	protocol string
}

func newSipReasonBody(protocol, cause, reason string) *sipReasonBody {
	return &sipReasonBody{
		reason:   reason,
		protocol: protocol,
		cause:    cause,
	}
}

func (s *sipReasonBody) String() string {
	var rval string
	if s.reason == "" {
		rval = s.protocol + "; cause=" + s.cause
	} else {
		rval = s.protocol + "; cause=" + s.cause + "; text=\"" + s.reason + "\""
	}
	return rval
}

type SipReason struct {
	normalName
	stringBody string
	body       *sipReasonBody
}

var sipReasonName = newNormalName("Reason")

func CreateSipReason(body string) []SipHeader {
	return []SipHeader{
		&SipReason{
			normalName: sipReasonName,
			stringBody: body,
		},
	}
}

func (s *SipReason) parse() error {
	arr := strings.SplitN(s.stringBody, ";", 2)
	if len(arr) != 2 {
		return errors.New("Error parsing Reason: (1)")
	}
	protocol, reasonParams := arr[0], arr[1]
	body := &sipReasonBody{
		protocol: strings.TrimSpace(protocol),
	}
	for _, reasonParam := range strings.Split(reasonParams, ";") {
		arr = strings.SplitN(reasonParam, "=", 2)
		if len(arr) != 2 {
			return errors.New("Error parsing Reason: (2)")
		}
		rpName, rpValue := strings.TrimSpace(arr[0]), strings.TrimSpace(arr[1])
		switch rpName {
		case "cause":
			body.cause = rpValue
		case "text":
			body.reason = strings.Trim(rpValue, "\"")
		}
	}
	s.body = body
	return nil
}

func (s *SipReason) StringBody() string {
	if s.body != nil {
		return s.body.String()
	}
	return s.stringBody
}

func (s *SipReason) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipReason) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.StringBody()
}

func NewSipReason(protocol, cause, reason string) *SipReason {
	return &SipReason{
		normalName: sipReasonName,
		body:       newSipReasonBody(protocol, cause, reason),
	}
}

func (s *SipReason) GetCopy() *SipReason {
	tmp := *s
	return &tmp
}

func (s *SipReason) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
