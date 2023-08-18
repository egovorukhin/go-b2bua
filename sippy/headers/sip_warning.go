package sippy_header

import (
	"errors"
	"os"
	"strings"

	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/utils"
)

type sipWarningBody struct {
	code  string
	agent string
	text  string
}

type SipWarning struct {
	normalName
	stringBody string
	body       *sipWarningBody
}

var sipWarningName = newNormalName("Warning")

func CreateSipWarning(body string) []SipHeader {
	return []SipHeader{
		&SipWarning{
			normalName: sipWarningName,
			stringBody: body,
		},
	}
}

func (s *SipWarning) parse() error {
	arr := sippy_utils.FieldsN(s.stringBody, 3)
	if len(arr) != 3 {
		return errors.New("Malformed Warning field")
	}
	s.body = &sipWarningBody{
		code:  arr[0],
		agent: arr[1],
		text:  strings.Trim(arr[2], "\""),
	}
	return nil
}

func NewSipWarning(text string) *SipWarning {
	return &SipWarning{
		normalName: sipWarningName,
		body:       newSipWarningBody(text),
	}
}

func newSipWarningBody(text string) *sipWarningBody {
	text = strings.Replace(text, "\"", "'", -1)
	s := &sipWarningBody{
		code:  "399",
		agent: "unknown",
		text:  text,
	}
	hostname, err := os.Hostname()
	if err == nil {
		s.agent = hostname
	}
	return s
}

func (s *SipWarning) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.String()
}

func (s *SipWarning) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipWarning) StringBody() string {
	if s.body != nil {
		return s.body.String()
	}
	return s.stringBody
}

func (s *sipWarningBody) String() string {
	return s.code + " " + s.agent + " \"" + s.text + "\""
}

func (s *SipWarning) GetCopy() *SipWarning {
	tmp := *s
	return &tmp
}

func (s *SipWarning) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
