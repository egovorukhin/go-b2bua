//

package sippy_net

import (
	"net"
	"strings"
)

type MyAddress struct {
	is_system bool
	address   string
}

type MyPort struct {
	is_system bool
	port      string
}

/* MyAddress methods */

func NewMyAddress(address string) *MyAddress {
	s := &MyAddress{
		is_system: false,
		address:   address,
	}
	s.normalize()
	return s
}

func NewSystemAddress(address string) *MyAddress {
	s := &MyAddress{
		is_system: true,
		address:   address,
	}
	s.normalize()
	return s
}

func (s *MyAddress) normalize() {
	if s.address == "" {
		return
	}
	if s.address[0] != '[' && (strings.IndexByte(s.address, ':') >= 0 || strings.IndexByte(s.address, '%') >= 0) {
		s.address = "[" + s.address + "]"
	}
}

func (s *MyAddress) IsSystemDefault() bool {
	return s.is_system
}

func (s *MyAddress) ParseIP() net.IP {
	if s.address[0] == '[' {
		return net.ParseIP(s.address[1 : len(s.address)-1])
	}
	return net.ParseIP(s.address)
}

func (s *MyAddress) String() string {
	return s.address
}

func (s *MyAddress) GetCopy() *MyAddress {
	tmp := *s
	return &tmp
}

/* MyPort methods */

func NewMyPort(port string) *MyPort {
	return &MyPort{
		is_system: false,
		port:      port,
	}
}

func NewSystemPort(port string) *MyPort {
	return &MyPort{
		is_system: true,
		port:      port,
	}
}

func (s *MyPort) String() string {
	return s.port
}

func (s *MyPort) IsSystemDefault() bool {
	return s.is_system
}

func (s *MyPort) GetCopy() *MyPort {
	tmp := *s
	return &tmp
}

/* HostPort */
type HostPort struct {
	Host *MyAddress
	Port *MyPort
}

func NewHostPort(host, port string) *HostPort {
	return &HostPort{
		Host: NewMyAddress(host),
		Port: NewMyPort(port),
	}
}

func NewHostPortFromAddr(addr net.Addr) (*HostPort, error) {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	return NewHostPort(host, port), nil
}

func (s *HostPort) ParseIP() net.IP {
	return s.Host.ParseIP()
}

func (s *HostPort) String() string {
	if s == nil {
		return "nil"
	}
	return s.Host.String() + ":" + s.Port.String()
}

func (s *HostPort) GetCopy() *HostPort {
	return &HostPort{
		Host: s.Host.GetCopy(),
		Port: s.Port.GetCopy(),
	}
}

func IsIP4(ip net.IP) bool {
	if strings.IndexByte(ip.String(), '.') >= 0 {
		return true
	}
	return false
}
