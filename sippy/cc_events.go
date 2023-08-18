package sippy

import (
	"sync"

	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

var global_event_seq int64 = 1
var global_event_seq_lock sync.Mutex

type CCEventGeneric struct {
	seq              int64
	rtime            *sippy_time.MonoTime
	extra_headers    []sippy_header.SipHeader
	origin           string
	sip_reason       *sippy_header.SipReason
	sip_max_forwards *sippy_header.SipMaxForwards
}

func newCCEventGeneric(rtime *sippy_time.MonoTime, origin string, extra_headers ...sippy_header.SipHeader) CCEventGeneric {
	global_event_seq_lock.Lock()
	new_seq := global_event_seq
	global_event_seq++
	global_event_seq_lock.Unlock()
	if rtime == nil {
		rtime, _ = sippy_time.NewMonoTime()
	}
	s := CCEventGeneric{
		rtime:         rtime,
		seq:           new_seq,
		origin:        origin,
		extra_headers: make([]sippy_header.SipHeader, 0, len(extra_headers)),
	}
	for _, eh := range extra_headers {
		switch header := eh.(type) {
		case *sippy_header.SipMaxForwards:
			s.sip_max_forwards = header
		case *sippy_header.SipReason:
			s.sip_reason = header
		default:
			s.extra_headers = append(s.extra_headers, eh)
		}
	}
	return s
}

func (s *CCEventGeneric) GetMaxForwards() *sippy_header.SipMaxForwards {
	return s.sip_max_forwards
}

func (s *CCEventGeneric) SetMaxForwards(max_forwards *sippy_header.SipMaxForwards) {
	s.sip_max_forwards = max_forwards
}

func (s *CCEventGeneric) AppendExtraHeader(eh sippy_header.SipHeader) {
	s.extra_headers = append(s.extra_headers, eh)
}

func (s *CCEventGeneric) GetReason() *sippy_header.SipReason {
	return s.sip_reason
}

func (s *CCEventGeneric) SetReason(sip_reason *sippy_header.SipReason) {
	s.sip_reason = sip_reason
}

func (s *CCEventGeneric) GetSeq() int64 {
	return s.seq
}

func (s *CCEventGeneric) GetRtime() *sippy_time.MonoTime {
	return s.rtime
}

func (s *CCEventGeneric) GetOrigin() string {
	return s.origin
}

func (s *CCEventGeneric) GetExtraHeaders() []sippy_header.SipHeader {
	ret := s.extra_headers
	if s.sip_reason != nil {
		ret = append(ret, s.sip_reason)
	}
	// The max_forwards should not be present here
	//if s.sip_max_forwards != nil { ret = append(ret, s.sip_max_forwards) }
	return ret
}

type CCEventTry struct {
	CCEventGeneric
	call_id     *sippy_header.SipCallId
	cli         string
	cld         string
	caller_name string
	auth_body   *sippy_header.SipAuthorizationBody
	auth_hdr    sippy_header.SipAuthorizationHeader
	body        sippy_types.MsgBody
	routes      []*sippy_header.SipRoute
}

func NewCCEventTry(call_id *sippy_header.SipCallId, cli string, cld string, body sippy_types.MsgBody, auth_hdr sippy_header.SipAuthorizationHeader, caller_name string, rtime *sippy_time.MonoTime, origin string, extra_headers ...sippy_header.SipHeader) (*CCEventTry, error) {
	var err error
	var auth_body *sippy_header.SipAuthorizationBody

	if auth_hdr != nil {
		auth_body, err = auth_hdr.GetBody()
		if err != nil {
			return nil, err
		}
	}
	return &CCEventTry{
		CCEventGeneric: newCCEventGeneric(rtime, origin, extra_headers...),
		call_id:        call_id,
		cli:            cli,
		cld:            cld,
		auth_hdr:       auth_hdr,
		auth_body:      auth_body,
		caller_name:    caller_name,
		body:           body,
		routes:         []*sippy_header.SipRoute{},
	}, nil
}

func (s *CCEventTry) GetBody() sippy_types.MsgBody {
	return s.body
}

func (s *CCEventTry) GetSipAuthorizationHF() sippy_header.SipAuthorizationHeader {
	return s.auth_hdr
}

func (s *CCEventTry) GetSipAuthorizationBody() *sippy_header.SipAuthorizationBody {
	return s.auth_body
}

func (s *CCEventTry) GetSipCallId() *sippy_header.SipCallId {
	return s.call_id
}

func (s *CCEventTry) GetCallerName() string {
	return s.caller_name
}

func (s *CCEventTry) GetCLD() string {
	return s.cld
}

func (s *CCEventTry) GetCLI() string {
	return s.cli
}

func (s *CCEventTry) String() string { return "CCEventTry" }

type CCEventRing struct {
	CCEventGeneric
	scode        int
	scode_reason string
	body         sippy_types.MsgBody
}

func (s *CCEventTry) SetRoutes(routes []*sippy_header.SipRoute) {
	s.routes = routes
}

func NewCCEventRing(scode int, scode_reason string, body sippy_types.MsgBody, rtime *sippy_time.MonoTime, origin string) *CCEventRing {
	return &CCEventRing{
		CCEventGeneric: newCCEventGeneric(rtime, origin),
		scode:          scode,
		scode_reason:   scode_reason,
		body:           body,
	}
}

func (s *CCEventRing) String() string { return "CCEventRing" }

type CCEventConnect struct {
	CCEventGeneric
	scode        int
	scode_reason string
	body         sippy_types.MsgBody
}

func (s *CCEventRing) GetScode() int                      { return s.scode }
func (s *CCEventRing) GetBody() sippy_types.MsgBody       { return s.body }
func (s *CCEventRing) SetScode(scode int)                 { s.scode = scode }
func (s *CCEventRing) SetScodeReason(scode_reason string) { s.scode_reason = scode_reason }

func NewCCEventConnect(scode int, scode_reason string, msg_body sippy_types.MsgBody, rtime *sippy_time.MonoTime, origin string, extra_headers ...sippy_header.SipHeader) *CCEventConnect {
	return &CCEventConnect{
		CCEventGeneric: newCCEventGeneric(rtime, origin, extra_headers...),
		scode:          scode,
		scode_reason:   scode_reason,
		body:           msg_body,
	}
}

func (s *CCEventConnect) String() string { return "CCEventConnect" }

func (s *CCEventConnect) GetBody() sippy_types.MsgBody {
	return s.body
}

type CCEventUpdate struct {
	CCEventGeneric
	body sippy_types.MsgBody
}

func NewCCEventUpdate(rtime *sippy_time.MonoTime, origin string, reason *sippy_header.SipReason, max_forwards *sippy_header.SipMaxForwards, msg_body sippy_types.MsgBody) *CCEventUpdate {
	s := &CCEventUpdate{
		CCEventGeneric: newCCEventGeneric(rtime, origin),
		body:           msg_body,
	}
	s.SetReason(reason)
	s.SetMaxForwards(max_forwards)
	return s
}

func (s *CCEventUpdate) String() string { return "CCEventUpdate" }

func (s *CCEventUpdate) GetBody() sippy_types.MsgBody {
	return s.body
}

type CCEventInfo struct {
	CCEventGeneric
	body sippy_types.MsgBody
}

func NewCCEventInfo(rtime *sippy_time.MonoTime, origin string, msg_body sippy_types.MsgBody, extra_headers ...sippy_header.SipHeader) *CCEventInfo {
	return &CCEventInfo{
		CCEventGeneric: newCCEventGeneric(rtime, origin, extra_headers...),
	}
}

func (s *CCEventInfo) String() string { return "CCEventInfo" }

func (s *CCEventInfo) GetBody() sippy_types.MsgBody {
	return s.body
}

type CCEventDisconnect struct {
	CCEventGeneric
	redirect_url *sippy_header.SipAddress
}

func NewCCEventDisconnect(also *sippy_header.SipAddress, rtime *sippy_time.MonoTime, origin string, extra_headers ...sippy_header.SipHeader) *CCEventDisconnect {
	return &CCEventDisconnect{
		CCEventGeneric: newCCEventGeneric(rtime, origin, extra_headers...),
		redirect_url:   also,
	}
}

func (s *CCEventDisconnect) String() string { return "CCEventDisconnect" }

func (s *CCEventDisconnect) GetRedirectURL() *sippy_header.SipAddress {
	return s.redirect_url
}

func (*CCEventDisconnect) GetBody() sippy_types.MsgBody {
	return nil
}

type CCEventFail struct {
	CCEventGeneric
	challenges   []sippy_header.SipHeader
	scode        int
	scode_reason string
	warning      *sippy_header.SipWarning
}

func NewCCEventFail(scode int, scode_reason string, rtime *sippy_time.MonoTime, origin string, extra_headers ...sippy_header.SipHeader) *CCEventFail {
	return &CCEventFail{
		CCEventGeneric: newCCEventGeneric(rtime, origin, extra_headers...),
		scode_reason:   scode_reason,
		scode:          scode,
	}
}

func (s *CCEventFail) String() string { return "CCEventFail" }

func (s *CCEventFail) GetScode() int                { return s.scode }
func (s *CCEventFail) SetScode(scode int)           { s.scode = scode }
func (s *CCEventFail) GetScodeReason() string       { return s.scode_reason }
func (s *CCEventFail) SetScodeReason(reason string) { s.scode_reason = reason }

func (s *CCEventFail) GetExtraHeaders() []sippy_header.SipHeader {
	extra_headers := s.CCEventGeneric.GetExtraHeaders()
	extra_headers = append(extra_headers, s.challenges...)
	return extra_headers
}

func (s *CCEventFail) SetWarning(text string) {
	s.warning = sippy_header.NewSipWarning(text)
}

func (*CCEventFail) GetBody() sippy_types.MsgBody {
	return nil
}

type CCEventPreConnect struct {
	CCEventGeneric
	scode        int
	scode_reason string
	body         sippy_types.MsgBody
}

func NewCCEventPreConnect(scode int, scode_reason string, body sippy_types.MsgBody, rtime *sippy_time.MonoTime, origin string) *CCEventPreConnect {
	return &CCEventPreConnect{
		CCEventGeneric: newCCEventGeneric(rtime, origin),
		scode:          scode,
		scode_reason:   scode_reason,
		body:           body,
	}
}

func (s *CCEventPreConnect) String() string               { return "CCEventPreConnect" }
func (s *CCEventPreConnect) GetScode() int                { return s.scode }
func (s *CCEventPreConnect) GetScodeReason() string       { return s.scode_reason }
func (s *CCEventPreConnect) GetBody() sippy_types.MsgBody { return s.body }
