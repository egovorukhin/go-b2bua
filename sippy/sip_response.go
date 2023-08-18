package sippy

import (
	"errors"
	"strconv"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/time"
	"github.com/sippy/go-b2bua/sippy/types"
	"github.com/sippy/go-b2bua/sippy/utils"
)

type sipResponse struct {
	*sipMsg
	scode  int
	reason string
	sipver string
}

func ParseSipResponse(buf []byte, rtime *sippy_time.MonoTime, config sippy_conf.Config) (*sipResponse, error) {
	var scode string

	s := &sipResponse{}
	super, err := ParseSipMsg(buf, rtime, config)
	if err != nil {
		return nil, err
	}
	s.sipMsg = super
	// parse startline
	sstartline := sippy_utils.FieldsN(s.startline, 3)
	if len(sstartline) == 2 {
		// Some brain-damaged UAs don't include reason in some cases
		s.sipver, scode = sstartline[0], sstartline[1]
		s.reason = "Unspecified"
	} else if len(sstartline) == 3 {
		s.sipver, scode, s.reason = sstartline[0], sstartline[1], sstartline[2]
	} else {
		return nil, errors.New("Bad response: " + s.startline)
	}
	s.scode, err = strconv.Atoi(scode)
	if err != nil {
		return nil, err
	}
	if s.scode != 100 || s.scode < 400 {
		err = s.init_body(config.ErrorLogger())
	}
	return s, err
}

func NewSipResponse(scode int, reason, sipver string, from *sippy_header.SipFrom, callid *sippy_header.SipCallId,
	vias []*sippy_header.SipVia, to *sippy_header.SipTo, cseq *sippy_header.SipCSeq, rrs []*sippy_header.SipRecordRoute,
	body sippy_types.MsgBody, server *sippy_header.SipServer, config sippy_conf.Config) *sipResponse {
	s := &sipResponse{
		scode:  scode,
		reason: reason,
		sipver: sipver,
	}
	s.sipMsg = NewSipMsg(nil, config)
	for _, via := range vias {
		s.AppendHeader(via)
	}
	for _, rr := range rrs {
		s.AppendHeader(rr)
	}
	s.AppendHeader(from)
	s.AppendHeader(to)
	s.AppendHeader(callid)
	s.AppendHeader(cseq)
	if server != nil {
		s.AppendHeader(server)
	}
	s.body = body
	return s
}

func (s *sipResponse) LocalStr(hostPort *sippy_net.HostPort, compact bool /*= False*/) string {
	return s.GetSL() + "\r\n" + s.localStr(hostPort, compact)
}

func (s *sipResponse) GetSL() string {
	return s.sipver + " " + strconv.Itoa(s.scode) + " " + s.reason
}

func (s *sipResponse) GetCopy() sippy_types.SipResponse {
	rval := &sipResponse{
		scode:  s.scode,
		reason: s.reason,
		sipver: s.sipver,
	}
	rval.sipMsg = s.sipMsg.getCopy()
	return rval
}

func (s *sipResponse) GetSCode() (int, string) {
	return s.scode, s.reason
}

func (s *sipResponse) SetSCode(scode int, reason string) {
	s.scode, s.reason = scode, reason
}

func (s *sipResponse) GetSCodeNum() int {
	return s.scode
}

func (s *sipResponse) GetSCodeReason() string {
	return s.reason
}

func (s *sipResponse) SetSCodeReason(reason string) {
	s.reason = reason
}

func (s *sipResponse) GetRTId() (*sippy_header.RTID, error) {
	if s.rseq == nil {
		return nil, errors.New("No RSeq present")
	}
	rseq, err := s.rseq.GetBody()
	if err != nil {
		return nil, errors.New("Error parsing RSeq: " + err.Error())
	}
	cseq, err := s.cseq.GetBody()
	if err != nil {
		return nil, errors.New("Error parsing CSeq: " + err.Error())
	}
	call_id := s.call_id.StringBody()
	from_body, err := s.from.GetBody(s.config)
	if err != nil {
		return nil, errors.New("Error parsing From: " + err.Error())
	}
	from_tag := from_body.GetTag()
	return sippy_header.NewRTID(call_id, from_tag, rseq.Number, cseq.CSeq, cseq.Method), nil
}

func (s *sipResponse) GetChallenges() []sippy_types.Challenge {
	res := make([]sippy_types.Challenge, 0, len(s.sip_www_authenticates)+len(s.sip_proxy_authenticates))
	for _, challenge := range s.sip_www_authenticates {
		res = append(res, challenge)
	}
	for _, challenge := range s.sip_proxy_authenticates {
		res = append(res, challenge)
	}
	return res
}
