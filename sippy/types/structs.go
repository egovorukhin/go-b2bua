package sippy_types

import (
	"github.com/egovorukhin/go-b2bua/sippy/time"
)

type UaContext struct {
	Response SipResponse
	CancelCB func(*sippy_time.MonoTime, SipRequest)
	NoAckCB  func(*sippy_time.MonoTime)
}
