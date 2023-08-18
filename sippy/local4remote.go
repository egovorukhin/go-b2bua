package sippy

import (
	"net"
	"sync"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/net"
)

type local4remote struct {
	config         sippy_conf.Config
	cache_r2l      map[string]*sippy_net.HostPort
	cache_r2l_old  map[string]*sippy_net.HostPort
	cache_l2s      map[string]sippy_net.Transport
	handleIncoming sippy_net.DataPacketReceiver
	fixed          bool
	tfactory       sippy_net.SipTransportFactory
	lock           sync.Mutex
}

func NewLocal4Remote(config sippy_conf.Config, handleIncoming sippy_net.DataPacketReceiver) (*local4remote, error) {
	s := &local4remote{
		config:         config,
		cache_r2l:      make(map[string]*sippy_net.HostPort),
		cache_r2l_old:  make(map[string]*sippy_net.HostPort),
		cache_l2s:      make(map[string]sippy_net.Transport),
		handleIncoming: handleIncoming,
		fixed:          false,
		tfactory:       config.GetSipTransportFactory(),
	}
	if s.tfactory == nil {
		s.tfactory = NewDefaultSipTransportFactory(config)
	}
	laddresses := make([]*sippy_net.HostPort, 0)
	if config.SipAddress().IsSystemDefault() {
		laddresses = append(laddresses, sippy_net.NewHostPort("0.0.0.0", config.SipPort().String()))
		if config.GetIPV6Enabled() {
			laddresses = append(laddresses, sippy_net.NewHostPort("[::]", config.SipPort().String()))
		}
	} else {
		laddresses = append(laddresses, sippy_net.NewHostPort(config.SipAddress().String(), config.SipPort().String()))
		s.fixed = true
	}
	var last_error error
	for _, laddress := range laddresses {
		/*
		   sopts := NewUdpServerOpts(laddress, handleIncoming)
		   server, err := NewUdpServer(config, sopts)
		*/
		server, err := s.tfactory.NewSipTransport(laddress, handleIncoming)
		if err != nil {
			if !config.SipAddress().IsSystemDefault() {
				return nil, err
			} else {
				last_error = err
			}
		} else {
			s.cache_l2s[laddress.String()] = server
		}
	}
	if len(s.cache_l2s) == 0 && last_error != nil {
		return nil, last_error
	}
	return s, nil
}

func (s *local4remote) getServer(address *sippy_net.HostPort, is_local bool /*= false*/) sippy_net.Transport {
	var laddress *sippy_net.HostPort
	var ok bool

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.fixed {
		for _, server := range s.cache_l2s {
			return server
		}
		return nil
	}
	if !is_local {
		laddress, ok = s.cache_r2l[address.Host.String()]
		if !ok {
			laddress, ok = s.cache_r2l_old[address.Host.String()]
			if ok {
				s.cache_r2l[address.Host.String()] = laddress
			}
		}
		if ok {
			server, ok := s.cache_l2s[laddress.String()]
			if !ok {
				return nil
			} else {
				//print 'local4remote-1: local address for %s is %s' % (address[0], laddress[0])
				return server
			}
		}
		lookup_address, err := net.ResolveUDPAddr("udp", address.String())
		if err != nil {
			return nil
		}
		_laddress := ""
		c, err := net.ListenUDP("udp", lookup_address)
		if err == nil {
			c.Close()
			_laddress, _, err = net.SplitHostPort(lookup_address.String())
			if err != nil {
				return nil // should not happen
			}
		} else {
			conn, err := net.DialUDP("udp", nil, lookup_address)
			if err != nil {
				return nil // should not happen
			}
			_laddress, _, err = net.SplitHostPort(conn.LocalAddr().String())
			conn.Close()
			if err != nil {
				return nil // should not happen
			}
		}
		laddress = sippy_net.NewHostPort(_laddress, s.config.SipPort().String())
		s.cache_r2l[address.Host.String()] = laddress
	} else {
		laddress = address
	}
	server, ok := s.cache_l2s[laddress.String()]
	if !ok {
		var err error
		/*
		   sopts := NewUdpServerOpts(laddress, s.handleIncoming)
		   server, err = NewUdpServer(s.config, sopts)
		*/
		server, err = s.tfactory.NewSipTransport(laddress, s.handleIncoming)
		if err != nil {
			s.config.ErrorLogger().Errorf("Cannot bind %s: %s", laddress.String(), err.Error())
			return nil
		}
		s.cache_l2s[laddress.String()] = server
	}
	//print 'local4remote-2: local address for %s is %s' % (address[0], laddress[0])
	return server
}

func (s *local4remote) rotateCache() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cache_r2l_old = s.cache_r2l
	s.cache_r2l = make(map[string]*sippy_net.HostPort)
}

func (s *local4remote) shutdown() {
	for _, userv := range s.cache_l2s {
		userv.Shutdown()
	}
	s.cache_l2s = make(map[string]sippy_net.Transport)
}
