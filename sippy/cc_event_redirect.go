package sippy

import (
	"sort"

	"github.com/egovorukhin/go-b2bua/sippy/headers"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type CCEventRedirect struct {
	CCEventGeneric
	redirect_addresses []*sippy_header.SipAddress
	scode              int
	scode_reason       string
	body               sippy_types.MsgBody
}

func NewCCEventRedirect(scode int, scode_reason string, body sippy_types.MsgBody, addrs []*sippy_header.SipAddress, rtime *sippy_time.MonoTime, origin string, extra_headers ...sippy_header.SipHeader) *CCEventRedirect {
	return &CCEventRedirect{
		CCEventGeneric:     newCCEventGeneric(rtime, origin, extra_headers...),
		scode:              scode,
		scode_reason:       scode_reason,
		body:               body,
		redirect_addresses: addrs,
	}
}

func (s *CCEventRedirect) String() string { return "CCEventRedirect" }

func (s *CCEventRedirect) GetRedirectURL() *sippy_header.SipAddress {
	return s.redirect_addresses[0]
}

func (s *CCEventRedirect) GetRedirectURLs() []*sippy_header.SipAddress {
	return s.redirect_addresses
}

func (s *CCEventRedirect) GetContacts() []*sippy_header.SipContact {
	addrs := s.redirect_addresses
	if addrs == nil || len(addrs) == 0 {
		return nil
	}
	ret := make([]*sippy_header.SipContact, len(addrs))
	for i, addr := range addrs {
		ret[i] = sippy_header.NewSipContactFromAddress(addr)
	}
	return ret
}

func (s *CCEventRedirect) SortAddresses() {
	if len(s.redirect_addresses) == 1 {
		return
	}
	sort.Sort(sortRedirectAddresses(s.redirect_addresses))
}

func (*CCEventRedirect) GetBody() sippy_types.MsgBody {
	return nil
}

type sortRedirectAddresses []*sippy_header.SipAddress

func (s sortRedirectAddresses) Len() int      { return len(s) }
func (s sortRedirectAddresses) Swap(x, y int) { s[x], s[y] = s[y], s[x] }
func (s sortRedirectAddresses) Less(x, y int) bool {
	// descending order
	return s[x].GetQ() > s[y].GetQ()
}
