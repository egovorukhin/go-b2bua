package sippy_sdp

import (
	"crypto/rand"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/egovorukhin/go-b2bua/sippy/net"
)

var _sdp_session_id int64

func init() {
	buf := make([]byte, 6)
	rand.Read(buf)
	for i := 0; i < len(buf); i++ {
		_sdp_session_id |= int64(buf[i]) << (uint(i) * 8)
	}
}

type SdpOrigin struct {
	username     string
	session_id   string
	version      int64
	network_type string
	address_type string
	address      string
}

func ParseSdpOrigin(body string) (*SdpOrigin, error) {
	arr := strings.Fields(body)
	if len(arr) != 6 {
		return nil, errors.New("Malformed field: " + body)
	}
	version, err := strconv.ParseInt(arr[2], 10, 64)
	if err != nil {
		return nil, err
	}
	return &SdpOrigin{
		username:     arr[0],
		session_id:   arr[1],
		version:      version,
		network_type: arr[3],
		address_type: arr[4],
		address:      arr[5],
	}, nil
}

func NewSdpOrigin() *SdpOrigin {
	// RFC4566
	// *******
	// For privacy reasons, it is sometimes desirable to obfuscate the
	// username and IP address of the session originator.  If this is a
	// concern, an arbitrary <username> and private <unicast-address> MAY be
	// chosen to populate the "o=" field, provided that these are selected
	// in a manner that does not affect the global uniqueness of the field.
	// *******
	sid := atomic.AddInt64(&_sdp_session_id, 1)
	return &SdpOrigin{
		username:     "-",
		session_id:   strconv.FormatInt(sid, 10),
		network_type: "IN",
		address_type: "IP4",
		address:      "192.0.2.1", // 192.0.2.0/24 (TEST-NET-1)
		version:      sid,
	}
}

func NewSdpOriginWithAddress(address string) (*SdpOrigin, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, errors.New("The address is not IP address: " + address)
	}
	address_type := "IP4"
	if !sippy_net.IsIP4(ip) {
		address_type = "IP6"
	}
	sid := atomic.AddInt64(&_sdp_session_id, 1)
	s := &SdpOrigin{
		username:     "-",
		session_id:   strconv.FormatInt(sid, 10),
		network_type: "IN",
		address_type: address_type,
		address:      address,
		version:      sid,
	}
	return s, nil
}

func (s *SdpOrigin) String() string {
	version := strconv.FormatInt(s.version, 10)
	return strings.Join([]string{s.username, s.session_id, version, s.network_type, s.address_type, s.address}, " ")
}

func (s *SdpOrigin) LocalStr(hostPort *sippy_net.HostPort) string {
	version := strconv.FormatInt(s.version, 10)
	return strings.Join([]string{s.username, s.session_id, version, s.network_type, s.address_type, s.address}, " ")
}

func (s *SdpOrigin) GetCopy() *SdpOrigin {
	if s == nil {
		return nil
	}
	var ret SdpOrigin = *s
	return &ret
}

func (s *SdpOrigin) IncVersion() {
	s.version++
}

func (s *SdpOrigin) GetSessionId() string {
	return s.session_id
}

func (s *SdpOrigin) GetVersion() int64 {
	return s.version
}

func (s *SdpOrigin) SetAddress(addr string) {
	s.address = addr
}

func (s *SdpOrigin) SetAddressType(t string) {
	s.address_type = t
}

func (s *SdpOrigin) SetNetworkType(t string) {
	s.network_type = t
}
