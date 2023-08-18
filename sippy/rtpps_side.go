package sippy

import (
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/sdp"
	"github.com/sippy/go-b2bua/sippy/types"
)

type _rtpps_side struct {
	otherside        *_rtpps_side
	owner            *Rtp_proxy_session
	session_exists   bool
	laddress         string
	raddress         *sippy_net.HostPort
	codecs           string
	repacketize      int
	after_sdp_change func(sippy_types.RtpProxyUpdateResult)
	from_tag         string
	to_tag           string
}

func (s *_rtpps_side) _play(prompt_name string, times int, result_callback func(string), index int) {
	if !s.session_exists {
		return
	}
	if !s.otherside.session_exists {
		s.otherside.update("0.0.0.0", "0", func(*rtpproxy_update_result) { s.__play(prompt_name, times, result_callback, index) }, "", index, "IP4")
		return
	}
	s.__play(prompt_name, times, result_callback, index)
}

func (s *_rtpps_side) __play(prompt_name string, times int, result_callback func(string), index int) {
	command := "P" + strconv.Itoa(times) + " " + s.owner.call_id + "-" + strconv.Itoa(index) + " " + prompt_name + " " + s.codecs + " " + s.from_tag + " " + s.to_tag
	s.owner.send_command(command, func(r string) { s.owner.command_result(r, result_callback) })
}

func (s *_rtpps_side) update(remote_ip string, remote_port string, result_callback func(*rtpproxy_update_result), options /*= ""*/ string, index /*= 0*/ int, atype /*= "IP4"*/ string) {
	var sbind_supported, is_local, tnot_supported bool
	var err error

	command := "U"
	s.owner.max_index = int(math.Max(float64(s.owner.max_index), float64(index)))
	if sbind_supported, err = s.owner.SBindSupported(); err != nil {
		return
	}
	if is_local, err = s.owner.IsLocal(); err != nil {
		return
	}
	if tnot_supported, err = s.owner.TNotSupported(); err != nil {
		return
	}
	if sbind_supported {
		if s.raddress != nil {
			//if s.owner.IsLocal() && atype == "IP4" {
			//    options += "L" + s.laddress
			//} else if ! s.owner.IsLocal() {
			//    options += "R" + s.raddress.Host.String()
			//}
			options += "R" + s.raddress.Host.String()
		} else if s.laddress != "" && is_local {
			options += "L" + s.laddress
		}
	}
	command += options
	if s.otherside.session_exists {
		command += " " + s.owner.call_id + "-" + strconv.Itoa(index) + " " + remote_ip + " " + remote_port + " " + s.from_tag + " " + s.to_tag
	} else {
		command += " " + s.owner.call_id + "-" + strconv.Itoa(index) + " " + remote_ip + " " + remote_port + " " + s.from_tag
	}
	if s.owner.notify_socket != "" && index == 0 && tnot_supported {
		command += " " + s.owner.notify_socket + " " + s.owner.notify_tag
	}
	s.owner.send_command(command, func(r string) { s.update_result(r, remote_ip, atype, result_callback) })
}

func (s *_rtpps_side) update_result(result, remote_ip, atype string, result_callback func(*rtpproxy_update_result)) {
	//print "%s.update_result(%s)" % (id(s), result)
	//result_callback, face, callback_parameters = args
	s.session_exists = true
	if result == "" {
		result_callback(nil)
		return
	}
	t1 := strings.Fields(result)
	if t1[0][0] == 'E' {
		result_callback(nil)
		return
	}
	rtpproxy_port, err := strconv.Atoi(t1[0])
	if err != nil || rtpproxy_port == 0 {
		result_callback(nil)
		return
	}
	family := "IP4"
	rtpproxy_address := ""
	if len(t1) > 1 {
		rtpproxy_address = t1[1]
		if len(t1) > 2 && t1[2] == "6" {
			family = "IP6"
		}
	} else {
		if rtpproxy_address, err = s.owner.GetProxyAddress(); err != nil {
			return
		}
	}
	sendonly := false
	if atype == "IP4" && remote_ip == "0.0.0.0" {
		sendonly = true
	} else if atype == "IP6" && remote_ip == "::" {
		sendonly = true
	}
	result_callback(&rtpproxy_update_result{
		rtpproxy_address: rtpproxy_address,
		rtpproxy_port:    t1[0],
		family:           family,
		sendonly:         sendonly,
	})
}

func (s *_rtpps_side) _on_sdp_change(sdp_body sippy_types.MsgBody, result_callback func(sippy_types.MsgBody)) error {
	parsed_body, err := sdp_body.GetSdp()
	if err != nil {
		return err
	}
	sects := []*sippy_sdp.SdpMediaDescription{}
	for _, sect := range parsed_body.GetSections() {
		switch strings.ToLower(sect.GetMHeader().GetTransport()) {
		case "udp", "udptl", "rtp/avp", "rtp/savp", "udp/bfcp":
			sects = append(sects, sect)
		default:
		}
	}
	if len(sects) == 0 {
		sdp_body.SetNeedsUpdate(false)
		result_callback(sdp_body)
		return nil
	}
	formats := sects[0].GetMHeader().GetFormats()
	s.codecs = strings.Join(formats, ",")
	options := ""
	if s.repacketize > 0 {
		options = "z" + strconv.Itoa(s.repacketize)
	}
	sections_left := int64(len(sects))
	for i, sect := range sects {
		sect_options := options
		if sect.GetCHeader().GetAType() == "IP6" {
			sect_options = "6" + options
		}
		s.update(sect.GetCHeader().GetAddr(), sect.GetMHeader().GetPort(),
			func(res *rtpproxy_update_result) {
				s._sdp_change_finish(res, sdp_body, parsed_body, sect, &sections_left, result_callback)
			},
			sect_options, i, sect.GetCHeader().GetAType())
	}
	return nil
}

func (s *_rtpps_side) _sdp_change_finish(cb_args *rtpproxy_update_result, sdp_body sippy_types.MsgBody, parsed_body sippy_types.Sdp, sect *sippy_sdp.SdpMediaDescription, sections_left *int64, result_callback func(sippy_types.MsgBody)) {
	if cb_args != nil {
		if s.after_sdp_change != nil {
			s.after_sdp_change(cb_args)
		}
		sect.GetCHeader().SetAType(cb_args.family)
		sect.GetCHeader().SetAddr(cb_args.rtpproxy_address)
		if sect.GetMHeader().GetPort() != "0" {
			sect.GetMHeader().SetPort(cb_args.rtpproxy_port)
		}
		if cb_args.sendonly {
			sect.RemoveAHeader("sendrecv")
			if !sect.HasAHeader([]string{"recvonly", "sendonly", "inactive"}) {
				sect.AddHeader("a", "sendonly")
			}
		}
		if s.repacketize > 0 {
			sect.RemoveAHeader("ptime:")
			sect.AddHeader("a", "ptime:"+strconv.Itoa(s.repacketize))
		}
	}
	if atomic.AddInt64(sections_left, -1) > 0 {
		// more work is in progress
		return
	}
	if s.owner.insert_nortpp {
		parsed_body.AppendAHeader("nortpproxy=yes")
	}
	sdp_body.SetNeedsUpdate(false)
	// RFC4566
	// *******
	// For privacy reasons, it is sometimes desirable to obfuscate the
	// username and IP address of the session originator.  If this is a
	// concern, an arbitrary <username> and private <unicast-address> MAY be
	// chosen to populate the "o=" field, provided that these are selected
	// in a manner that does not affect the global uniqueness of the field.
	// *******
	origin := parsed_body.GetOHeader()
	origin.SetAddress("192.0.2.1")
	origin.SetAddressType("IP4")
	origin.SetNetworkType("IN")
	result_callback(sdp_body)
}

func (s *_rtpps_side) _stop_play(cb func(string), index int) {
	if !s.otherside.session_exists {
		return
	}
	command := "S " + s.owner.call_id + "-" + strconv.Itoa(index) + " " + s.from_tag + " " + s.to_tag
	s.owner.send_command(command, func(r string) { s.owner.command_result(r, cb) })
}
