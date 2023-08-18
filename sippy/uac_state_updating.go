package sippy

import (
	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/headers"
	"github.com/sippy/go-b2bua/sippy/types"
)

type UacStateUpdating struct {
	*uaStateGeneric
	triedauth bool
}

func NewUacStateUpdating(ua sippy_types.UA, config sippy_conf.Config) *UacStateUpdating {
	s := &UacStateUpdating{
		uaStateGeneric: newUaStateGeneric(ua, config),
		triedauth:      false,
	}
	s.connected = true
	return s
}

func (s *UacStateUpdating) String() string {
	return "Updating(UAC)"
}

func (s *UacStateUpdating) RecvRequest(req sippy_types.SipRequest, t sippy_types.ServerTransaction) (sippy_types.UaState, func()) {
	if req.GetMethod() == "INVITE" {
		t.SendResponse(req.GenResponse(491, "Request Pending", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
		return nil, nil
	} else if req.GetMethod() == "BYE" {
		s.ua.GetClientTransaction().Cancel()
		t.SendResponse(req.GenResponse(200, "OK", nil, s.ua.GetLocalUA().AsSipServer()), false, nil)
		//print "BYE received in the Updating state, going to the Disconnected state"
		event := NewCCEventDisconnect(nil, req.GetRtime(), s.ua.GetOrigin())
		event.SetReason(req.GetReason())
		s.ua.Enqueue(event)
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(req.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(req.GetRtime(), s.ua.GetOrigin(), 0, nil) }
	}
	//print "wrong request %s in the state Updating" % req.getMethod()
	return nil, nil
}

func (s *UacStateUpdating) RecvResponse(resp sippy_types.SipResponse, tr sippy_types.ClientTransaction) (sippy_types.UaState, func()) {
	var err error
	var event sippy_types.CCEvent

	body := resp.GetBody()
	code, reason := resp.GetSCode()
	if code < 200 {
		s.ua.Enqueue(NewCCEventRing(code, reason, body, resp.GetRtime(), s.ua.GetOrigin()))
		return nil, nil
	}
	if code >= 200 && code < 300 {
		if !s.ua.GetLateMedia() || body == nil {
			event = NewCCEventConnect(code, reason, body, resp.GetRtime(), s.ua.GetOrigin())
		} else {
			event = NewCCEventPreConnect(code, reason, body, resp.GetRtime(), s.ua.GetOrigin())
			tr.SetUAck(true)
			s.ua.SetPendingTr(tr)
		}
		if body != nil {
			if s.ua.HasOnRemoteSdpChange() {
				if err := s.ua.OnRemoteSdpChange(body, func(x sippy_types.MsgBody) { s.ua.DelayedRemoteSdpUpdate(event, x) }); err != nil {
					ev := NewCCEventFail(502, "Bad Gateway", event.GetRtime(), "")
					ev.SetWarning("Malformed SDP Body received from downstream: \"" + err.Error() + "\"")
					return s.updateFailed(ev)
				}
				return NewUaStateConnected(s.ua, s.config), nil
			} else {
				s.ua.SetRSDP(body.GetCopy())
			}
		} else {
			s.ua.SetRSDP(nil)
		}
		s.ua.Enqueue(event)
		return NewUaStateConnected(s.ua, s.config), nil
	}
	reason_rfc3326 := resp.GetReason()
	if (code == 301 || code == 302) && len(resp.GetContacts()) > 0 {
		var contact *sippy_header.SipAddress

		contact, err = resp.GetContacts()[0].GetBody(s.config)
		if err != nil {
			s.config.ErrorLogger().Error("UacStateUpdating::RecvResponse: #1: " + err.Error())
			return nil, nil
		}
		event = NewCCEventRedirect(code, reason, body,
			[]*sippy_header.SipAddress{contact.GetCopy()},
			resp.GetRtime(), s.ua.GetOrigin())
	} else if code == 300 && len(resp.GetContacts()) > 0 {
		urls := make([]*sippy_header.SipAddress, 0)
		for _, contact := range resp.GetContacts() {
			var cbody *sippy_header.SipAddress

			cbody, err = contact.GetBody(s.config)
			if err != nil {
				s.config.ErrorLogger().Error("UacStateUpdating::RecvResponse: #2: " + err.Error())
				return nil, nil
			}
			urls = append(urls, cbody.GetCopy())
		}
		event = NewCCEventRedirect(code, reason, body, urls, resp.GetRtime(), s.ua.GetOrigin())
	} else {
		s.ua.SetLSDP(nil)
		event = NewCCEventFail(code, reason, resp.GetRtime(), s.ua.GetOrigin())
		event.SetReason(reason_rfc3326)
	}
	if code == 408 || code == 481 {
		// (Call/Transaction Does Not Exist) or a 408 (Request Timeout), the
		// UAC SHOULD terminate the dialog.  A UAC SHOULD also terminate a
		// dialog if no response at all is received for the request (the
		// client transaction would inform the TU about the timeout.)
		return s.updateFailed(event)
	}
	s.ua.Enqueue(event)
	return NewUaStateConnected(s.ua, s.config), nil
}

func (s *UacStateUpdating) updateFailed(event sippy_types.CCEvent) (sippy_types.UaState, func()) {
	s.ua.Enqueue(event)
	eh := []sippy_header.SipHeader{}
	if event.GetReason() != nil {
		eh = append(eh, event.GetReason())
	}
	req, err := s.ua.GenRequest("BYE", nil, nil, eh...)
	if err != nil {
		s.config.ErrorLogger().Error("UacStateUpdating::updateFailed: #1: " + err.Error())
		return nil, nil
	}
	s.ua.BeginNewClientTransaction(req, nil)

	s.ua.CancelCreditTimer()
	s.ua.SetDisconnectTs(event.GetRtime())
	event = NewCCEventDisconnect(nil, event.GetRtime(), s.ua.GetOrigin())
	s.ua.Enqueue(event)
	return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), 0, nil) }
}

func (s *UacStateUpdating) RecvEvent(event sippy_types.CCEvent) (sippy_types.UaState, func(), error) {
	send_bye := false
	switch event.(type) {
	case *CCEventDisconnect:
		send_bye = true
	case *CCEventFail:
		send_bye = true
	case *CCEventRedirect:
		send_bye = true
	}
	if send_bye {
		s.ua.GetClientTransaction().Cancel()
		req, err := s.ua.GenRequest("BYE", nil, nil, event.GetExtraHeaders()...)
		if err != nil {
			return nil, nil, err
		}
		s.ua.BeginNewClientTransaction(req, nil)
		s.ua.CancelCreditTimer()
		s.ua.SetDisconnectTs(event.GetRtime())
		return NewUaStateDisconnected(s.ua, s.config), func() { s.ua.DiscCb(event.GetRtime(), event.GetOrigin(), 0, nil) }, nil
	}
	//return nil, fmt.Errorf("wrong event %s in the Updating state", event.String())
	return nil, nil, nil
}

func (s *UacStateUpdating) ID() sippy_types.UaStateID {
	return sippy_types.UAC_STATE_UPDATING
}
