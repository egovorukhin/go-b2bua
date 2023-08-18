package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipH323ConfId struct {
	normalName
	body string
}

var sipH323ConfIdName normalName = newNormalName("h323-conf-id")

func CreateSipH323ConfId(body string) []SipHeader {
	return []SipHeader{
		&SipH323ConfId{
			normalName: sipH323ConfIdName,
			body:       body,
		},
	}
}

func (s *SipH323ConfId) GetCopy() *SipH323ConfId {
	tmp := *s
	return &tmp
}

func (s *SipH323ConfId) StringBody() string {
	return s.body
}

func (s *SipH323ConfId) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipH323ConfId) String() string {
	return s.Name() + ": " + s.body
}

func (s *SipH323ConfId) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipH323ConfId) AsCiscoGUID() *SipCiscoGUID {
	return &SipCiscoGUID{
		normalName: sipCiscoGuidName,
		body:       s.body,
	}
}
