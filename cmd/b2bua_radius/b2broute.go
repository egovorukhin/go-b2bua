package main

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy"
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type ainfo_item struct {
	ip   net.IP
	port string
}

func (s *ainfo_item) HostPort() *sippy_net.HostPort {
	return sippy_net.NewHostPort(s.ip.String(), s.port)
}

type B2BRoute struct {
	cld                 string
	cld_set             bool
	hostPort            string
	hostonly            string
	huntstop_scodes     []int
	ainfo               []*ainfo_item
	credit_time         time.Duration
	crt_set             bool
	expires             time.Duration
	no_progress_expires time.Duration
	forward_on_fail     bool
	user                string
	passw               string
	cli                 string
	cli_set             bool
	caller_name         string
	extra_headers       []sippy_header.SipHeader
	rtpp                bool
	outbound_proxy      *sippy_net.HostPort
	rnum                int
}

/*
from sippy.SipHeader import SipHeader
from sippy.SipConf import SipConf

from urllib import unquote
from socket import getaddrinfo, SOCK_STREAM, AF_INET, AF_INET6

class B2BRoute(object):
    rnum = nil
    addrinfo = nil
    params = nil
    ainfo = nil
*/

func NewB2BRoute(sroute string, global_config sippy_conf.Config) (*B2BRoute, error) {
	var hostPort []string
	var err error

	r := &B2BRoute{
		huntstop_scodes: []int{},
		cld_set:         false,
		crt_set:         false,
		forward_on_fail: false,
		cli_set:         false,
		extra_headers:   []sippy_header.SipHeader{},
		rtpp:            true,
	}
	route := strings.Split(sroute, ";")
	if strings.IndexRune(route[0], '@') != -1 {
		tmp := strings.SplitN(route[0], "@", 2)
		r.cld, r.hostPort = tmp[0], tmp[1]
		// Allow CLD to be forcefully removed by sending `Routing:@host" entry,
		// as opposed to the Routing:host, which means that CLD should be obtained
		// from the incoming call leg.
		r.cld_set = true
	} else {
		r.hostPort = route[0]
	}
	ipv6only := false
	if r.hostPort[0] != '[' {
		hostPort = strings.SplitN(r.hostPort, ":", 2)
		r.hostonly = hostPort[0]
	} else {
		hostPort = strings.SplitN(r.hostPort[1:], "]", 2)
		if len(hostPort) > 1 {
			if hostPort[1] == "" {
				hostPort = hostPort[:1]
			} else {
				hostPort[1] = hostPort[1][1:]
			}
		}
		ipv6only = true
		r.hostonly = "[" + hostPort[0] + "]"
	}
	var port *sippy_net.MyPort
	if len(hostPort) == 1 {
		port = global_config.GetMyPort()
	} else {
		port = sippy_net.NewMyPort(hostPort[1])
	}
	r.ainfo = make([]*ainfo_item, 0)
	ips, err := net.LookupIP(hostPort[0])
	if err != nil {
		return nil, errors.New("NewB2BRoute: error resolving host IP '" + hostPort[0] + "': " + err.Error())
	}
	for _, ip := range ips {
		if ipv6only && sippy_net.IsIP4(ip) {
			continue
		}
		r.ainfo = append(r.ainfo, &ainfo_item{ip, port.String()})
	}
	//s.params = []string{}
	for _, x := range route[1:] {
		av := strings.SplitN(x, "=", 2)
		switch av[0] {
		case "credit-time":
			v, err := strconv.Atoi(av[1])
			if err != nil {
				return nil, errors.New("Error parsing credit-time '" + av[1] + "': " + err.Error())
			}
			if v < 0 {
				v = 0
			}
			r.credit_time = time.Duration(v * int(time.Second))
			r.crt_set = true
		case "expires":
			v, err := strconv.Atoi(av[1])
			if err != nil {
				return nil, errors.New("Error parsing the expires '" + av[1] + "': " + err.Error())
			}
			if v < 0 {
				v = 0
			}
			r.expires = time.Duration(v * int(time.Second))
		case "hs_scodes":
			for _, s := range strings.Split(av[1], ",") {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				scode, err := strconv.Atoi(s)
				if err != nil {
					return nil, errors.New("Error parsing hs_scodes '" + s + "': " + err.Error())
				}
				r.huntstop_scodes = append(r.huntstop_scodes, scode)
			}
		case "np_expires":
			v, err := strconv.Atoi(av[1])
			if err != nil {
				return nil, errors.New("Error parsing the no_progress_expires '" + av[1] + "': " + err.Error())
			}
			if v < 0 {
				v = 0
			}
			r.no_progress_expires = time.Duration(v * int(time.Second))
		case "forward_on_fail":
			r.forward_on_fail = true
		case "auth":
			tmp := strings.SplitN(av[1], ":", 2)
			if len(tmp) != 2 {
				return nil, errors.New("Error parsing the auth (no colon) '" + av[1] + "': " + err.Error())
			}
			r.user, r.passw = tmp[0], tmp[1]
		case "cli":
			r.cli = av[1]
			r.cli_set = true
		case "cnam":
			r.caller_name, err = url.QueryUnescape(av[1])
			if err != nil {
				return nil, errors.New("Error parsing the cnam '" + av[1] + "': " + err.Error())
			}
		case "ash":
			var v string
			var ash []sippy_header.SipHeader
			v, err = url.QueryUnescape(av[1])
			if err == nil {
				ash, err = sippy.ParseSipHeader(v)
			}
			if err != nil {
				return nil, errors.New("Error parsing the ash '" + av[1] + "': " + err.Error())
			}
			r.extra_headers = append(r.extra_headers, ash...)
		case "rtpp":
			v, err := strconv.Atoi(av[1])
			if err != nil {
				return nil, errors.New("Error parsing the rtpp '" + av[1] + "': " + err.Error())
			}
			r.rtpp = (v != 0)
		case "op":
			host_port := strings.SplitN(av[1], ":", 2)
			if len(host_port) == 1 {
				r.outbound_proxy = sippy_net.NewHostPort(av[1], "5060")
			} else {
				r.outbound_proxy = sippy_net.NewHostPort(host_port[0], host_port[1])
			}
			//default:
			//    s.params[a] = v
		}
	}
	return r, nil
}

func (s *B2BRoute) customize(rnum int, default_cld, default_cli string, default_credit_time time.Duration, pass_headers []sippy_header.SipHeader, max_credit_time time.Duration) {
	s.rnum = rnum
	if !s.cld_set {
		s.cld = default_cld
	}
	if !s.cli_set {
		s.cli = default_cli
	}
	if !s.crt_set {
		s.credit_time = default_credit_time
	}
	//if s.params.has_key("gt") {
	//    timeout, skip = s.params["gt"].split(",", 1)
	//    s.params["group_timeout"] = (int(timeout), rnum + int(skip))
	//}
	s.extra_headers = append(s.extra_headers, pass_headers...)
	if max_credit_time != 0 {
		if s.credit_time == 0 || s.credit_time > max_credit_time {
			s.credit_time = max_credit_time
		}
	}
}

func (s *B2BRoute) getCopy() *B2BRoute {
	if s == nil {
		return nil
	}
	cself := *s
	if s.outbound_proxy != nil {
		cself.outbound_proxy = s.outbound_proxy.GetCopy()
	}

	cself.huntstop_scodes = make([]int, len(s.huntstop_scodes))
	copy(cself.huntstop_scodes, s.huntstop_scodes)

	cself.ainfo = make([]*ainfo_item, len(s.ainfo))
	copy(cself.ainfo, s.ainfo)

	cself.extra_headers = make([]sippy_header.SipHeader, len(s.extra_headers))
	copy(cself.extra_headers, s.extra_headers)

	return &cself
}

func (s *B2BRoute) getNHAddr(source *sippy_net.HostPort) *sippy_net.HostPort {
	src_ip := net.ParseIP(source.Host.String())
	if src_ip == nil {
		return s.ainfo[0].HostPort()
	}
	src_is_ipv4 := sippy_net.IsIP4(src_ip)
	for _, it := range s.ainfo {
		if src_is_ipv4 && sippy_net.IsIP4(it.ip) {
			return it.HostPort()
		} else if !src_is_ipv4 && !sippy_net.IsIP4(it.ip) {
			return it.HostPort()
		}
	}
	return s.ainfo[0].HostPort()
}
