package sippy

import (
	"errors"
	"strconv"
	"strings"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/fmt"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/log"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
)

type sipMsg struct {
	headers                 []sippy_header.SipHeader
	__mbody                 *string
	startline               string
	vias                    []*sippy_header.SipVia
	contacts                []*sippy_header.SipContact
	to                      *sippy_header.SipTo
	from                    *sippy_header.SipFrom
	cseq                    *sippy_header.SipCSeq
	rseq                    *sippy_header.SipRSeq
	rack                    *sippy_header.SipRAck
	content_length          *sippy_header.SipContentLength
	content_type            *sippy_header.SipContentType
	call_id                 *sippy_header.SipCallId
	refer_to                *sippy_header.SipReferTo
	maxforwards             *sippy_header.SipMaxForwards
	also                    []*sippy_header.SipAlso
	rtime                   *sippy_time.MonoTime
	body                    sippy_types.MsgBody
	source                  *sippy_net.HostPort
	record_routes           []*sippy_header.SipRecordRoute
	routes                  []*sippy_header.SipRoute
	target                  *sippy_net.HostPort
	reason_hf               *sippy_header.SipReason
	sip_warning             *sippy_header.SipWarning
	sip_www_authenticates   []*sippy_header.SipWWWAuthenticate
	sip_authorization       *sippy_header.SipAuthorization
	sip_proxy_authorization *sippy_header.SipProxyAuthorization
	sip_proxy_authenticates []*sippy_header.SipProxyAuthenticate
	sip_server              *sippy_header.SipServer
	sip_user_agent          *sippy_header.SipUserAgent
	sip_cisco_guid          *sippy_header.SipCiscoGUID
	sip_h323_conf_id        *sippy_header.SipH323ConfId
	sip_require             []*sippy_header.SipRequire
	sip_supported           []*sippy_header.SipSupported
	sip_date                *sippy_header.SipDate
	config                  sippy_conf.Config
}

func NewSipMsg(rtime *sippy_time.MonoTime, config sippy_conf.Config) *sipMsg {
	s := &sipMsg{
		headers:       make([]sippy_header.SipHeader, 0),
		__mbody:       nil,
		vias:          make([]*sippy_header.SipVia, 0),
		contacts:      make([]*sippy_header.SipContact, 0),
		record_routes: make([]*sippy_header.SipRecordRoute, 0),
		routes:        make([]*sippy_header.SipRoute, 0),
		also:          make([]*sippy_header.SipAlso, 0),
		sip_require:   make([]*sippy_header.SipRequire, 0),
		sip_supported: make([]*sippy_header.SipSupported, 0),
		rtime:         rtime,
		config:        config,
	}
	return s
}

func ParseSipMsg(_buf []byte, rtime *sippy_time.MonoTime, config sippy_conf.Config) (*sipMsg, error) {
	s := NewSipMsg(rtime, config)
	buf := string(_buf)
	// Locate a body
	for _, bdel := range []string{"\r\n\r\n", "\r\r", "\n\n"} {
		boff := strings.Index(buf, bdel)
		if boff != -1 {
			tmp := buf[boff+len(bdel):]
			s.__mbody = &tmp
			buf = buf[:boff]
			if len(*s.__mbody) == 0 {
				s.__mbody = nil
			}
			break
		}
	}
	// Split message into lines and put aside start line
	lines := strings.FieldsFunc(buf, func(c rune) bool { return c == '\n' || c == '\r' })
	s.startline = lines[0]
	header_lines := make([]string, 0)
	prev_l := ""
	for _, l := range lines[1:] {
		if l == "" || l[0] == ' ' || l[0] == '\t' {
			prev_l += strings.TrimSpace(l)
		} else {
			if len(prev_l) > 0 {
				header_lines = append(header_lines, prev_l)
			}
			prev_l = l
		}
	}
	if prev_l != "" {
		header_lines = append(header_lines, prev_l)
	}

	// Parse headers
	for _, line := range header_lines {
		headers, err := ParseSipHeader(line)
		if err != nil {
			return nil, err
		}
		for _, header := range headers {
			if contact, ok := header.(*sippy_header.SipContact); ok {
				if contact.Asterisk {
					continue
				}
			}
			s.AppendHeader(header)
		}
	}
	if len(s.vias) == 0 {
		return nil, errors.New("Via HF is missed")
	}
	if s.to == nil {
		return nil, errors.New("To HF is missed")
	}
	if s.from == nil {
		return nil, errors.New("From HF is missed")
	}
	if s.cseq == nil {
		return nil, errors.New("CSeq HF is missed")
	}
	if s.call_id == nil {
		return nil, errors.New("Call-ID HF is missed")
	}
	return s, nil
}

func (m *sipMsg) appendHeaders(hdrs []sippy_header.SipHeader) {
	if hdrs == nil {
		return
	}
	for _, hdr := range hdrs {
		m.AppendHeader(hdr)
	}
}

func (m *sipMsg) AppendHeader(hdr sippy_header.SipHeader) {
	switch t := hdr.(type) {
	case *sippy_header.SipCSeq:
		m.cseq = t
	case *sippy_header.SipRSeq:
		m.rseq = t
	case *sippy_header.SipRAck:
		m.rack = t
	case *sippy_header.SipCallId:
		m.call_id = t
	case *sippy_header.SipFrom:
		m.from = t
	case *sippy_header.SipTo:
		m.to = t
	case *sippy_header.SipMaxForwards:
		m.maxforwards = t
		return
	case *sippy_header.SipVia:
		m.vias = append(m.vias, t)
		return
	case *sippy_header.SipContentLength:
		m.content_length = t
		return
	case *sippy_header.SipContentType:
		m.content_type = t
		return
	case *sippy_header.SipExpires:
	case *sippy_header.SipRecordRoute:
		m.record_routes = append(m.record_routes, t)
	case *sippy_header.SipRoute:
		m.routes = append(m.routes, t)
		return
	case *sippy_header.SipContact:
		m.contacts = append(m.contacts, t)
	case *sippy_header.SipWWWAuthenticate:
		m.sip_www_authenticates = append(m.sip_www_authenticates, t)
	case *sippy_header.SipAuthorization:
		m.sip_authorization = t
		m.sip_proxy_authorization = nil
		return
	case *sippy_header.SipServer:
		m.sip_server = t
	case *sippy_header.SipUserAgent:
		m.sip_user_agent = t
	case *sippy_header.SipCiscoGUID:
		m.sip_cisco_guid = t
	case *sippy_header.SipH323ConfId:
		m.sip_h323_conf_id = t
	case *sippy_header.SipAlso:
		m.also = append(m.also, t)
	case *sippy_header.SipReferTo:
		m.refer_to = t
	case *sippy_header.SipCCDiversion:
	case *sippy_header.SipReferredBy:
	case *sippy_header.SipProxyAuthenticate:
		m.sip_proxy_authenticates = append(m.sip_proxy_authenticates, t)
	case *sippy_header.SipProxyAuthorization:
		m.sip_proxy_authorization = t
		m.sip_authorization = nil
		return
	case *sippy_header.SipReplaces:
	case *sippy_header.SipReason:
		m.reason_hf = t
	case *sippy_header.SipWarning:
		m.sip_warning = t
	case *sippy_header.SipRequire:
		m.sip_require = append(m.sip_require, t)
	case *sippy_header.SipSupported:
		m.sip_supported = append(m.sip_supported, t)
	case *sippy_header.SipDate:
		m.sip_date = t
	case nil:
		return
	}
	m.headers = append(m.headers, hdr)
}

func (m *sipMsg) init_body(logger sippy_log.ErrorLogger) error {
	var blen_hf *sippy_header.SipNumericHF
	if m.content_length != nil {
		blen_hf, _ = m.content_length.GetBody()
	}
	if blen_hf != nil {
		blen := blen_hf.Number
		mblen := 0
		if m.__mbody != nil {
			mblen = len([]byte(*m.__mbody)) // length in bytes, not runes
		}
		if blen == 0 {
			m.__mbody = nil
			mblen = 0
		} else if m.__mbody == nil {
			// XXX: Should generate 400 Bad Request if such condition
			// happens with request
			return &ESipParseException{msg: sippy_fmt.Sprintf("Missed SIP body, %d bytes expected", blen)}
		} else if blen > mblen {
			if blen-mblen < 7 && mblen > 7 && (*m.__mbody)[len(*m.__mbody)-4:] == "\r\n\r\n" {
				// XXX: we should not really be doing this, but it appears to be
				// a common off-by-one/two/.../six problem with SDPs generates by
				// the consumer-grade devices.
				logger.Debugf("Truncated SIP body, %d bytes expected, %d received, fixing...", blen, mblen)
				blen = mblen
			} else if blen-mblen == 2 && (*m.__mbody)[len(*m.__mbody)-2:] == "\r\n" {
				// Missed last 2 \r\n is another common problem.
				logger.Debugf("Truncated SIP body, %d bytes expected, %d received, fixing...", blen, mblen)
				(*m.__mbody) += "\r\n"
			} else if blen-mblen == 1 && (*m.__mbody)[len(*m.__mbody)-3:] == "\r\n\n" {
				// Another possible mishap
				logger.Debugf("Truncated SIP body, %d bytes expected, %d received, fixing...", blen, mblen)
				(*m.__mbody) = (*m.__mbody)[:len(*m.__mbody)-3] + "\r\n\r\n"
			} else if blen-mblen == 1 && (*m.__mbody)[len(*m.__mbody)-2:] == "\r\n" {
				// One more
				logger.Debugf("Truncated SIP body, %d bytes expected, %d received, fixing...", blen, mblen)
				(*m.__mbody) += "\r\n"
				blen += 1
				mblen += 2
			} else {
				// XXX: Should generate 400 Bad Request if such condition
				// happens with request
				return &ESipParseException{msg: sippy_fmt.Sprintf("Truncated SIP body, %d bytes expected, %d received", blen, mblen)}
			}
		} else if blen < mblen {
			*m.__mbody = (*m.__mbody)[:blen]
			mblen = blen
		}
	}
	if m.__mbody != nil {
		if m.content_type != nil {
			m.body = NewMsgBody(*m.__mbody, strings.ToLower(m.content_type.StringBody()))
		} else {
			m.body = NewMsgBody(*m.__mbody, "application/sdp")
		}
	}
	return nil
}

func (m *sipMsg) Bytes() []byte {
	s := m.startline + "\r\n"
	for _, via := range m.vias {
		s += via.String() + "\r\n"
	}
	for _, via := range m.routes {
		s += via.String() + "\r\n"
	}
	if m.maxforwards != nil {
		s += m.maxforwards.String() + "\r\n"
	}
	for _, header := range m.headers {
		s += header.String() + "\r\n"
	}
	mbody := []byte{}
	if m.body != nil {
		mbody = []byte(m.body.String())
		s += "Content-Length: " + strconv.Itoa(len(mbody)) + "\r\n"
		s += "Content-Type: " + m.body.GetMtype() + "\r\n\r\n"
	} else {
		s += "Content-Length: 0\r\n\r\n"
	}
	ret := []byte(s)
	ret = append(ret, mbody...)
	return ret
}

func (m *sipMsg) localStr(hostPort *sippy_net.HostPort, compact bool /*= False*/) (s string) {
	for _, via := range m.vias {
		s += via.LocalStr(hostPort, compact) + "\r\n"
	}
	for _, via := range m.routes {
		s += via.LocalStr(hostPort, compact) + "\r\n"
	}
	if m.maxforwards != nil {
		s += m.maxforwards.LocalStr(hostPort, compact) + "\r\n"
	}
	for _, header := range m.headers {
		s += header.LocalStr(hostPort, compact) + "\r\n"
	}
	if m.sip_authorization != nil {
		s += m.sip_authorization.LocalStr(hostPort, compact) + "\r\n"
	} else if m.sip_proxy_authorization != nil {
		s += m.sip_proxy_authorization.LocalStr(hostPort, compact) + "\r\n"
	}
	if m.body != nil {
		mbody := m.body.LocalStr(hostPort)
		bmbody := []byte(mbody)
		if compact {
			s += "l: " + strconv.Itoa(len(bmbody)) + "\r\n"
			s += "c: " + m.body.GetMtype() + "\r\n\r\n"
		} else {
			s += "Content-Length: " + strconv.Itoa(len(bmbody)) + "\r\n"
			s += "Content-Type: " + m.body.GetMtype() + "\r\n\r\n"
		}
		s += mbody
	} else {
		if compact {
			s += "l: 0\r\n\r\n"
		} else {
			s += "Content-Length: 0\r\n\r\n"
		}
	}
	return
}

func (m *sipMsg) setBody(body sippy_types.MsgBody) {
	m.body = body
}

func (m *sipMsg) GetBody() sippy_types.MsgBody {
	return m.body
}

func (m *sipMsg) SetBody(body sippy_types.MsgBody) {
	m.body = body
}

func (m *sipMsg) GetTarget() *sippy_net.HostPort {
	return m.target
}

func (m *sipMsg) SetTarget(address *sippy_net.HostPort) {
	m.target = address
}

func (m *sipMsg) GetSource() *sippy_net.HostPort {
	return m.source
}

func (m *sipMsg) GetTId(wCSM, wBRN, wTTG bool) (*sippy_header.TID, error) {
	var call_id, cseq, cseq_method, from_tag, to_tag, via_branch string
	var cseq_hf *sippy_header.SipCSeqBody
	var from_hf *sippy_header.SipAddress
	var err error

	if m.call_id != nil {
		call_id = m.call_id.CallId
	}
	if m.cseq == nil {
		return nil, errors.New("no CSeq: field")
	}
	if cseq_hf, err = m.cseq.GetBody(); err != nil {
		return nil, err
	}
	cseq = strconv.Itoa(cseq_hf.CSeq)
	if m.from != nil {
		if from_hf, err = m.from.GetBody(m.config); err != nil {
			return nil, err
		}
		from_tag = from_hf.GetTag()
	}
	if wCSM {
		cseq_method = cseq_hf.Method
	}
	if wBRN {
		if len(m.vias) > 0 {
			var via0 *sippy_header.SipViaBody
			via0, err = m.vias[0].GetBody()
			if err != nil {
				return nil, err
			}
			via_branch = via0.GetBranch()
		}
	}
	if wTTG {
		var to_hf *sippy_header.SipAddress
		to_hf, err = m.to.GetBody(m.config)
		if err != nil {
			return nil, err
		}
		to_tag = to_hf.GetTag()
	}
	return sippy_header.NewTID(call_id, cseq, cseq_method, from_tag, to_tag, via_branch), nil
}

func (m *sipMsg) getTIds() ([]*sippy_header.TID, error) {
	var call_id, cseq, method, ftag string
	var from_hf *sippy_header.SipAddress
	var cseq_hf *sippy_header.SipCSeqBody
	var err error

	call_id = m.call_id.CallId
	from_hf, err = m.from.GetBody(m.config)
	if err != nil {
		return nil, err
	}
	ftag = from_hf.GetTag()
	if m.cseq == nil {
		return nil, errors.New("no CSeq: field")
	}
	if cseq_hf, err = m.cseq.GetBody(); err != nil {
		return nil, err
	}
	if cseq_hf != nil {
		cseq, method = strconv.Itoa(cseq_hf.CSeq), cseq_hf.Method
	}
	ret := []*sippy_header.TID{}
	for _, via := range m.vias {
		var via_hf *sippy_header.SipViaBody

		if via_hf, err = via.GetBody(); err != nil {
			return nil, err
		}
		ret = append(ret, sippy_header.NewTID(call_id, cseq, method, ftag, via_hf.GetBranch(), ""))
	}
	return ret, nil
}

func (m *sipMsg) getCopy() *sipMsg {
	cself := NewSipMsg(m.rtime, m.config)
	for _, header := range m.vias {
		cself.AppendHeader(header.GetCopyAsIface())
	}
	for _, header := range m.routes {
		cself.AppendHeader(header.GetCopyAsIface())
	}
	for _, header := range m.headers {
		cself.AppendHeader(header.GetCopyAsIface())
	}
	if m.body != nil {
		cself.body = m.body.GetCopy()
	}
	cself.startline = m.startline
	cself.target = m.target
	cself.source = m.source
	return cself
}

func (m *sipMsg) GetSipProxyAuthorization() *sippy_header.SipProxyAuthorization {
	return m.sip_proxy_authorization
}

func (m *sipMsg) GetSipServer() *sippy_header.SipServer {
	return m.sip_server
}

func (m *sipMsg) GetSipUserAgent() *sippy_header.SipUserAgent {
	return m.sip_user_agent
}

func (m *sipMsg) GetCSeq() *sippy_header.SipCSeq {
	return m.cseq
}

func (m *sipMsg) GetRSeq() *sippy_header.SipRSeq {
	return m.rseq
}

func (m *sipMsg) GetSipRAck() *sippy_header.SipRAck {
	return m.rack
}

func (m *sipMsg) GetSipProxyAuthenticates() []*sippy_header.SipProxyAuthenticate {
	return m.sip_proxy_authenticates
}

func (m *sipMsg) GetSipWWWAuthenticates() []*sippy_header.SipWWWAuthenticate {
	return m.sip_www_authenticates
}

func (m *sipMsg) GetTo() *sippy_header.SipTo {
	return m.to
}

func (m *sipMsg) GetReason() *sippy_header.SipReason {
	return m.reason_hf
}

func (m *sipMsg) GetVias() []*sippy_header.SipVia {
	return m.vias
}

func (m *sipMsg) GetCallId() *sippy_header.SipCallId {
	return m.call_id
}

func (m *sipMsg) SetRtime(rtime *sippy_time.MonoTime) {
	m.rtime = rtime
}

func (m *sipMsg) InsertFirstVia(via *sippy_header.SipVia) {
	m.vias = append([]*sippy_header.SipVia{via}, m.vias...)
}

func (m *sipMsg) RemoveFirstVia() {
	m.vias = m.vias[1:]
}

func (m *sipMsg) SetRoutes(routes []*sippy_header.SipRoute) {
	m.routes = routes
}

func (m *sipMsg) GetFrom() *sippy_header.SipFrom {
	return m.from
}

func (m *sipMsg) GetReferTo() *sippy_header.SipReferTo {
	return m.refer_to
}

func (m *sipMsg) GetRtime() *sippy_time.MonoTime {
	return m.rtime
}

func (m *sipMsg) GetAlso() []*sippy_header.SipAlso {
	return m.also
}

func (m *sipMsg) GetContacts() []*sippy_header.SipContact {
	return m.contacts
}

func (m *sipMsg) GetRecordRoutes() []*sippy_header.SipRecordRoute {
	return m.record_routes
}

func (m *sipMsg) GetCGUID() *sippy_header.SipCiscoGUID {
	return m.sip_cisco_guid
}

func (m *sipMsg) GetH323ConfId() *sippy_header.SipH323ConfId {
	return m.sip_h323_conf_id
}

func (m *sipMsg) GetSipAuthorization() *sippy_header.SipAuthorization {
	return m.sip_authorization
}

func match_name(name string, hf sippy_header.SipHeader) bool {
	return strings.ToLower(hf.Name()) == strings.ToLower(name) ||
		strings.ToLower(hf.CompactName()) == strings.ToLower(name)
}

func (m *sipMsg) GetFirstHF(name string) sippy_header.SipHeader {
	for _, hf := range m.headers {
		if match_name(name, hf) {
			return hf
		}
	}
	if m.content_length != nil && match_name(name, m.content_length) {
		return m.content_length
	}
	if m.content_type != nil && match_name(name, m.content_type) {
		return m.content_type
	}
	if len(m.vias) > 0 && match_name(name, m.vias[0]) {
		return m.vias[0]
	}
	if len(m.routes) > 0 && match_name(name, m.routes[0]) {
		return m.routes[0]
	}
	return nil
}

func (m *sipMsg) GetHFs(name string) []sippy_header.SipHeader {
	rval := make([]sippy_header.SipHeader, 0)
	for _, hf := range m.headers {
		if match_name(name, hf) {
			rval = append(rval, hf)
		}
	}
	if m.content_length != nil && match_name(name, m.content_length) {
		rval = append(rval, m.content_length)
	}
	if m.content_type != nil && match_name(name, m.content_type) {
		rval = append(rval, m.content_type)
	}
	if len(m.vias) > 0 && match_name(name, m.vias[0]) {
		for _, via := range m.vias {
			rval = append(rval, via)
		}
	}
	if len(m.routes) > 0 && match_name(name, m.routes[0]) {
		for _, route := range m.routes {
			rval = append(rval, route)
		}
	}
	return rval
}

func (m *sipMsg) GetMaxForwards() *sippy_header.SipMaxForwards {
	return m.maxforwards
}

func (m *sipMsg) SetMaxForwards(maxforwards *sippy_header.SipMaxForwards) {
	m.maxforwards = maxforwards
}

func (m *sipMsg) GetSipRequire() []*sippy_header.SipRequire {
	return m.sip_require
}

func (m *sipMsg) GetSipSupported() []*sippy_header.SipSupported {
	return m.sip_supported
}

func (m *sipMsg) GetSipDate() *sippy_header.SipDate {
	return m.sip_date
}
