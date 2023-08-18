package sippy_header

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/security"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type SipWWWAuthenticateBody struct {
	realm       *sippy_net.MyAddress
	nonce       string
	algorithm   string
	qop         []string
	otherParams []string
	opaque      string
}

type SipWWWAuthenticate struct {
	normalName
	stringBody string
	body       *SipWWWAuthenticateBody
	aClass     func(*SipAuthorizationBody) SipHeader
}

var sipWwwAuthenticateName = newNormalName("WWW-Authenticate")

func CreateSipWWWAuthenticate(body string) []SipHeader {
	return []SipHeader{createSipWWWAuthenticateObj(body)}
}

func NewSipWWWAuthenticateWithRealm(realm, algorithm string, nowMono time.Time) *SipWWWAuthenticate {
	return &SipWWWAuthenticate{
		normalName: sipWwwAuthenticateName,
		body:       newSipWWWAuthenticateBody(realm, algorithm, nowMono),
		aClass:     func(body *SipAuthorizationBody) SipHeader { return NewSipAuthorizationWithBody(body) },
	}
}

func newSipWWWAuthenticateBody(realm, algorithm string, nowMono time.Time) *SipWWWAuthenticateBody {
	s := &SipWWWAuthenticateBody{
		algorithm: algorithm,
		realm:     sippy_net.NewMyAddress(realm),
		qop:       []string{},
	}
	if algorithm != "" {
		s.qop = append(s.qop, "auth")
	}
	alg := sippy_security.GetAlgorithm(algorithm)
	if alg == nil {
		buf := make([]byte, 20)
		_, _ = rand.Read(buf)
		s.nonce = hex.EncodeToString(buf)
	} else {
		s.nonce = sippy_security.HashOracle.EmitChallenge(alg.Mask, nowMono)
	}
	return s
}

func createSipWWWAuthenticateObj(body string) *SipWWWAuthenticate {
	return &SipWWWAuthenticate{
		normalName: sipWwwAuthenticateName,
		stringBody: body,
		aClass:     func(body *SipAuthorizationBody) SipHeader { return NewSipAuthorizationWithBody(body) },
	}
}

func (s *SipWWWAuthenticate) parse() error {
	tmp := sippy_utils.FieldsN(s.stringBody, 2)
	if len(tmp) != 2 {
		return errors.New("Error parsing authentication (1)")
	}
	body := &SipWWWAuthenticateBody{}
	for _, part := range strings.Split(tmp[1], ",") {
		arr := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(arr) != 2 {
			continue
		}
		switch arr[0] {
		case "realm":
			body.realm = sippy_net.NewMyAddress(strings.Trim(arr[1], "\""))
		case "nonce":
			body.nonce = strings.Trim(arr[1], "\"")
		case "opaque":
			body.opaque = strings.Trim(arr[1], "\"")
		case "algorithm":
			body.algorithm = strings.Trim(arr[1], "\"")
		case "qop":
			qops := strings.Trim(arr[1], "\"")
			body.qop = strings.Split(qops, ",")
		default:
			body.otherParams = append(body.otherParams, part)
		}
	}
	s.body = body
	return nil
}

func (s *SipWWWAuthenticate) GetBody() (*SipWWWAuthenticateBody, error) {
	if s.body == nil {
		if err := s.parse(); err != nil {
			return nil, err
		}
	}
	return s.body, nil
}

func (s *SipWWWAuthenticate) StringBody() string {
	return s.LocalStringBody(nil)
}

func (s *SipWWWAuthenticate) String() string {
	return s.LocalStr(nil, false)
}

func (s *SipWWWAuthenticate) LocalStr(hostPort *sippy_net.HostPort, compact bool) string {
	return s.Name() + ": " + s.LocalStringBody(hostPort)
}

func (s *SipWWWAuthenticate) LocalStringBody(hostPort *sippy_net.HostPort) string {
	if s.body != nil {
		return s.body.localString(hostPort)
	}
	return s.stringBody
}

func (s *SipWWWAuthenticateBody) localString(hostPort *sippy_net.HostPort) string {
	realm := s.realm.String()
	if hostPort != nil && s.realm.IsSystemDefault() {
		realm = hostPort.Host.String()
	}
	ret := "Digest realm=\"" + realm + "\",nonce=\"" + s.nonce + "\""
	if s.algorithm != "" {
		ret += ",algorithm=\"" + s.algorithm + "\""
	}
	if s.opaque != "" {
		ret += ",opaque=\"" + s.opaque + "\""
	}
	if len(s.qop) == 1 {
		ret += ",qop=" + s.qop[0]
	} else if len(s.qop) > 1 {
		ret += ",qop=\"" + strings.Join(s.qop, ",") + "\""
	}
	if len(s.otherParams) > 0 {
		ret += "," + strings.Join(s.otherParams, ",")
	}
	return ret
}

func (s *SipWWWAuthenticateBody) GetRealm() string {
	return s.realm.String()
}

func (s *SipWWWAuthenticateBody) GetNonce() string {
	return s.nonce
}

func (s *SipWWWAuthenticate) GetCopy() *SipWWWAuthenticate {
	tmp := *s
	if s.body != nil {
		s.body = s.body.getCopy()
	}
	return &tmp
}

func (s *SipWWWAuthenticateBody) getCopy() *SipWWWAuthenticateBody {
	tmp := *s
	return &tmp
}

func (s *SipWWWAuthenticate) GenAuthHF(username, password, method, uri, entityBody string) (SipHeader, error) {
	body, err := s.GetBody()
	if err != nil {
		return nil, err
	}
	auth := newSipAuthorizationBody(body.realm.String(), body.nonce, uri, username, body.algorithm)
	if len(body.qop) > 0 {
		auth.qop = body.qop[0]
		auth.nc = "00000001"
		buf := make([]byte, 4)
		_, _ = rand.Read(buf)
		auth.cnonce = hex.EncodeToString(buf)
	}
	if body.opaque != "" {
		auth.opaque = body.opaque
	}
	auth.GenResponse(password, method, entityBody)
	return s.aClass(auth), nil
}

func (s *SipWWWAuthenticate) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipWWWAuthenticate) Algorithm() (string, error) {
	body, err := s.GetBody()
	if err != nil {
		return "", err
	}
	return body.algorithm, nil
}

func (s *SipWWWAuthenticate) SupportedAlgorithm() (bool, error) {
	body, err := s.GetBody()
	if err != nil {
		return false, err
	}
	alg := sippy_security.GetAlgorithm(body.algorithm)
	return alg != nil, nil
}
