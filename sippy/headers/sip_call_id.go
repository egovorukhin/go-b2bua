package sippy_header

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/net"
)

type SipCallId struct {
	compactName
	CallId string
}

var sipCallIdName compactName = newCompactName("Call-ID", "i")

func CreateSipCallId(body string) []SipHeader {
	s := &SipCallId{
		compactName: sipCallIdName,
		CallId:      body,
	}
	return []SipHeader{s}
}

func (s *SipCallId) genCallId(config sippy_conf.Config) {
	buf := make([]byte, 16)
	rand.Read(buf)
	s.CallId = hex.EncodeToString(buf) + "@" + config.GetMyAddress().String()
}

func NewSipCallIdFromString(call_id string) *SipCallId {
	return &SipCallId{
		compactName: sipCallIdName,
		CallId:      call_id,
	}
}

func GenerateSipCallId(config sippy_conf.Config) *SipCallId {
	s := &SipCallId{
		compactName: sipCallIdName,
	}
	s.genCallId(config)
	return s
}

func (s *SipCallId) GetCopy() *SipCallId {
	tmp := *s
	return &tmp
}

func (s *SipCallId) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipCallId) StringBody() string {
	return s.CallId
}

func (s *SipCallId) String() string {
	return s.Name() + ": " + s.CallId
}

func (s *SipCallId) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	if compact {
		return s.CompactName() + ": " + s.CallId
	}
	return s.String()
}
