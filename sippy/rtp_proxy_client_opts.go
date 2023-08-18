package sippy

import (
	"net"
	"strings"
	"time"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/log"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/types"
)

type rtpProxyClientOpts struct {
	no_version_check bool
	nworkers         *int
	hrtb_retr_ival   time.Duration
	hrtb_ival        time.Duration
	rtpp_class       func(sippy_types.RtpProxyClient, sippy_conf.Config, net.Addr, *sippy_net.HostPort) (rtp_proxy_transport, error)
	rtppaddr         net.Addr
	config           sippy_conf.Config
	logger           sippy_log.ErrorLogger
	proxy_address    string
	bind_address     *sippy_net.HostPort
}

func NewRtpProxyClientOpts(spath string, bind_address *sippy_net.HostPort, config sippy_conf.Config, logger sippy_log.ErrorLogger) (*rtpProxyClientOpts, error) {
	s := &rtpProxyClientOpts{
		hrtb_retr_ival:   60 * time.Second,
		hrtb_ival:        1 * time.Second,
		no_version_check: false,
		logger:           logger,
		config:           config,
		bind_address:     bind_address,
	}
	var err error

	if strings.HasPrefix(spath, "udp:") {
		tmp := strings.SplitN(spath, ":", 3)
		if len(tmp) == 2 {
			s.rtppaddr, err = net.ResolveUDPAddr("udp", tmp[1]+":22222")
		} else {
			s.rtppaddr, err = net.ResolveUDPAddr("udp", tmp[1]+":"+tmp[2])
		}
		if err != nil {
			return nil, err
		}
		s.proxy_address, _, err = net.SplitHostPort(s.rtppaddr.String())
		if err != nil {
			return nil, err
		}
		s.rtpp_class = newRtp_proxy_client_udp
	} else if strings.HasPrefix(spath, "udp6:") {
		tmp := strings.SplitN(spath, ":", 2)
		spath := tmp[1]
		rtp_proxy_host, rtp_proxy_port := spath, "22222"
		if spath[len(spath)-1] != ']' {
			idx := strings.LastIndexByte(spath, ':')
			if idx < 0 {
				rtp_proxy_host = spath
			} else {
				rtp_proxy_host, rtp_proxy_port = spath[:idx], spath[idx+1:]
			}
		}
		if rtp_proxy_host[0] != '[' {
			rtp_proxy_host = "[" + rtp_proxy_host + "]"
		}
		s.rtppaddr, err = net.ResolveUDPAddr("udp", rtp_proxy_host+":"+rtp_proxy_port)
		if err != nil {
			return nil, err
		}
		s.proxy_address, _, err = net.SplitHostPort(s.rtppaddr.String())
		if err != nil {
			return nil, err
		}
		s.rtpp_class = newRtp_proxy_client_udp
	} else if strings.HasPrefix(spath, "tcp:") {
		tmp := strings.SplitN(spath, ":", 3)
		if len(tmp) == 2 {
			s.rtppaddr, err = net.ResolveTCPAddr("tcp", tmp[1]+":22222")
		} else {
			s.rtppaddr, err = net.ResolveTCPAddr("tcp", tmp[1]+":"+tmp[2])
		}
		if err != nil {
			return nil, err
		}
		s.proxy_address, _, err = net.SplitHostPort(s.rtppaddr.String())
		if err != nil {
			return nil, err
		}
		s.rtpp_class = newRtp_proxy_client_stream
	} else if strings.HasPrefix(spath, "tcp6:") {
		tmp := strings.SplitN(spath, ":", 2)
		spath := tmp[1]
		rtp_proxy_host, rtp_proxy_port := spath, "22222"
		if spath[len(spath)-1] != ']' {
			idx := strings.LastIndexByte(spath, ':')
			if idx < 0 {
				rtp_proxy_host = spath
			} else {
				rtp_proxy_host, rtp_proxy_port = spath[:idx], spath[idx+1:]
			}
		}
		if rtp_proxy_host[0] != '[' {
			rtp_proxy_host = "[" + rtp_proxy_host + "]"
		}
		s.rtppaddr, err = net.ResolveTCPAddr("tcp", rtp_proxy_host+":"+rtp_proxy_port)
		if err != nil {
			return nil, err
		}
		s.proxy_address, _, err = net.SplitHostPort(s.rtppaddr.String())
		if err != nil {
			return nil, err
		}
		s.rtpp_class = newRtp_proxy_client_stream
	} else {
		if strings.HasPrefix(spath, "unix:") {
			s.rtppaddr, err = net.ResolveUnixAddr("unix", spath[5:])
		} else if strings.HasPrefix(spath, "cunix:") {
			s.rtppaddr, err = net.ResolveUnixAddr("unix", spath[6:])
		} else {
			s.rtppaddr, err = net.ResolveUnixAddr("unix", spath)
		}
		if err != nil {
			return nil, err
		}
		s.proxy_address = s.config.SipAddress().String()
		s.rtpp_class = newRtp_proxy_client_stream
	}
	return s, nil
}

func (s *rtpProxyClientOpts) SetHeartbeatInterval(ival time.Duration) {
	s.hrtb_ival = ival
}

func (s *rtpProxyClientOpts) SetHeartbeatRetryInterval(ival time.Duration) {
	s.hrtb_retr_ival = ival
}

func (s *rtpProxyClientOpts) GetNWorkers() *int {
	return s.nworkers
}
