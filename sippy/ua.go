package sippy

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type Ua struct {
	sip_tm                 sippy_types.SipTransactionManager
	sip_tm_lock            sync.RWMutex
	config                 sippy_conf.Config
	call_controller        sippy_types.CallController
	session_lock           sync.Locker
	state                  sippy_types.UaState
	equeue                 []sippy_types.CCEvent
	elast_seq              int64
	setup_ts               *sippy_time.MonoTime
	connect_ts             *sippy_time.MonoTime
	disconnect_ts          *sippy_time.MonoTime
	origin                 string
	on_local_sdp_change    sippy_types.OnLocalSdpChange
	on_remote_sdp_change   sippy_types.OnRemoteSdpChange
	cId                    *sippy_header.SipCallId
	rTarget                *sippy_header.SipURL
	rAddr0                 *sippy_net.HostPort
	rUri                   *sippy_header.SipTo
	lUri                   *sippy_header.SipFrom
	lTag                   string
	lCSeq                  int
	lContact               *sippy_header.SipContact
	routes                 []*sippy_header.SipRoute
	lSDP                   sippy_types.MsgBody
	rSDP                   sippy_types.MsgBody
	outbound_proxy         *sippy_net.HostPort
	rAddr                  *sippy_net.HostPort
	local_ua               *sippy_header.SipUserAgent
	username               string
	password               string
	extra_headers          []sippy_header.SipHeader
	dlg_headers            []sippy_header.SipHeader
	reqs                   map[int]*sipRequest
	tr                     sippy_types.ClientTransaction
	source_address         *sippy_net.HostPort
	remote_ua              string
	expire_time            time.Duration
	expire_timer           *Timeout
	no_progress_time       time.Duration
	no_reply_time          time.Duration
	no_reply_timer         *Timeout
	no_progress_timer      *Timeout
	ltag                   string
	rCSeq                  int
	branch                 string
	conn_cb                sippy_types.OnConnectListener
	dead_cb                sippy_types.OnDeadListener
	disc_cb                sippy_types.OnDisconnectListener
	fail_cb                sippy_types.OnFailureListener
	ring_cb                sippy_types.OnRingingListener
	credit_timer           *Timeout
	uasResp                sippy_types.SipResponse
	useRefer               bool
	kaInterval             time.Duration
	godead_timeout         time.Duration
	last_scode             int
	_np_mtime              *sippy_time.MonoTime
	_nr_mtime              *sippy_time.MonoTime
	_ex_mtime              *sippy_time.MonoTime
	p100_ts                *sippy_time.MonoTime
	p1xx_ts                *sippy_time.MonoTime
	credit_time            *time.Duration
	credit_times           map[int64]*sippy_time.MonoTime
	auth                   sippy_header.SipHeader
	pass_auth              bool
	pending_tr             sippy_types.ClientTransaction
	late_media             bool
	heir                   sippy_types.UA
	uas_lossemul           int
	on_uac_setup_complete  func()
	expire_starts_on_setup bool
	pr_rel                 bool
	auth_enalgs            map[string]bool
}

func (s *Ua) me() sippy_types.UA {
	if s.heir == nil {
		return s
	}
	return s.heir
}

func (s *Ua) UasLossEmul() int {
	return s.uas_lossemul
}

func (s *Ua) String() string {
	return "UA state: " + s.state.String() + ", Call-Id: " + s.cId.CallId
}

func NewUA(sip_tm sippy_types.SipTransactionManager, config sippy_conf.Config, nh_address *sippy_net.HostPort, call_controller sippy_types.CallController, session_lock sync.Locker, heir sippy_types.UA) *Ua {
	return &Ua{
		sip_tm:          sip_tm,
		call_controller: call_controller,
		equeue:          make([]sippy_types.CCEvent, 0),
		elast_seq:       -1,
		reqs:            make(map[int]*sipRequest),
		rCSeq:           -1,
		useRefer:        true,
		kaInterval:      0,
		godead_timeout:  time.Duration(32 * time.Second),
		last_scode:      100,
		p100_ts:         nil,
		p1xx_ts:         nil,
		credit_times:    make(map[int64]*sippy_time.MonoTime),
		config:          config,
		rAddr:           nh_address,
		rAddr0:          nh_address,
		ltag:            sippy_utils.GenTag(),
		//fail_cb         : nil,
		//ring_cb         : nil,
		//disc_cb         : nil,
		//conn_cb         : nil,
		//dead_cb         : nil,
		session_lock:           session_lock,
		pass_auth:              false,
		late_media:             false,
		heir:                   heir,
		expire_starts_on_setup: true,
		pr_rel:                 false,
	}
}

func (s *Ua) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) *sippy_types.UaContext {
	//print "Received request %s in state %s instance %s" % (req.getMethod(), s.state, s)
	//print s.rCSeq, req.getHFBody("cseq").getCSeqNum()
	t.SetBeforeResponseSent(s.me().BeforeResponseSent)
	if s.remote_ua == "" {
		s.update_ua(req)
	}
	cseq_body, err := req.GetCSeq().GetBody()
	if err != nil || (s.rCSeq != -1 && s.rCSeq >= cseq_body.CSeq) {
		return &sippy_types.UaContext{
			Response: req.GenResponse(500, "Server Internal Error" /*body*/, nil /*server*/, s.local_ua.AsSipServer()),
			CancelCB: nil,
			NoAckCB:  nil,
		}
	}
	s.rCSeq = cseq_body.CSeq
	if s.state == nil {
		if req.GetMethod() == "INVITE" {
			if req.GetBody() == nil {
				found := false
			REQ_LOOP:
				for _, sip_req_str := range req.GetSipRequire() {
					for _, it := range strings.Split(sip_req_str.StringBody(), ",") {
						if strings.TrimSpace(it) == "100rel" {
							found = true
							break REQ_LOOP
						}
					}
				}
				if found {
					resp := req.GenResponse(420, "Bad Extension" /*body*/, nil /*server*/, s.local_ua.AsSipServer())
					usup := sippy_header.NewSipGenericHF("Unsupported", "100rel")
					resp.AppendHeader(usup)
					return &sippy_types.UaContext{
						Response: resp,
						CancelCB: nil,
						NoAckCB:  nil,
					}
				}
			} else {
				t.Setup100rel(req)
			}
			s.pr_rel = t.PrRel()
			s.me().ChangeState(NewUasStateIdle(s.me(), s.config), nil)
		} else {
			return nil
		}
	}
	newstate, cb := s.state.RecvRequest(req, t)
	if newstate != nil {
		s.me().ChangeState(newstate, cb)
	}
	s.emitPendingEvents()
	if newstate != nil && req.GetMethod() == "INVITE" {
		disc_fn := func(rtime *sippy_time.MonoTime) { s.me().Disconnect(rtime, "") }
		if s.pr_rel {
			t.SetPrackCBs(s.RecvPRACK, disc_fn)
		}
		return &sippy_types.UaContext{
			Response: nil,
			CancelCB: s.state.RecvCancel,
			NoAckCB:  disc_fn,
		}
	} else {
		return nil
	}
}

func (s *Ua) RecvResponse(resp sippy_types.SipResponse, tr sippy_types.ClientTransaction) {
	var err error
	var cseq_body *sippy_header.SipCSeqBody

	if s.state == nil {
		return
	}
	cseq_body, err = resp.GetCSeq().GetBody()
	if err != nil {
		s.logError("UA::RecvResponse: cannot parse CSeq: " + err.Error())
		return
	}
	s.update_ua(resp)
	code, _ := resp.GetSCode()
	orig_req, cseq_found := s.reqs[cseq_body.CSeq]
	if cseq_body.Method == "INVITE" && !s.pass_auth && cseq_found {
		if code == 401 && s.processWWWChallenge(resp, cseq_body.CSeq, orig_req, tr.GetReqExtraHeaders()) {
			return
		} else if code == 407 && s.processProxyChallenge(resp, cseq_body.CSeq, orig_req, tr.GetReqExtraHeaders()) {
			return
		}
	}
	if code >= 200 && cseq_found {
		delete(s.reqs, cseq_body.CSeq)
	}
	newstate, cb := s.state.RecvResponse(resp, tr)
	if newstate != nil {
		s.me().ChangeState(newstate, cb)
	}
	s.emitPendingEvents()
}

func (s *Ua) PrepTr(req sippy_types.SipRequest, eh []sippy_header.SipHeader) (sippy_types.ClientTransaction, error) {
	sip_tm := s.get_sip_tm()
	if sip_tm == nil {
		return nil, errors.New("UA already dead")
	}
	tr, err := sip_tm.CreateClientTransaction(req, s.me(), s.session_lock /*lAddress*/, s.source_address /*udp_server*/, nil, eh, s.me().BeforeRequestSent)
	if err != nil {
		return nil, err
	}
	tr.SetOutboundProxy(s.outbound_proxy)
	if s.isConnected() {
		routes := make([]*sippy_header.SipRoute, len(s.routes))
		copy(routes, s.routes)
		if s.outbound_proxy == nil {
			tr.SetAckRparams(s.rAddr, s.rTarget, routes)
		} else {
			tr.SetAckRparams(s.outbound_proxy, s.rTarget, routes)
		}
	}
	return tr, nil
}

func (s *Ua) RecvEvent(event sippy_types.CCEvent) {
	//print s, event
	if s.state == nil {
		switch event.(type) {
		case *CCEventTry:
		case *CCEventFail:
		case *CCEventDisconnect:
		default:
			return
		}
		s.me().ChangeState(NewUacStateIdle(s.me(), s.config), nil)
	}
	newstate, cb, err := s.state.RecvEvent(event)
	if err != nil {
		s.logError("UA::RecvEvent error #1: " + err.Error())
		return
	}
	if newstate != nil {
		s.me().ChangeState(newstate, cb)
	}
	s.emitPendingEvents()
}

func (s *Ua) Disconnect(rtime *sippy_time.MonoTime, origin string) {
	sip_tm := s.get_sip_tm()
	if sip_tm == nil {
		return // we are already in a dead state
	}
	if rtime == nil {
		rtime, _ = sippy_time.NewMonoTime()
	}
	s.equeue = append(s.equeue, NewCCEventDisconnect(nil, rtime, origin))
	s.RecvEvent(NewCCEventDisconnect(nil, rtime, origin))
}

func (s *Ua) expires() {
	s.expire_timer = nil
	s.me().Disconnect(nil, "")
}

func (s *Ua) no_progress_expires() {
	s.no_progress_timer = nil
	s.me().Disconnect(nil, "")
}

func (s *Ua) no_reply_expires() {
	s.no_reply_timer = nil
	s.me().Disconnect(nil, "")
}

func (s *Ua) credit_expires(rtime *sippy_time.MonoTime) {
	s.credit_timer = nil
	s.me().Disconnect(rtime, "")
}

func (s *Ua) ChangeState(newstate sippy_types.UaState, cb func()) {
	if s.state != nil {
		s.state.OnDeactivate()
	}
	s.state = newstate //.Newstate(s, s.config)
	if newstate != nil {
		newstate.OnActivation()
		if cb != nil {
			cb()
		}
	}
}

func (s *Ua) EmitEvent(event sippy_types.CCEvent) {
	if s.call_controller != nil {
		if s.elast_seq != -1 && s.elast_seq >= event.GetSeq() {
			//print "ignoring out-of-order event", event, event.seq, s.elast_seq, s.cId
			return
		}
		s.elast_seq = event.GetSeq()
		sippy_utils.SafeCall(func() { s.call_controller.RecvEvent(event, s.me()) }, nil, s.config.ErrorLogger())
	}
}

func (s *Ua) emitPendingEvents() {
	for len(s.equeue) != 0 && s.call_controller != nil {
		event := s.equeue[0]
		s.equeue = s.equeue[1:]
		if s.elast_seq != -1 && s.elast_seq >= event.GetSeq() {
			//print "ignoring out-of-order event", event, event.seq, s.elast_seq, s.cId
			continue
		}
		s.elast_seq = event.GetSeq()
		sippy_utils.SafeCall(func() { s.call_controller.RecvEvent(event, s.me()) }, nil, s.config.ErrorLogger())
	}
}

func (s *Ua) GenRequest(method string, body sippy_types.MsgBody, challenge sippy_types.Challenge, extra_headers ...sippy_header.SipHeader) (sippy_types.SipRequest, error) {
	var target *sippy_net.HostPort

	if s.outbound_proxy != nil {
		target = s.outbound_proxy
	} else {
		target = s.rAddr
	}
	req, err := NewSipRequest(method /*ruri*/, s.rTarget /*sipver*/, "" /*to*/, s.rUri /*fr0m*/, s.lUri,
		/*via*/ nil, s.lCSeq, s.cId /*maxforwars*/, nil, body, s.lContact, s.routes,
		target /*user_agent*/, s.local_ua /*expires*/, nil, s.config)
	if err != nil {
		return nil, err
	}
	if challenge != nil {
		entity_body := ""
		if body != nil {
			entity_body = body.String()
		}
		auth, err := challenge.GenAuthHF(s.username, s.password, method, s.rTarget.String(), entity_body)
		if err == nil {
			req.AppendHeader(auth)
		}
	}
	if s.extra_headers != nil {
		req.appendHeaders(s.extra_headers)
	}
	if extra_headers != nil {
		req.appendHeaders(extra_headers)
	}
	if s.dlg_headers != nil {
		req.appendHeaders(s.dlg_headers)
	}
	s.reqs[s.lCSeq] = req
	s.lCSeq++
	return req, nil
}

func (s *Ua) GetUasResp() sippy_types.SipResponse {
	return s.uasResp
}

func (s *Ua) SetUasResp(resp sippy_types.SipResponse) {
	s.uasResp = resp
}

func (s *Ua) SendUasResponse(t sippy_types.ServerTransaction, scode int, reason string, body sippy_types.MsgBody /*= nil*/, contacts []*sippy_header.SipContact /*= nil*/, ack_wait bool, extra_headers ...sippy_header.SipHeader) {
	uasResp := s.uasResp.GetCopy()
	uasResp.SetSCode(scode, reason)
	uasResp.SetBody(body)
	if contacts != nil {
		for _, contact := range contacts {
			uasResp.AppendHeader(contact)
		}
	}
	for _, eh := range extra_headers {
		uasResp.AppendHeader(eh)
	}
	var ack_cb func(sippy_types.SipRequest)
	if ack_wait {
		ack_cb = s.me().RecvACK
	}
	if t != nil {
		t.SendResponseWithLossEmul(uasResp /*retrans*/, false, ack_cb, s.uas_lossemul)
	} else {
		// the lock on the server transaction is already aquired so find it but do not try to lock
		if sip_tm := s.get_sip_tm(); sip_tm != nil {
			sip_tm.SendResponseWithLossEmul(uasResp /*lock*/, false, ack_cb, s.uas_lossemul)
		}
	}
}

func (s *Ua) RecvACK(req sippy_types.SipRequest) {
	if !s.isConnected() {
		return
	}
	//print 'UA::recvACK', req
	s.state.RecvACK(req)
	s.emitPendingEvents()
}

func (s *Ua) IsYours(req sippy_types.SipRequest, br0k3n_to bool /*= False*/) bool {
	var via0 *sippy_header.SipViaBody
	var err error

	via0, err = req.GetVias()[0].GetBody()
	if err != nil {
		s.logError("UA::IsYours error #1: " + err.Error())
		return false
	}
	//print s.branch, req.getHFBody("via").getBranch()
	if req.GetMethod() != "BYE" && s.branch != "" && s.branch != via0.GetBranch() {
		return false
	}
	call_id := req.GetCallId().CallId
	//print str(s.cId), call_id
	if call_id != s.cId.CallId {
		return false
	}
	//print s.rUri.getTag(), from_tag
	if s.rUri != nil {
		var rUri *sippy_header.SipAddress
		var from_body *sippy_header.SipAddress

		from_body, err = req.GetFrom().GetBody(s.config)
		if err != nil {
			s.logError("UA::IsYours error #2: " + err.Error())
			return false
		}
		rUri, err = s.rUri.GetBody(s.config)
		if err != nil {
			s.logError("UA::IsYours error #4: " + err.Error())
			return false
		}
		if rUri.GetTag() != from_body.GetTag() {
			return false
		}
	}
	//print s.lUri.getTag(), to_tag
	if s.lUri != nil && !br0k3n_to {
		var lUri *sippy_header.SipAddress
		var to_body *sippy_header.SipAddress

		to_body, err = req.GetTo().GetBody(s.config)
		if err != nil {
			s.logError("UA::IsYours error #3: " + err.Error())
			return false
		}
		lUri, err = s.lUri.GetBody(s.config)
		if err != nil {
			s.logError("UA::IsYours error #5: " + err.Error())
			return false
		}
		if lUri.GetTag() != to_body.GetTag() {
			return false
		}
	}
	return true
}

func (s *Ua) DelayedRemoteSdpUpdate(event sippy_types.CCEvent, remote_sdp_body sippy_types.MsgBody) {
	s.rSDP = remote_sdp_body.GetCopy()
	s.me().Enqueue(event)
	s.emitPendingEvents()
}

func (s *Ua) update_ua(msg sippy_types.SipMsg) {
	if msg.GetSipUserAgent() != nil {
		s.remote_ua = msg.GetSipUserAgent().UserAgent
	} else if msg.GetSipServer() != nil {
		s.remote_ua = msg.GetSipServer().Server
	}
}

func (s *Ua) CancelCreditTimer() {
	//print("UA::cancelCreditTimer()")
	if s.credit_timer != nil {
		s.credit_timer.Cancel()
		s.credit_timer = nil
	}
}

func (s *Ua) StartCreditTimer(rtime *sippy_time.MonoTime) {
	//print("UA::startCreditTimer()")
	if s.credit_time != nil {
		s.credit_times[0] = rtime.Add(*s.credit_time)
		s.credit_time = nil
	}
	//credit_time = min([x for x in s.credit_times.values() if x != nil])
	var credit_time *sippy_time.MonoTime = nil
	for _, t := range s.credit_times {
		if credit_time == nil || (*credit_time).After(t) {
			credit_time = t
		}
	}
	if credit_time == nil {
		return
	}
	// TODO make use of the mono time properly
	now, _ := sippy_time.NewMonoTime()
	s.credit_timer = StartTimeout(func() { s.credit_expires(credit_time) }, s.session_lock, credit_time.Sub(now), 1, s.config.ErrorLogger())
}

func (s *Ua) UpdateRouting(resp sippy_types.SipResponse, update_rtarget bool /*true*/, reverse_routes bool /*true*/) {
	if update_rtarget && len(resp.GetContacts()) > 0 {
		contact, err := resp.GetContacts()[0].GetBody(s.config)
		if err != nil {
			s.logError("UA::UpdateRouting: error #1: " + err.Error())
			return
		}
		s.rTarget = contact.GetUrl().GetCopy()
	}
	s.routes = make([]*sippy_header.SipRoute, len(resp.GetRecordRoutes()))
	for i, r := range resp.GetRecordRoutes() {
		if reverse_routes {
			s.routes[len(resp.GetRecordRoutes())-i-1] = r.AsSipRoute()
		} else {
			s.routes[i] = r.AsSipRoute()
		}
	}
	if s.outbound_proxy != nil {
		s.routes = append([]*sippy_header.SipRoute{sippy_header.NewSipRoute(sippy_header.NewSipAddress("", sippy_header.NewSipURL("", s.outbound_proxy.Host, s.outbound_proxy.Port, true)))}, s.routes...)
	}
	if len(s.routes) > 0 {
		r0, err := s.routes[0].GetBody(s.config)
		if err != nil {
			s.logError("UA::UpdateRouting: error #2: " + err.Error())
			return
		}
		if !r0.GetUrl().Lr {
			s.routes = append(s.routes, sippy_header.NewSipRoute( /*address*/ sippy_header.NewSipAddress("" /*url*/, s.rTarget)))
			s.rTarget = r0.GetUrl()
			s.routes = s.routes[1:]
			s.rAddr = s.rTarget.GetAddr(s.config)
		} else {
			s.rAddr = r0.GetUrl().GetAddr(s.config)
		}
	} else {
		s.rAddr = s.rTarget.GetAddr(s.config)
	}
}

func (s *Ua) GetSetupTs() *sippy_time.MonoTime {
	return s.setup_ts
}

func (s *Ua) SetSetupTs(ts *sippy_time.MonoTime) {
	s.setup_ts = ts
}

func (s *Ua) GetOrigin() string {
	return s.origin
}

func (s *Ua) SetOrigin(origin string) {
	s.origin = origin
}

func (s *Ua) OnLocalSdpChange(body sippy_types.MsgBody, cb func(sippy_types.MsgBody)) error {
	if s.on_local_sdp_change == nil {
		return nil
	}
	return s.on_local_sdp_change(body, cb)
}

func (s *Ua) HasOnLocalSdpChange() bool {
	return s.on_local_sdp_change != nil
}

func (s *Ua) SetCallId(call_id *sippy_header.SipCallId) {
	s.cId = call_id
}

func (s *Ua) GetCallId() *sippy_header.SipCallId {
	return s.cId
}

func (s *Ua) SetRTarget(url *sippy_header.SipURL) {
	s.rTarget = url
}

func (s *Ua) GetRAddr0() *sippy_net.HostPort {
	return s.rAddr0
}

func (s *Ua) SetRAddr0(addr *sippy_net.HostPort) {
	s.rAddr0 = addr
}

func (s *Ua) GetRTarget() *sippy_header.SipURL {
	return s.rTarget
}

func (s *Ua) SetRUri(ruri *sippy_header.SipTo) {
	s.rUri = ruri
}

func (s *Ua) GetRUri() *sippy_header.SipTo {
	return s.rUri
}

func (s *Ua) SetLUri(from *sippy_header.SipFrom) {
	s.lUri = from
}

func (s *Ua) GetLUri() *sippy_header.SipFrom {
	return s.lUri
}

func (s *Ua) GetLTag() string {
	return s.ltag
}

func (s *Ua) SetLCSeq(cseq int) {
	s.lCSeq = cseq
}

func (s *Ua) GetLContact() *sippy_header.SipContact {
	return s.lContact
}

func (s *Ua) GetLContacts() []*sippy_header.SipContact {
	contact := s.lContact // copy the value into a local variable for thread safety
	if contact == nil {
		return nil
	}
	return []*sippy_header.SipContact{contact}
}

func (s *Ua) SetLContact(contact *sippy_header.SipContact) {
	s.lContact = contact
}

func (s *Ua) SetRoutes(routes []*sippy_header.SipRoute) {
	s.routes = routes
}

func (s *Ua) GetLSDP() sippy_types.MsgBody {
	return s.lSDP
}

func (s *Ua) SetLSDP(msg sippy_types.MsgBody) {
	s.lSDP = msg
}

func (s *Ua) GetRSDP() sippy_types.MsgBody {
	return s.rSDP
}

func (s *Ua) SetRSDP(sdp sippy_types.MsgBody) {
	s.rSDP = sdp
}

func (s *Ua) GetSourceAddress() *sippy_net.HostPort {
	return s.source_address
}

func (s *Ua) SetSourceAddress(addr *sippy_net.HostPort) {
	s.source_address = addr
}

func (s *Ua) SetClientTransaction(tr sippy_types.ClientTransaction) {
	s.tr = tr
}

func (s *Ua) GetClientTransaction() sippy_types.ClientTransaction {
	return s.tr
}

func (s *Ua) GetOutboundProxy() *sippy_net.HostPort {
	return s.outbound_proxy
}

func (s *Ua) SetOutboundProxy(outbound_proxy *sippy_net.HostPort) {
	s.outbound_proxy = outbound_proxy
}

func (s *Ua) GetNoReplyTime() time.Duration {
	return s.no_reply_time
}

func (s *Ua) SetNoReplyTime(no_reply_time time.Duration) {
	s.no_reply_time = no_reply_time
}

func (s *Ua) GetExpireTime() time.Duration {
	return s.expire_time
}

func (s *Ua) SetExpireTime(expire_time time.Duration) {
	s.expire_time = expire_time
}

func (s *Ua) GetNoProgressTime() time.Duration {
	return s.no_progress_time
}

func (s *Ua) SetNoProgressTime(no_progress_time time.Duration) {
	s.no_progress_time = no_progress_time
}

func (s *Ua) StartNoReplyTimer() {
	now, _ := sippy_time.NewMonoTime()
	s.no_reply_timer = StartTimeout(s.no_reply_expires, s.session_lock, s._nr_mtime.Sub(now), 1, s.config.ErrorLogger())
}

func (s *Ua) StartNoProgressTimer() {
	now, _ := sippy_time.NewMonoTime()
	s.no_progress_timer = StartTimeout(s.no_progress_expires, s.session_lock, s._np_mtime.Sub(now), 1, s.config.ErrorLogger())
}

func (s *Ua) StartExpireTimer(start *sippy_time.MonoTime) {
	var d time.Duration
	now, _ := sippy_time.NewMonoTime()
	if s.expire_starts_on_setup {
		d = s._ex_mtime.Sub(now)
	} else {
		d = s.expire_time - now.Sub(start)
	}
	s.expire_timer = StartTimeout(s.expires, s.session_lock, d, 1, s.config.ErrorLogger())
}

func (s *Ua) CancelExpireTimer() {
	if s.expire_timer != nil {
		s.expire_timer.Cancel()
		s.expire_timer = nil
	}
}

func (s *Ua) GetDisconnectTs() *sippy_time.MonoTime {
	return s.disconnect_ts
}

func (s *Ua) SetDisconnectTs(ts *sippy_time.MonoTime) {
	if s.connect_ts != nil && s.connect_ts.After(ts) {
		s.disconnect_ts = s.connect_ts
	} else {
		s.disconnect_ts = ts
	}
}

func (s *Ua) DiscCb(rtime *sippy_time.MonoTime, origin string, scode int, inreq sippy_types.SipRequest) {
	if disc_cb := s.disc_cb; disc_cb != nil {
		disc_cb(rtime, origin, scode, inreq)
	}
}

func (s *Ua) GetDiscCb() sippy_types.OnDisconnectListener {
	return s.disc_cb
}

func (s *Ua) SetDiscCb(disc_cb sippy_types.OnDisconnectListener) {
	s.disc_cb = disc_cb
}

func (s *Ua) FailCb(rtime *sippy_time.MonoTime, origin string, scode int) {
	if fail_cb := s.fail_cb; fail_cb != nil {
		fail_cb(rtime, origin, scode)
	}
}

func (s *Ua) GetFailCb() sippy_types.OnFailureListener {
	return s.fail_cb
}

func (s *Ua) SetFailCb(fail_cb sippy_types.OnFailureListener) {
	s.fail_cb = fail_cb
}

func (s *Ua) GetDeadCb() sippy_types.OnDeadListener {
	return s.dead_cb
}

func (s *Ua) SetDeadCb(dead_cb sippy_types.OnDeadListener) {
	s.dead_cb = dead_cb
}

func (s *Ua) GetRAddr() *sippy_net.HostPort {
	return s.rAddr
}

func (s *Ua) SetRAddr(addr *sippy_net.HostPort) {
	s.rAddr = addr
}

func (s *Ua) OnDead() {
	s.sip_tm_lock.Lock()
	defer s.sip_tm_lock.Unlock()
	if s.sip_tm == nil {
		return
	}
	if s.cId != nil {
		s.sip_tm.UnregConsumer(s.me(), s.cId.CallId)
	}
	s.tr = nil
	s.call_controller = nil
	s.conn_cb = nil
	s.fail_cb = nil
	s.ring_cb = nil
	s.disc_cb = nil
	s.on_local_sdp_change = nil
	s.on_remote_sdp_change = nil
	s.expire_timer = nil
	s.no_progress_timer = nil
	s.credit_timer = nil
	// Keep this at the very end of processing
	if s.dead_cb != nil {
		s.dead_cb()
	}
	s.dead_cb = nil
	s.sip_tm = nil
}

func (s *Ua) GetLocalUA() *sippy_header.SipUserAgent {
	return s.local_ua
}

func (s *Ua) SetLocalUA(ua *sippy_header.SipUserAgent) {
	s.local_ua = ua
}

func (s *Ua) Enqueue(event sippy_types.CCEvent) {
	s.equeue = append(s.equeue, event)
}

func (s *Ua) OnRemoteSdpChange(body sippy_types.MsgBody, f func(x sippy_types.MsgBody)) error {
	if s.on_remote_sdp_change != nil {
		return s.on_remote_sdp_change(body, f)
	}
	return nil
}

func (s *Ua) ShouldUseRefer() bool {
	return s.useRefer
}

func (s *Ua) GetStateName() string {
	if state := s.state; state != nil {
		return state.String()
	}
	return "None"
}

func (s *Ua) GetState() sippy_types.UaStateID {
	if state := s.state; state != nil {
		return state.ID()
	}
	return sippy_types.UA_STATE_NONE
}

func (s *Ua) GetUsername() string {
	return s.username
}

func (s *Ua) SetUsername(username string) {
	s.username = username
}

func (s *Ua) GetPassword() string {
	return s.password
}

func (s *Ua) SetPassword(passwd string) {
	s.password = passwd
}

func (s *Ua) GetKaInterval() time.Duration {
	return s.kaInterval
}

func (s *Ua) SetKaInterval(ka time.Duration) {
	s.kaInterval = ka
}

func (s *Ua) ResetOnLocalSdpChange() {
	s.on_local_sdp_change = nil
}

func (s *Ua) GetOnLocalSdpChange() sippy_types.OnLocalSdpChange {
	return s.on_local_sdp_change
}

func (s *Ua) SetOnLocalSdpChange(on_local_sdp_change sippy_types.OnLocalSdpChange) {
	s.on_local_sdp_change = on_local_sdp_change
}

func (s *Ua) GetOnRemoteSdpChange() sippy_types.OnRemoteSdpChange {
	return s.on_remote_sdp_change
}

func (s *Ua) SetOnRemoteSdpChange(on_remote_sdp_change sippy_types.OnRemoteSdpChange) {
	s.on_remote_sdp_change = on_remote_sdp_change
}

func (s *Ua) ResetOnRemoteSdpChange() {
	s.on_remote_sdp_change = nil
}

func (s *Ua) GetGoDeadTimeout() time.Duration {
	return s.godead_timeout
}

func (s *Ua) GetLastScode() int {
	return s.last_scode
}

func (s *Ua) SetLastScode(scode int) {
	s.last_scode = scode
}

func (s *Ua) HasNoReplyTimer() bool {
	return s.no_reply_timer != nil
}

func (s *Ua) CancelNoReplyTimer() {
	if s.no_reply_timer != nil {
		s.no_reply_timer.Cancel()
		s.no_reply_timer = nil
	}
}

func (s *Ua) GetNpMtime() *sippy_time.MonoTime {
	return s._np_mtime
}

func (s *Ua) GetExMtime() *sippy_time.MonoTime {
	return s._ex_mtime
}

func (s *Ua) SetExMtime(t *sippy_time.MonoTime) {
	s._ex_mtime = t
}

func (s *Ua) GetP100Ts() *sippy_time.MonoTime {
	return s.p100_ts
}

func (s *Ua) SetP100Ts(ts *sippy_time.MonoTime) {
	if s.p100_ts == nil {
		s.p100_ts = ts
	}
}

func (s *Ua) HasNoProgressTimer() bool {
	return s.no_progress_timer != nil
}

func (s *Ua) CancelNoProgressTimer() {
	if s.no_progress_timer != nil {
		s.no_progress_timer.Cancel()
		s.no_progress_timer = nil
	}
}

func (s *Ua) HasOnRemoteSdpChange() bool {
	return s.on_remote_sdp_change != nil
}

func (s *Ua) GetP1xxTs() *sippy_time.MonoTime {
	return s.p1xx_ts
}

func (s *Ua) SetP1xxTs(ts *sippy_time.MonoTime) {
	s.p1xx_ts = ts
}

func (s *Ua) RingCb(rtime *sippy_time.MonoTime, origin string, scode int) {
	if ring_cb := s.ring_cb; ring_cb != nil {
		ring_cb(rtime, origin, scode)
	}
}

func (s *Ua) GetConnectTs() *sippy_time.MonoTime {
	return s.connect_ts
}

func (s *Ua) SetConnectTs(connect_ts *sippy_time.MonoTime) {
	if s.connect_ts == nil {
		if s.disconnect_ts != nil && connect_ts.After(s.disconnect_ts) {
			s.connect_ts = s.disconnect_ts
		} else {
			s.connect_ts = connect_ts
		}
	}
}

func (s *Ua) SetBranch(branch string) {
	s.branch = branch
}

func (s *Ua) ConnCb(rtime *sippy_time.MonoTime, origin string) {
	if conn_cb := s.conn_cb; conn_cb != nil {
		conn_cb(rtime, origin)
	}
}

func (s *Ua) GetConnCb() sippy_types.OnConnectListener {
	return s.conn_cb
}

func (s *Ua) SetConnCb(conn_cb sippy_types.OnConnectListener) {
	s.conn_cb = conn_cb
}

func (s *Ua) SetAuth(auth sippy_header.SipHeader) {
	s.auth = auth
}

func (s *Ua) SetNpMtime(t *sippy_time.MonoTime) {
	s._np_mtime = t
}

func (s *Ua) GetNrMtime() *sippy_time.MonoTime {
	return s._nr_mtime
}

func (s *Ua) SetNrMtime(t *sippy_time.MonoTime) {
	s._nr_mtime = t
}

func (s *Ua) logError(args ...interface{}) {
	s.config.ErrorLogger().Error(args...)
}

func (s *Ua) GetController() sippy_types.CallController {
	return s.call_controller
}

func (s *Ua) SetCreditTime(credit_time time.Duration) {
	s.credit_time = &credit_time
}

func (s *Ua) GetSessionLock() sync.Locker {
	return s.session_lock
}

func (s *Ua) isConnected() bool {
	if s.state != nil {
		return s.state.IsConnected()
	}
	return false
}

func (s *Ua) GetPendingTr() sippy_types.ClientTransaction {
	return s.pending_tr
}

func (s *Ua) SetPendingTr(tr sippy_types.ClientTransaction) {
	s.pending_tr = tr
}

func (s *Ua) GetLateMedia() bool {
	return s.late_media
}

func (s *Ua) SetLateMedia(late_media bool) {
	s.late_media = late_media
}

func (s *Ua) GetPassAuth() bool {
	return s.pass_auth
}

func (s *Ua) GetRemoteUA() string {
	return s.remote_ua
}

func (s *Ua) ResetCreditTime(rtime *sippy_time.MonoTime, new_credit_times map[int64]*sippy_time.MonoTime) {
	for k, v := range new_credit_times {
		s.credit_times[k] = v
	}
	if s.state.IsConnected() {
		s.me().CancelCreditTimer()
		s.me().StartCreditTimer(rtime)
	}
}

func (s *Ua) GetExtraHeaders() []sippy_header.SipHeader {
	return s.extra_headers
}

func (s *Ua) SetExtraHeaders(extra_headers []sippy_header.SipHeader) {
	s.extra_headers = extra_headers
}

func (s *Ua) OnUnregister() {
}

func (s *Ua) GetAcct(disconnect_ts *sippy_time.MonoTime) (duration time.Duration, delay time.Duration, connected bool, disconnected bool) {
	if s.disconnect_ts != nil {
		disconnect_ts = s.disconnect_ts
		disconnected = true
	} else {
		if disconnect_ts == nil {
			disconnect_ts, _ = sippy_time.NewMonoTime()
		}
		disconnected = false
	}
	if s.connect_ts != nil {
		duration = disconnect_ts.Sub(s.connect_ts)
		delay = s.connect_ts.Sub(s.setup_ts)
		connected = true
		return
	}
	duration = 0
	delay = disconnect_ts.Sub(s.setup_ts)
	connected = false
	return
}

func (s *Ua) GetCLD() string {
	if s.rUri == nil {
		return ""
	}
	rUri, err := s.rUri.GetBody(s.config)
	if err != nil {
		s.logError("UA::GetCLD: " + err.Error())
		return ""
	}
	return rUri.GetUrl().Username
}

func (s *Ua) GetCLI() string {
	if s.lUri == nil {
		return ""
	}
	lUri, err := s.lUri.GetBody(s.config)
	if err != nil {
		s.logError("UA::GetCLI: " + err.Error())
		return ""
	}
	return lUri.GetUrl().Username
}

func (s *Ua) GetUasLossEmul() int {
	return 0
}

func (s *Ua) Config() sippy_conf.Config {
	return s.config
}

func (s *Ua) BeforeResponseSent(sippy_types.SipResponse) {
}

func (s *Ua) BeforeRequestSent(sippy_types.SipRequest) {
}

func (s *Ua) OnUacSetupComplete() {
	if s.on_uac_setup_complete != nil {
		s.on_uac_setup_complete()
	}
}

func (s *Ua) SetOnUacSetupComplete(fn func()) {
	s.on_uac_setup_complete = fn
}

func (s *Ua) Cleanup() {
}

func (s *Ua) OnEarlyUasDisconnect(ev sippy_types.CCEvent) (int, string) {
	return 500, "Disconnected"
}

func (s *Ua) SetExpireStartsOnSetup(v bool) {
	s.expire_starts_on_setup = v
}

func (s *Ua) RecvPRACK(req sippy_types.SipRequest, resp sippy_types.SipResponse) {
	state := s.state
	if state != nil {
		state.RecvPRACK(req, resp)
	}
}

func (s *Ua) PrRel() bool {
	return s.pr_rel
}

func (s *Ua) processProxyChallenge(resp sippy_types.SipResponse, cseq int, orig_req sippy_types.SipRequest, eh []sippy_header.SipHeader) bool {
	if s.username == "" || s.password == "" || orig_req.GetSipProxyAuthorization() != nil {
		return false
	}
	auths := resp.GetSipProxyAuthenticates()
	challenges := make([]sippy_types.Challenge, len(auths))
	for i, hdr := range auths {
		challenges[i] = hdr
	}
	return s.processChallenge(challenges, cseq, eh)
}

func (s *Ua) processWWWChallenge(resp sippy_types.SipResponse, cseq int, orig_req sippy_types.SipRequest, eh []sippy_header.SipHeader) bool {
	if s.username == "" || s.password == "" || orig_req.GetSipAuthorization() != nil {
		return false
	}
	auths := resp.GetSipWWWAuthenticates()
	challenges := make([]sippy_types.Challenge, len(auths))
	for i, hdr := range auths {
		challenges[i] = hdr
	}
	return s.processChallenge(challenges, cseq, eh)
}

func (s *Ua) processChallenge(challenges []sippy_types.Challenge, cseq int, eh []sippy_header.SipHeader) bool {
	var challenge sippy_types.Challenge
	found := false
	for _, challenge = range challenges {
		algorithm, err := challenge.Algorithm()
		if err != nil {
			s.logError("UA::processChallenge: cannot get algorithm: " + err.Error())
			return false
		}
		if s.auth_enalgs != nil {
			if _, ok := s.auth_enalgs[algorithm]; !ok {
				continue
			}
		}
		supported, err := challenge.SupportedAlgorithm()
		if err == nil && supported {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	if challenge == nil {
		// no supported challenge has been found
		return false
	}
	req, err := s.GenRequest("INVITE", s.lSDP, challenge, eh...)
	if err != nil {
		s.logError("UA::processChallenge: cannot create INVITE: " + err.Error())
		return false
	}
	s.lCSeq += 1
	s.tr, err = s.me().PrepTr(req, eh)
	if err != nil {
		s.logError("UA::processChallenge: cannot prepare client transaction: " + err.Error())
		return false
	}
	s.tr.SetTxnHeaders(s.dlg_headers)
	s.BeginClientTransaction(req, s.tr)
	delete(s.reqs, cseq)
	return true
}

func (s *Ua) PassAuth() bool {
	return s.pass_auth
}

func (s *Ua) get_sip_tm() sippy_types.SipTransactionManager {
	s.sip_tm_lock.RLock()
	defer s.sip_tm_lock.RUnlock()
	return s.sip_tm
}

func (s *Ua) BeginClientTransaction(req sippy_types.SipRequest, tr sippy_types.ClientTransaction) {
	sip_tm := s.get_sip_tm()
	if sip_tm == nil {
		return
	}
	sip_tm.BeginClientTransaction(req, tr)
}

func (s *Ua) BeginNewClientTransaction(req sippy_types.SipRequest, resp_receiver sippy_types.ResponseReceiver) {
	sip_tm := s.get_sip_tm()
	if sip_tm == nil {
		return
	}
	sip_tm.BeginNewClientTransaction(req, resp_receiver, s.session_lock, s.source_address, nil /*userv*/, s.me().BeforeRequestSent)
}

func (s *Ua) RegConsumer(consumer sippy_types.UA, call_id string) {
	sip_tm := s.get_sip_tm()
	if sip_tm == nil {
		return
	}
	sip_tm.RegConsumer(consumer, call_id)
}

func (s *Ua) GetDlgHeaders() []sippy_header.SipHeader {
	return s.dlg_headers
}

func (s *Ua) SetDlgHeaders(hdrs []sippy_header.SipHeader) {
	s.dlg_headers = hdrs
}

func (s *Ua) OnReinvite(req sippy_types.SipRequest, event_update sippy_types.CCEvent) {
}
