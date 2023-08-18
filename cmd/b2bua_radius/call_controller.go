package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type callController struct {
	id                int64
	uaA               sippy_types.UA
	uaO               sippy_types.UA
	global_config     *myConfigParser
	state             CCState
	remote_ip         *sippy_net.MyAddress
	source            *sippy_net.HostPort
	routes            []*B2BRoute
	pass_headers      []sippy_header.SipHeader
	lock              *sync.Mutex // this must be a reference to prevent memory leak
	cId               *sippy_header.SipCallId
	cGUID             *sippy_header.SipCiscoGUID
	cli               string
	cld               string
	caller_name       string
	rtp_proxy_session *sippy.Rtp_proxy_session
	eTry              *sippy.CCEventTry
	huntstop_scodes   []int
	acctA             *fakeAccounting
	sip_tm            sippy_types.SipTransactionManager
	proxied           bool
	sdp_session       *sippy.SdpSession
	cmap              *CallMap
}

/*
class CallController(object):

	cld = nil
	acctA = nil
	acctO = nil
	global_config = nil
	rtp_proxy_session = nil
	huntstop_scodes = nil
	auth_proc = nil
	challenge = nil
*/
func NewCallController(id int64, remote_ip *sippy_net.MyAddress, source *sippy_net.HostPort, global_config *myConfigParser,
	pass_headers []sippy_header.SipHeader, sip_tm sippy_types.SipTransactionManager, cguid *sippy_header.SipCiscoGUID,
	cmap *CallMap) *callController {
	s := &callController{
		id:              id,
		global_config:   global_config,
		state:           CCStateIdle,
		remote_ip:       remote_ip,
		source:          source,
		routes:          make([]*B2BRoute, 0),
		pass_headers:    pass_headers,
		lock:            new(sync.Mutex),
		huntstop_scodes: make([]int, 0),
		proxied:         false,
		sip_tm:          sip_tm,
		sdp_session:     sippy.NewSdpSession(),
		cGUID:           cguid,
		cmap:            cmap,
	}
	s.uaA = sippy.NewUA(sip_tm, global_config, nil, s, s.lock, nil)
	s.uaA.SetKaInterval(s.global_config.keepalive_ans)
	s.uaA.SetLocalUA(sippy_header.NewSipUserAgent(s.global_config.GetMyUAName()))
	s.uaA.SetConnCb(s.aConn)
	s.uaA.SetDiscCb(s.aDisc)
	s.uaA.SetFailCb(s.aFail)
	s.uaA.SetDeadCb(s.aDead)
	return s
}

func (s *callController) RecvEvent(event sippy_types.CCEvent, ua sippy_types.UA) {
	if ua == s.uaA {
		if s.state == CCStateIdle {
			ev_try, ok := event.(*sippy.CCEventTry)
			if !ok {
				// Some weird event received
				s.uaA.RecvEvent(sippy.NewCCEventDisconnect(nil, event.GetRtime(), ""))
				return
			}
			s.cId = ev_try.GetSipCallId()
			s.cli = ev_try.GetCLI()
			s.cld = ev_try.GetCLD()
			//, body, auth,
			s.caller_name = ev_try.GetCallerName()
			if s.cld == "" {
				s.uaA.RecvEvent(sippy.NewCCEventFail(500, "Internal Server Error (1)", event.GetRtime(), ""))
				s.state = CCStateDead
				return
			}
			/*
			   if body != nil && s.global_config.has_key('_allowed_pts') {
			       try:
			           body.parse()
			       except:
			           s.uaA.RecvEvent(CCEventFail((400, "Malformed SDP Body"), rtime = event.rtime))
			           s.state = CCStateDead
			           return
			       allowed_pts = s.global_config['_allowed_pts']
			       mbody = body.content.sections[0].m_header
			       if mbody.transport.lower() == "rtp/avp" {
			           old_len = len(mbody.formats)
			           mbody.formats = [x for x in mbody.formats if x in allowed_pts]
			           if len(mbody.formats) == 0 {
			               s.uaA.RecvEvent(CCEventFail((488, "Not Acceptable Here")))
			               s.state = CCStateDead
			               return
			           if old_len > len(mbody.formats) {
			               body.content.sections[0].optimize_a()
			*/
			if strings.HasPrefix(s.cld, "nat-") {
				s.cld = s.cld[4:]
				if ev_try.GetBody() != nil {
					ev_try.GetBody().AppendAHeader("nated:yes")
				}
				event, _ = sippy.NewCCEventTry(s.cId, s.cli, s.cld, ev_try.GetBody(), ev_try.GetSipAuthorizationHF(), s.caller_name, nil, "")
			}
			/*
			   if s.global_config.has_key('static_tr_in') {
			       s.cld = re_replace(s.global_config['static_tr_in'], s.cld)
			       event = sippy.NewCCEventTry(s.cId, s.cGUID, s.cli, s.cld, body, auth, s.caller_name)
			   }
			*/
			if len(s.cmap.rtp_proxy_clients) > 0 {
				var err error
				s.rtp_proxy_session, err = sippy.NewRtp_proxy_session(s.global_config, s.cmap.rtp_proxy_clients, s.cId.CallId, "", "", s.global_config.B2bua_socket /*notify_tag*/, fmt.Sprintf("r%%20%d", s.id), s.lock)
				if err != nil {
					s.uaA.RecvEvent(sippy.NewCCEventFail(500, "Internal Server Error (4)", event.GetRtime(), ""))
					s.state = CCStateDead
					return
				}
				s.rtp_proxy_session.SetCalleeRaddress(sippy_net.NewHostPort(s.remote_ip.String(), "5060"))
				s.rtp_proxy_session.SetInsertNortpp(true)
			}
			s.eTry = ev_try
			s.state = CCStateWaitRoute
			//if ! s.global_config['auth_enable'] {
			//s.username = s.remote_ip
			s.rDone()
			//} else if auth == nil || auth.username == nil || len(auth.username) == 0 {
			//    s.username = s.remote_ip
			//    s.auth_proc = s.global_config['_radius_client'].do_auth(s.remote_ip, s.cli, s.cld, s.cGUID, \
			//      s.cId, s.remote_ip, s.rDone)
			//} else {
			//    s.username = auth.username
			//    s.auth_proc = s.global_config['_radius_client'].do_auth(auth.username, s.cli, s.cld, s.cGUID,
			//      s.cId, s.remote_ip, s.rDone, auth.realm, auth.nonce, auth.uri, auth.response)
			//}
			return
		}
		if (s.state != CCStateARComplete && s.state != CCStateConnected && s.state != CCStateDisconnecting) || s.uaO == nil {
			return
		}
		s.uaO.RecvEvent(event)
	} else {
		ev_fail, is_ev_fail := event.(*sippy.CCEventFail)
		_, is_ev_disconnect := event.(*sippy.CCEventFail)
		if (is_ev_fail || is_ev_disconnect) && s.state == CCStateARComplete &&
			(s.uaA.GetState() == sippy_types.UAS_STATE_TRYING ||
				s.uaA.GetState() == sippy_types.UAS_STATE_RINGING) && len(s.routes) > 0 {
			huntstop := false
			if is_ev_fail {
				for _, c := range s.huntstop_scodes {
					if c == ev_fail.GetScode() {
						huntstop = true
						break
					}
				}
			}
			if !huntstop {
				route := s.routes[0]
				s.routes = s.routes[1:]
				s.placeOriginate(route)
				return
			}
		}
		s.sdp_session.FixupVersion(event.GetBody())
		s.uaA.RecvEvent(event)
	}
}

func (s *callController) rDone( /*results*/ ) {
	/*
	   // Check that we got necessary result from Radius
	   if len(results) != 2 || results[1] != 0:
	       if isinstance(s.uaA.state, UasStateTrying):
	           if s.challenge != nil:
	               event = CCEventFail((401, "Unauthorized"))
	               event.extra_header = s.challenge
	           else:
	               event = CCEventFail((403, "Auth Failed"))
	           s.uaA.RecvEvent(event)
	           s.state = CCStateDead
	       return
	   if s.global_config['acct_enable']:
	       s.acctA = RadiusAccounting(s.global_config, "answer", \
	         send_start = s.global_config['start_acct_enable'], lperiod = \
	         s.global_config.getdefault('alive_acct_int', nil))
	       s.acctA.ms_precision = s.global_config.getdefault('precise_acct', false)
	       s.acctA.setParams(s.username, s.cli, s.cld, s.cGUID, s.cId, s.remote_ip)
	   else:
	*/
	s.acctA = NewFakeAccounting()
	// Check that uaA is still in a valid state, send acct stop
	if s.uaA.GetState() != sippy_types.UAS_STATE_TRYING {
		//s.acctA.disc(s.uaA, time(), "caller")
		return
	}
	/*
	   cli = [x[1][4:] for x in results[0] if x[0] == "h323-ivr-in" && x[1].startswith("CLI:")]
	   if len(cli) > 0:
	       s.cli = cli[0]
	       if len(s.cli) == 0:
	           s.cli = nil
	   caller_name = [x[1][5:] for x in results[0] if x[0] == "h323-ivr-in" && x[1].startswith("CNAM:")]
	   if len(caller_name) > 0:
	       s.caller_name = caller_name[0]
	       if len(s.caller_name) == 0:
	           s.caller_name = nil
	   credit_time = [x for x in results[0] if x[0] == "h323-credit-time"]
	   if len(credit_time) > 0:
	       credit_time = int(credit_time[0][1])
	   else:
	       credit_time := time.Duration(0)
	   if ! s.global_config.has_key('_static_route'):
	       routing = [x for x in results[0] if x[0] == "h323-ivr-in" && x[1].startswith("Routing:")]
	       if len(routing) == 0:
	           s.uaA.RecvEvent(CCEventFail((500, "Internal Server Error (2)")))
	           s.state = CCStateDead
	           return
	       routing = [B2BRoute(x[1][8:]) for x in routing]
	   else {
	*/
	routing := []*B2BRoute{s.cmap.static_route.getCopy()}
	//    }
	rnum := 0
	for _, oroute := range routing {
		rnum += 1
		oroute.customize(rnum, s.cld, s.cli, 0, s.pass_headers, 0)
		//oroute.customize(rnum, s.cld, s.cli, credit_time, s.pass_headers, s.global_config.max_credit_time)
		//if oroute.credit_time == 0 || oroute.expires == 0 {
		//    continue
		//}
		s.routes = append(s.routes, oroute)
		//println "Got route:", oroute.hostPort, oroute.cld
	}
	if len(s.routes) == 0 {
		s.uaA.RecvEvent(sippy.NewCCEventFail(500, "Internal Server Error (3)", nil, ""))
		s.state = CCStateDead
		return
	}
	s.state = CCStateARComplete
	route := s.routes[0]
	s.routes = s.routes[1:]
	s.placeOriginate(route)
}

func (s *callController) placeOriginate(oroute *B2BRoute) {
	//cId, cGUID, cli, cld, body, auth, caller_name = s.eTry.getData()
	cld := oroute.cld
	s.huntstop_scodes = oroute.huntstop_scodes
	//if s.global_config.has_key('static_tr_out') {
	//    cld = re_replace(s.global_config['static_tr_out'], cld)
	//}
	var nh_address *sippy_net.HostPort
	if oroute.hostPort == "sip-ua" {
		//host = s.source[0]
		nh_address = s.source
	} else {
		//host = oroute.hostonly
		nh_address = oroute.getNHAddr(s.source)
	}
	//if ! oroute.forward_on_fail && s.global_config['acct_enable'] {
	//    s.acctO = RadiusAccounting(s.global_config, "originate",
	//      send_start = s.global_config['start_acct_enable'], /*lperiod*/
	//      s.global_config.getdefault('alive_acct_int', nil))
	//    s.acctO.ms_precision = s.global_config.getdefault('precise_acct', false)
	//    s.acctO.setParams(oroute.params.get('bill-to', s.username), oroute.params.get('bill-cli', oroute.cli), \
	//      oroute.params.get('bill-cld', cld), s.cGUID, s.cId, host)
	//else {
	//    s.acctO = nil
	//}
	//s.acctA.credit_time = oroute.credit_time
	//disc_handlers = []
	//if ! oroute.forward_on_fail && s.global_config['acct_enable'] {
	//    disc_handlers.append(s.acctO.disc)
	//}
	s.uaO = sippy.NewUA(s.sip_tm, s.global_config, nh_address, s, s.lock, nil)
	// oroute.user, oroute.passw, nh_address, oroute.credit_time,
	//  /*expire_time*/ oroute.expires, /*no_progress_time*/ oroute.no_progress_expires, /*extra_headers*/ oroute.extra_headers)
	//s.uaO.SetConnCbs([]sippy_types.OnConnectListener{ s.oConn })
	extra_headers := []sippy_header.SipHeader{s.cGUID, s.cGUID.AsH323ConfId()}
	extra_headers = append(extra_headers, oroute.extra_headers...)
	s.uaO.SetExtraHeaders(extra_headers)
	s.uaO.SetDeadCb(s.oDead)
	s.uaO.SetLocalUA(sippy_header.NewSipUserAgent(s.global_config.GetMyUAName()))
	if oroute.outbound_proxy != nil && s.source.String() != oroute.outbound_proxy.String() {
		s.uaO.SetOutboundProxy(oroute.outbound_proxy)
	}
	var body sippy_types.MsgBody
	if s.rtp_proxy_session != nil && oroute.rtpp {
		s.uaO.SetOnLocalSdpChange(s.rtp_proxy_session.OnCallerSdpChange)
		s.uaO.SetOnRemoteSdpChange(s.rtp_proxy_session.OnCalleeSdpChange)
		s.rtp_proxy_session.SetCallerRaddress(nh_address)
		if s.eTry.GetBody() != nil {
			body = s.eTry.GetBody().GetCopy()
		}
		s.proxied = true
	}
	s.uaO.SetKaInterval(s.global_config.keepalive_orig)
	//if oroute.params.has_key('group_timeout') {
	//    timeout, skipto = oroute.params['group_timeout']
	//    Timeout(s.group_expires, timeout, 1, skipto)
	//}
	//if s.global_config.getdefault('hide_call_id', false) {
	//    cId = SipCallId(md5(str(cId)).hexdigest() + ("-b2b_%d" % oroute.rnum))
	//} else {
	cId := sippy_header.NewSipCallIdFromString(s.eTry.GetSipCallId().CallId + fmt.Sprintf("-b2b_%d", oroute.rnum))
	//}
	caller_name := oroute.caller_name
	if caller_name == "" {
		caller_name = s.caller_name
	}
	event, _ := sippy.NewCCEventTry(cId, oroute.cli, cld, body, s.eTry.GetSipAuthorizationHF(), caller_name, nil, "")
	//if s.eTry.max_forwards != nil {
	//    event.max_forwards = s.eTry.max_forwards - 1
	//    if event.max_forwards <= 0 {
	//        s.uaA.RecvEvent(sippy.NewCCEventFail(483, "Too Many Hops", nil, ""))
	//        s.state = CCStateDead
	//        return
	//    }
	//}
	event.SetReason(s.eTry.GetReason())
	s.uaO.RecvEvent(event)
}

func (s *callController) disconnect(rtime *sippy_time.MonoTime) {
	s.uaA.Disconnect(rtime, "")
}

/*
def oConn(s, ua, rtime, origin):

	if s.acctO != nil:
	    s.acctO.conn(ua, rtime, origin)
*/
func (s *callController) aConn(rtime *sippy_time.MonoTime, origin string) {
	s.state = CCStateConnected
	//s.acctA.conn(rtime, origin)
}

func (s *callController) aFail(rtime *sippy_time.MonoTime, origin string, result int) {
	s.aDisc(rtime, origin, result, nil)
}

func (s *callController) aDisc(rtime *sippy_time.MonoTime, origin string, result int, inreq sippy_types.SipRequest) {
	//if s.state == CCStateWaitRoute && s.auth_proc != nil {
	//    s.auth_proc.cancel()
	//    s.auth_proc = nil
	//}
	if s.uaO != nil && s.state != CCStateDead {
		s.state = CCStateDisconnecting
	} else {
		s.state = CCStateDead
	}
	//if s.acctA != nil {
	//    s.acctA.disc(ua, rtime, origin, result)
	//}
	if s.rtp_proxy_session != nil {
		s.rtp_proxy_session.Delete()
		s.rtp_proxy_session = nil
	}
}

func (s *callController) aDead() {
	if s.uaO == nil || s.uaO.GetState() == sippy_types.UA_STATE_DEAD {
		if s.cmap.debug_mode {
			println("garbadge collecting", s)
		}
		s.acctA = nil
		//s.acctO = nil
		s.cmap.DropCC(s.id)
		s.cmap = nil
	}
}

func (s *callController) oDead() {
	if s.uaA.GetState() == sippy_types.UA_STATE_DEAD {
		if s.cmap.debug_mode {
			println("garbadge collecting", s)
		}
		s.acctA = nil
		//s.acctO = nil
		s.cmap.DropCC(s.id)
	}
}

/*
   def group_expires(s, skipto):
       if s.state != CCStateARComplete || len(s.routes) == 0 || s.routes[0][0] > skipto || \
         (! isinstance(s.uaA.state, UasStateTrying) && ! isinstance(s.uaA.state, UasStateRinging)):
           return
       // When the last group in the list has timeouted don't disconnect
       // the current attempt forcefully. Instead, make sure that if the
       // current originate call leg fails no more routes will be
       // processed.
       if skipto == s.routes[-1][0] + 1:
           s.routes = []
           return
       while s.routes[0][0] != skipto:
           s.routes.pop(0)
       s.uaO.disconnect()
*/
