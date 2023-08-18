package sippy_header

import (
	"crypto/rand"
	"strconv"

	"github.com/sippy/go-b2bua/sippy/net"
)

type SipCiscoGUID struct {
	normalName
	body string
}

var sipCiscoGuidName normalName = newNormalName("Cisco-GUID")

func CreateSipCiscoGUID(body string) []SipHeader {
	return []SipHeader{
		&SipCiscoGUID{
			normalName: sipCiscoGuidName,
			body:       body,
		},
	}
}

func (g *SipCiscoGUID) StringBody() string {
	return g.body
}

func (g *SipCiscoGUID) String() string {
	return g.Name() + ": " + g.body
}

func (g *SipCiscoGUID) LocalStr(*sippy_net.HostPort, bool) string {
	return g.String()
}

func (g *SipCiscoGUID) AsH323ConfId() *SipH323ConfId {
	return &SipH323ConfId{
		normalName: sipH323ConfIdName,
		body:       g.body,
	}
}

func (g *SipCiscoGUID) GetCopy() *SipCiscoGUID {
	tmp := *g
	return &tmp
}

func (g *SipCiscoGUID) GetCopyAsIface() SipHeader {
	return g.GetCopy()
}

func NewSipCiscoGUID() *SipCiscoGUID {
	arr := make([]byte, 16)
	_, _ = rand.Read(arr)
	s := ""
	for i := 0; i < 4; i++ {
		i2 := i * 4
		x := uint64(arr[i2]) + (uint64(arr[i2+1]) << 8) +
			(uint64(arr[i2+2]) << 16) + (uint64(arr[i2+3]) << 24)
		if i != 0 {
			s += "-"
		}
		s += strconv.FormatUint(x, 10)
	}
	return &SipCiscoGUID{
		normalName: sipCiscoGuidName,
		body:       s,
	}
}
