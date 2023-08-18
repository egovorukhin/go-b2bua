package sippy

import (
	"github.com/sippy/go-b2bua/sippy/types"
)

type redirectController struct {
	ua sippy_types.UA
}

func newRedirectController(ua sippy_types.UA) *redirectController {
	return &redirectController{
		ua: ua,
	}
}

func (s *redirectController) RecvResponse(resp sippy_types.SipResponse, t sippy_types.ClientTransaction) {
	req, err := s.ua.GenRequest("BYE", nil, nil)
	if err != nil {
		return
	}
	s.ua.BeginNewClientTransaction(req, nil)
}
