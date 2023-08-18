package sippy

import (
	"errors"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type sipRequest struct {
	*sipMsg
	method     string
	sipver     string
	ruri       *sippy_header.SipURL
	expires    *sippy_header.SipExpires
	user_agent *sippy_header.SipUserAgent
	nated      bool
}

func ParseSipRequest(buf []byte, rtime *sippy_time.MonoTime, config sippy_conf.Config) (*sipRequest, error) {
	s := &sipRequest{nated: false}
	super, err := ParseSipMsg(buf, rtime, config)
	if err != nil {
		return nil, err
	}
	s.sipMsg = super
	arr := strings.Fields(s.startline)
	if len(arr) != 3 {
		return nil, errors.New("SIP bad start line in SIP request: " + s.startline)
	}
	s.method, s.sipver = arr[0], arr[2]
	s.ruri, err = sippy_header.ParseSipURL(arr[1], false /* relaxedparser */, config)
	if err != nil {
		return nil, errors.New("Bad SIP URL in SIP request: " + arr[1])
	}
	err = s.init_body(config.ErrorLogger())
	if err != nil {
		if e, ok := err.(*ESipParseException); ok {
			e.sip_response = s.GenResponse(400, "Bad Request - "+e.Error(), nil, nil)
		}
	}
	return s, err
}

func NewSipRequest(method string, ruri *sippy_header.SipURL, sipver string, to *sippy_header.SipTo,
	from *sippy_header.SipFrom, via *sippy_header.SipVia, cseq int, callid *sippy_header.SipCallId,
	maxforwards *sippy_header.SipMaxForwards, body sippy_types.MsgBody, contact *sippy_header.SipContact,
	routes []*sippy_header.SipRoute, target *sippy_net.HostPort, user_agent *sippy_header.SipUserAgent,
	expires *sippy_header.SipExpires, config sippy_conf.Config) (*sipRequest, error) {
	if routes == nil {
		routes = make([]*sippy_header.SipRoute, 0)
	}
	s := &sipRequest{nated: false}
	s.sipMsg = NewSipMsg(nil, config)
	s.method = method
	s.ruri = ruri
	if target == nil {
		if len(routes) == 0 {
			s.SetTarget(s.ruri.GetAddr(config))
		} else {
			var r0 *sippy_header.SipAddress
			var err error
			if r0, err = routes[0].GetBody(config); err != nil {
				return nil, err
			}
			s.SetTarget(r0.GetUrl().GetAddr(config))
		}
	} else {
		s.SetTarget(target)
	}
	if sipver == "" {
		s.sipver = "SIP/2.0"
	} else {
		s.sipver = sipver
	}
	if via == nil {
		var via0 *sippy_header.SipViaBody
		var err error
		s.AppendHeader(sippy_header.NewSipVia(config))
		if via0, err = s.vias[0].GetBody(); err != nil {
			return nil, err
		}
		via0.GenBranch()
	} else {
		s.AppendHeader(via)
	}
	for _, route := range routes {
		s.AppendHeader(route)
	}
	if maxforwards == nil {
		maxforwards = sippy_header.NewSipMaxForwardsDefault()
	}
	s.AppendHeader(maxforwards)
	if from == nil {
		from = sippy_header.NewSipFrom(nil, config)
	}
	s.AppendHeader(from)
	if to == nil {
		to = sippy_header.NewSipTo(sippy_header.NewSipAddress("", ruri), config)
	}
	s.AppendHeader(to)
	if callid == nil {
		callid = sippy_header.GenerateSipCallId(config)
	}
	s.AppendHeader(callid)
	s.AppendHeader(sippy_header.NewSipCSeq(cseq, method))
	if contact != nil {
		s.AppendHeader(contact)
	}
	if expires == nil && method == "INVITE" {
		s.AppendHeader(sippy_header.NewSipExpires())
	} else if expires != nil {
		s.AppendHeader(expires)
	}
	if user_agent != nil {
		s.user_agent = user_agent
	} else {
		s.user_agent = sippy_header.NewSipUserAgent(config.GetMyUAName())
	}
	s.AppendHeader(s.user_agent)
	s.setBody(body)
	return s, nil
}

func (s *sipRequest) LocalStr(hostPort *sippy_net.HostPort, compact bool /*= False*/) string {
	return s.GetSL() + "\r\n" + s.localStr(hostPort, compact)
}

func (s *sipRequest) GetTo() *sippy_header.SipTo {
	return s.to
}

func (s *sipRequest) GetSL() string {
	return s.method + " " + s.ruri.String() + " " + s.sipver
}

func (s *sipRequest) GetMethod() string {
	return s.method
}

func (s *sipRequest) GetRURI() *sippy_header.SipURL {
	return s.ruri
}

func (s *sipRequest) SetRURI(ruri *sippy_header.SipURL) {
	s.ruri = ruri
}

func (s *sipRequest) GenResponse(scode int, reason string, body sippy_types.MsgBody, server *sippy_header.SipServer) sippy_types.SipResponse {
	// Should be done at the transaction level
	// to = s.getHF('to').getBody().getCopy()
	// if code > 100 and to.getTag() == None:
	//    to.genTag()
	vias := make([]*sippy_header.SipVia, 0)
	rrs := make([]*sippy_header.SipRecordRoute, 0)
	for _, via := range s.vias {
		vias = append(vias, via.GetCopy())
	}
	for _, rr := range s.record_routes {
		rrs = append(rrs, rr.GetCopy())
	}
	return NewSipResponse(scode, reason, s.sipver, s.from.GetCopy(),
		s.call_id.GetCopy(), vias, s.to.GetCopy(),
		s.cseq.GetCopy(), rrs, body, server, s.config)
}

func (s *sipRequest) GenACK(to *sippy_header.SipTo) (sippy_types.SipRequest, error) {
	if to == nil {
		to = s.to.GetCopy()
	}
	var maxforwards *sippy_header.SipMaxForwards = nil

	if s.maxforwards != nil {
		maxforwards = s.maxforwards.GetCopy()
	}
	cseq, err := s.cseq.GetBody()
	if err != nil {
		return nil, err
	}
	return NewSipRequest("ACK", s.ruri.GetCopy(), s.sipver,
		to, s.from.GetCopy(), s.vias[0].GetCopy(),
		cseq.CSeq, s.call_id.GetCopy(),
		maxforwards /*body*/, nil /*contact*/, nil,
		/*routes*/ nil /*target*/, nil, s.user_agent,
		/*expires*/ nil, s.config)
}

func (s *sipRequest) GenCANCEL() (sippy_types.SipRequest, error) {
	var maxforwards *sippy_header.SipMaxForwards = nil

	if s.maxforwards != nil {
		maxforwards = s.maxforwards.GetCopy()
	}
	routes := make([]*sippy_header.SipRoute, len(s.routes))
	for i, r := range s.routes {
		routes[i] = r.GetCopy()
	}
	cseq, err := s.cseq.GetBody()
	if err != nil {
		return nil, err
	}
	return NewSipRequest("CANCEL", s.ruri.GetCopy(), s.sipver,
		s.to.GetCopy(), s.from.GetCopy(), s.vias[0].GetCopy(),
		cseq.CSeq, s.call_id.GetCopy(), maxforwards /*body*/, nil,
		/*contact*/ nil, routes, s.GetTarget(), s.user_agent,
		/*expires*/ nil, s.config)
}

func (s *sipRequest) GetExpires() *sippy_header.SipExpires {
	return s.expires
}

func (s *sipRequest) GetNated() bool {
	return s.nated
}

func (s *sipRequest) GetRTId() (*sippy_header.RTID, error) {
	if s.rack == nil {
		return nil, errors.New("No RAck field present")
	}
	rack, err := s.rack.GetBody()
	if err != nil {
		return nil, errors.New("Error parsing RSeq: " + err.Error())
	}
	call_id := s.call_id.StringBody()
	from_body, err := s.GetFrom().GetBody(s.config)
	if err != nil {
		return nil, errors.New("Error parsing From: " + err.Error())
	}
	from_tag := from_body.GetTag()
	return sippy_header.NewRTID(call_id, from_tag, rack.RSeq, rack.CSeq, rack.Method), nil
}

func (s *sipRequest) GetSipAuthorizationHF() sippy_header.SipAuthorizationHeader {
	if s.sip_authorization != nil {
		return s.sip_authorization
	}
	if s.sip_proxy_authorization != nil {
		return s.sip_proxy_authorization
	}
	return nil
}
