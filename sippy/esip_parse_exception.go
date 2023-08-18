package sippy

import (
	"github.com/sippy/go-b2bua/sippy/types"
)

type ESipParseException struct {
	sip_response sippy_types.SipResponse
	msg          string
}

func (s *ESipParseException) Error() string {
	return s.msg
}
