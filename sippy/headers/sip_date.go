package sippy_header

import (
	"github.com/sippy/go-b2bua/sippy/net"
	"time"
)

var sipDateName normalName = newNormalName("Date")

type SipDate struct {
	normalName
	strBody string
	ts      time.Time
	parsed  bool
}

func CreateSipDate(body string) []SipHeader {
	return []SipHeader{
		&SipDate{
			normalName: sipDateName,
			strBody:    body,
			parsed:     false,
		},
	}
}

func NewSipDate(ts time.Time) *SipDate {
	return &SipDate{
		normalName: sipDateName,
		strBody:    ts.In(time.FixedZone("GMT", 0)).Format("Mon, 2 Jan 2006 15:04:05 MST"),
		parsed:     true,
		ts:         ts,
	}
}

func (s *SipDate) GetCopy() *SipDate {
	tmp := *s
	return &tmp
}

func (s *SipDate) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipDate) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.String()
}

func (s *SipDate) String() string {
	return s.Name() + ": " + s.strBody
}

func (s *SipDate) StringBody() string {
	return s.strBody
}

func (s *SipDate) GetTime() (time.Time, error) {
	var err error

	if s.parsed {
		return s.ts, nil
	}
	s.ts, err = time.Parse("Mon, 2 Jan 2006 15:04:05 MST", s.strBody)
	if err == nil {
		s.parsed = true
	}
	return s.ts, err
}
