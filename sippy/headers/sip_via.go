package sippy_header

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"strings"

	"github.com/sippy/go-b2bua/sippy/conf"
	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/utils"
)

type SipViaBody struct {
	sipVer       string
	host         *sippy_net.MyAddress
	port         *sippy_net.MyPort
	extraHeaders string

	received  *string
	rPort     *string
	ttl       *string
	mAddr     *string
	branch    *string
	extension *string

	receivedExists  bool
	rPortExists     bool
	ttlExists       bool
	mAddrExists     bool
	branchExists    bool
	extensionExists bool
}

type SipVia struct {
	compactName
	stringBody string
	body       *SipViaBody
}

var sipViaName compactName = newCompactName("Via", "v")

func CreateSipVia(body string) []SipHeader {
	vias := strings.Split(body, ",")
	rval := make([]SipHeader, len(vias))
	for i, via := range vias {
		rval[i] = &SipVia{
			compactName: sipViaName,
			stringBody:  via,
		}
	}
	return rval
}

func (s *SipVia) parse() error {
	arr := sippy_utils.FieldsN(s.stringBody, 2)
	if len(arr) != 2 {
		return errors.New("Bad via: '" + s.stringBody + "'")
	}
	via := &SipViaBody{
		sipVer: arr[0],
	}
	arr = strings.Split(arr[1], ";")
	var val *string
	for _, param := range arr[1:] {
		param = strings.TrimSpace(param)
		sparam := strings.SplitN(param, "=", 2)
		val = nil
		if len(sparam) == 2 {
			val = &sparam[1]
		}
		switch sparam[0] {
		case "received":
			via.received = val
			via.receivedExists = true
		case "rPort":
			via.rPort = val
			via.rPortExists = true
		case "ttl":
			via.ttl = val
			via.ttlExists = true
		case "mAddr":
			via.mAddr = val
			via.mAddrExists = true
		case "branch":
			via.branch = val
			via.branchExists = true
		case "extension":
			via.extension = val
			via.extensionExists = true
		default:
			via.extraHeaders += ";" + sparam[0]
			if val != nil {
				via.extraHeaders += "=" + *val
			}
		}
	}
	host, port, err := net.SplitHostPort(arr[0])
	if err != nil {
		via.host = sippy_net.NewMyAddress(arr[0])
		via.port = nil
	} else {
		via.host = sippy_net.NewMyAddress(host)
		via.port = sippy_net.NewMyPort(port)
	}
	s.body = via
	return nil
}

func (s *SipVia) GetBody() (*SipViaBody, error) {
	if s.body == nil {
		if err := s.parse(); err != nil {
			return nil, err
		}
	}
	return s.body, nil
}

func NewSipVia(config sippy_conf.Config) *SipVia {
	return &SipVia{
		compactName: sipViaName,
		body:        newSipViaBody(config),
	}
}

func newSipViaBody(config sippy_conf.Config) *SipViaBody {
	return &SipViaBody{
		rPortExists: true,
		sipVer:      "SIP/2.0/UDP",
		host:        config.GetMyAddress(),
		port:        config.DefaultPort(),

		receivedExists:  false,
		ttlExists:       false,
		mAddrExists:     false,
		branchExists:    false,
		extensionExists: false,
	}
}

func (s *SipVia) StringBody() string {
	return s.LocalStringBody(nil)
}

func (s *SipVia) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipVia) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	if compact {
		return s.CompactName() + ": " + s.LocalStringBody(hostPort)
	}
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipVia) LocalStringBody(hostPort *sippy_net.HostPort) string {
	if s.body != nil {
		return s.body.localString(hostPort)
	}
	return s.stringBody
}

func (b *SipViaBody) localString(hostPort *sippy_net.HostPort) (s string) {
	if hostPort != nil && b.host.IsSystemDefault() {
		s = b.sipVer + " " + hostPort.Host.String()
	} else {
		s = b.sipVer + " " + b.host.String()
	}
	if b.port != nil {
		if hostPort != nil && b.port.IsSystemDefault() {
			s += ":" + hostPort.Port.String()
		} else {
			s += ":" + b.port.String()
		}
	}
	for _, it := range []struct {
		key    string
		val    *string
		exists bool
	}{
		{"received", b.received, b.receivedExists},
		{"rPort", b.rPort, b.rPortExists},
		{"ttl", b.ttl, b.ttlExists},
		{"mAddr", b.mAddr, b.mAddrExists},
		{"branch", b.branch, b.branchExists},
		{"extension", b.extension, b.extensionExists},
	} {
		if it.exists {
			s += ";" + it.key
			if it.val != nil {
				s += "=" + *it.val
			}
		}
	}
	return s + b.extraHeaders
}

func (s *SipVia) GetCopy() *SipVia {
	tmp := *s
	if s.body != nil {
		tmp.body = s.body.getCopy()
	}
	return &tmp
}

func (b *SipViaBody) getCopy() *SipViaBody {
	tmp := *b
	if b.received != nil {
		tmp_s := *b.received
		tmp.received = &tmp_s
	}
	if b.rPort != nil {
		tmp_s := *b.rPort
		tmp.rPort = &tmp_s
	}
	if b.ttl != nil {
		tmp_s := *b.ttl
		tmp.ttl = &tmp_s
	}
	if b.mAddr != nil {
		tmp_s := *b.mAddr
		tmp.mAddr = &tmp_s
	}
	if b.branch != nil {
		tmp_s := *b.branch
		tmp.branch = &tmp_s
	}
	if b.extension != nil {
		tmp_s := *b.extension
		tmp.extension = &tmp_s
	}
	return &tmp
}

func (s *SipVia) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (b *SipViaBody) GenBranch() {
	buf := make([]byte, 16)
	rand.Read(buf)
	tmp := "z9hG4bK" + hex.EncodeToString(buf)
	b.branch = &tmp
	b.branchExists = true
}

func (b *SipViaBody) GetBranch() string {
	if b.branchExists && b.branch != nil {
		return *b.branch
	}
	return ""
}

func (b *SipViaBody) GetAddr(config sippy_conf.Config) (string, string) {
	if b.port == nil {
		return b.host.String(), config.DefaultPort().String()
	} else {
		return b.host.String(), b.port.String()
	}
}

func (b *SipViaBody) GetTAddr(config sippy_conf.Config) *sippy_net.HostPort {
	var host, rPort string

	if b.rPortExists && b.rPort != nil {
		rPort = *b.rPort
	} else {
		_, rPort = b.GetAddr(config)
	}
	if b.receivedExists && b.received != nil {
		host = *b.received
	} else {
		host, _ = b.GetAddr(config)
	}
	return sippy_net.NewHostPort(host, rPort)
}

func (b *SipViaBody) SetRPort(v *string) {
	b.rPortExists = true
	b.rPort = v
}

func (b *SipViaBody) SetReceived(v string) {
	b.receivedExists = true
	b.received = &v
}

func (b *SipViaBody) HasRPort() bool {
	return b.rPortExists
}
