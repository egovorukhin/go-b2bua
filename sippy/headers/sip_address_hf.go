package sippy_header

import (
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/conf"
	"github.com/egovorukhin/go-b2bua/sippy/net"
)

type sipAddressHFBody struct {
	Address *SipAddress
}

func (b *sipAddressHFBody) getCopy() *sipAddressHFBody {
	return &sipAddressHFBody{
		Address: b.Address.GetCopy(),
	}
}

type sipAddressHF struct {
	stringBody string
	body       *sipAddressHFBody
}

func newSipAddressHF(addr *SipAddress) *sipAddressHF {
	return &sipAddressHF{
		body: &sipAddressHFBody{Address: addr},
	}
}

func CreateSipAddressHFs(body string) []*sipAddressHF {
	addresses := []string{}
	pidx := 0
	for {
		idx := strings.IndexByte(body[pidx:], ',')
		if idx == -1 {
			break
		}
		idx += pidx
		onum, cnum, qnum := 0, 0, 0
		for _, r := range body[:idx] {
			switch r {
			case '<':
				onum++
			case '>':
				cnum++
			case '"':
				qnum++
			}
		}
		if (onum == 0 && cnum == 0 && qnum == 0) || (onum > 0 &&
			onum == cnum && (qnum%2 == 0)) {
			addresses = append(addresses, body[:idx])
			body = body[idx+1:]
			pidx = 0
		} else {
			pidx = idx + 1
		}
	}
	if body != "" {
		addresses = append(addresses, body)
	}
	retval := make([]*sipAddressHF, len(addresses))
	for i, address := range addresses {
		retval[i] = &sipAddressHF{stringBody: address}
	}
	return retval
}

func (h *sipAddressHF) parse(config sippy_conf.Config) error {
	addr, err := ParseSipAddress(h.stringBody, false /* relaxedparser */, config)
	if err != nil {
		return err
	}
	h.body = &sipAddressHFBody{
		Address: addr,
	}
	return nil
}

func (h *sipAddressHF) getCopy() *sipAddressHF {
	cself := *h
	if h.body != nil {
		cself.body = h.body.getCopy()
	}
	return &cself
}

func (h *sipAddressHF) GetBody(config sippy_conf.Config) (*SipAddress, error) {
	if h.body == nil {
		if err := h.parse(config); err != nil {
			return nil, err
		}
	}
	return h.body.Address, nil
}

func (h *sipAddressHF) StringBody() string {
	return h.LocalStringBody(nil)
}

func (h *sipAddressHF) LocalStringBody(hostPort *sippy_net.HostPort) string {
	if h.body != nil {
		return h.body.Address.LocalStr(hostPort)
	}
	return h.stringBody
}
