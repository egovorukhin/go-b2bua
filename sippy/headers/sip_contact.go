package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type SipContact struct {
	compactName
	*sipAddressHF
	Asterisk bool
}

var sipContactName compactName = newCompactName("Contact", "m")

func NewSipContact(config sippy_conf.Config) *SipContact {
	return NewSipContactFromHostPort(config.GetMyAddress(), config.GetMyPort())
}

func NewSipContactFromHostPort(host *sippy_net.MyAddress, port *sippy_net.MyPort) *SipContact {
	return &SipContact{
		compactName: sipContactName,
		Asterisk:    false,
		sipAddressHF: newSipAddressHF(
			NewSipAddress("Anonymous",
				NewSipURL("", host, port, false))),
	}
}

func NewSipContactFromAddress(addr *SipAddress) *SipContact {
	return &SipContact{
		compactName:  sipContactName,
		Asterisk:     false,
		sipAddressHF: newSipAddressHF(addr),
	}
}

func (c *SipContact) GetCopy() *SipContact {
	return &SipContact{
		compactName:  sipContactName,
		sipAddressHF: c.sipAddressHF.getCopy(),
		Asterisk:     c.Asterisk,
	}
}

func (c *SipContact) GetCopyAsIface() SipHeader {
	return c.GetCopy()
}

func CreateSipContact(body string) []SipHeader {
	var rval []SipHeader
	if body == "*" {
		rval = append(rval, &SipContact{
			Asterisk:    true,
			compactName: sipContactName,
		})
	} else {
		addresses := CreateSipAddressHFs(body)
		for _, addr := range addresses {
			rval = append(rval, &SipContact{
				sipAddressHF: addr,
				Asterisk:     false,
				compactName:  sipContactName,
			})
		}
	}
	return rval
}

func (c *SipContact) StringBody() string {
	return c.LocalStringBody(nil)
}

func (c *SipContact) LocalStringBody(hostPort *sippy_net.HostPort) string {
	if c.Asterisk {
		return "*"
	}
	return c.sipAddressHF.LocalStringBody(hostPort)
}

func (c *SipContact) String() string {
	return c.LocalStr(nil, false)
}

func (c *SipContact) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	hname := c.Name()
	if compact {
		hname = c.CompactName()
	}
	return hname + ": " + c.LocalStringBody(hostPort)
}
