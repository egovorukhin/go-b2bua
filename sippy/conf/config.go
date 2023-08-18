//

package sippy_conf

import (
	"net"
	"os"

	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type Config interface {
	SipAddress() *sippy_net.MyAddress
	SipPort() *sippy_net.MyPort
	GetIPV6Enabled() bool
	SetIPV6Enabled(bool)
	SetSipAddress(*sippy_net.MyAddress)
	SetSipPort(*sippy_net.MyPort)
	SipLogger() sippy_log.SipLogger
	ErrorLogger() sippy_log.ErrorLogger
	GetMyUAName() string
	SetMyUAName(s string)
	SetAllowFormats(f []int)

	GetMyAddress() *sippy_net.MyAddress
	SetMyAddress(*sippy_net.MyAddress)
	GetMyPort() *sippy_net.MyPort
	SetMyPort(*sippy_net.MyPort)
	DefaultPort() *sippy_net.MyPort

	AutoConvertTelUrl() bool
	SetAutoConvertTelUrl(bool)
	GetSipTransportFactory() sippy_net.SipTransportFactory
	SetSipTransportFactory(sippy_net.SipTransportFactory)
}

type config struct {
	sipAddress  *sippy_net.MyAddress
	sipPort     *sippy_net.MyPort
	defaultPort *sippy_net.MyPort
	sipLogger   sippy_log.SipLogger
	errorLogger sippy_log.ErrorLogger
	ipv6Enabled bool

	myAddress         *sippy_net.MyAddress
	myPort            *sippy_net.MyPort
	myUaName          string
	allowFormats      []int
	autoConvertTelUrl bool
	tFactory          sippy_net.SipTransportFactory
}

func NewConfig(errorLogger sippy_log.ErrorLogger, sipLogger sippy_log.SipLogger) Config {
	address := "127.0.0.1"
	if hostname, err := os.Hostname(); err == nil {
		if addresses, err := net.LookupHost(hostname); err == nil && len(addresses) > 0 {
			address = addresses[0]
		}
	}
	return &config{
		ipv6Enabled:       true,
		errorLogger:       errorLogger,
		sipLogger:         sipLogger,
		myAddress:         sippy_net.NewSystemAddress(address),
		myPort:            sippy_net.NewSystemPort("5060"),
		myUaName:          "Sippy",
		allowFormats:      make([]int, 0),
		autoConvertTelUrl: false,
		defaultPort:       sippy_net.NewSystemPort("5060"),
	}
}

func (c *config) SipAddress() *sippy_net.MyAddress {
	if c.sipAddress == nil {
		return c.myAddress
	}
	return c.sipAddress
}

func (c *config) SipLogger() sippy_log.SipLogger {
	return c.sipLogger
}

func (c *config) SipPort() *sippy_net.MyPort {
	if c.sipPort == nil {
		return c.myPort
	}
	return c.sipPort
}

func (c *config) SetIPV6Enabled(v bool) {
	c.ipv6Enabled = v
}

func (c *config) GetIPV6Enabled() bool {
	return c.ipv6Enabled
}

func (c *config) ErrorLogger() sippy_log.ErrorLogger {
	return c.errorLogger
}

func (c *config) SetSipAddress(addr *sippy_net.MyAddress) {
	c.sipAddress = addr
}

func (c *config) SetSipPort(port *sippy_net.MyPort) {
	c.sipPort = port
}

func (c *config) GetMyAddress() *sippy_net.MyAddress {
	return c.myAddress
}

func (c *config) SetMyAddress(addr *sippy_net.MyAddress) {
	c.myAddress = addr
}

func (c *config) GetMyPort() *sippy_net.MyPort {
	return c.myPort
}

func (c *config) SetMyPort(port *sippy_net.MyPort) {
	c.myPort = port
}

func (c *config) GetMyUAName() string {
	return c.myUaName
}

func (c *config) SetMyUAName(s string) {
	c.myUaName = s
}

func (c *config) SetAllowFormats(f []int) {
	c.allowFormats = f
}

func (c *config) AutoConvertTelUrl() bool {
	return c.autoConvertTelUrl
}

func (c *config) SetAutoConvertTelUrl(v bool) {
	c.autoConvertTelUrl = v
}

func (c *config) GetSipTransportFactory() sippy_net.SipTransportFactory {
	return c.tFactory
}

func (c *config) SetSipTransportFactory(tFactory sippy_net.SipTransportFactory) {
	c.tFactory = tFactory
}

func (c *config) DefaultPort() *sippy_net.MyPort {
	return c.defaultPort
}
