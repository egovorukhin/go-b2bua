package sippy_header

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/security"
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type SipAuthorizationBody struct {
	username    string
	realm       string
	nonce       string
	uri         string
	response    string
	qop         string
	nc          string
	cnonce      string
	algorithm   string
	opaque      string
	otherparams string
}

type SipAuthorization struct {
	normalName
	body       *SipAuthorizationBody
	stringBody string
}

var sipAuthorizationName normalName = newNormalName("Authorization")

func NewSipAuthorizationWithBody(body *SipAuthorizationBody) *SipAuthorization {
	return &SipAuthorization{
		normalName: sipAuthorizationName,
		body:       body,
	}
}

func NewSipAuthorization(realm, nonce, uri, username, algorithm string) *SipAuthorization {
	return &SipAuthorization{
		normalName: sipAuthorizationName,
		body: &SipAuthorizationBody{
			realm:     realm,
			nonce:     nonce,
			uri:       uri,
			username:  username,
			algorithm: algorithm,
		},
	}
}

func CreateSipAuthorization(body string) []SipHeader {
	s := createSipAuthorizationObj(body)
	return []SipHeader{s}
}

func createSipAuthorizationObj(body string) *SipAuthorization {
	return &SipAuthorization{
		normalName: sipAuthorizationName,
		stringBody: body,
	}
}

func newSipAuthorizationBody(realm, nonce, uri, username, algorithm string) *SipAuthorizationBody {
	return &SipAuthorizationBody{
		realm:     realm,
		nonce:     nonce,
		uri:       uri,
		username:  username,
		algorithm: algorithm,
	}
}

func parseSipAuthorizationBody(body string) (*SipAuthorizationBody, error) {
	s := &SipAuthorizationBody{}
	arr := sippy_utils.FieldsN(body, 2)
	if len(arr) != 2 {
		return nil, errors.New("Error parsing authorization (1)")
	}
	for _, param := range strings.Split(arr[1], ",") {
		kv := strings.SplitN(strings.TrimSpace(param), "=", 2)
		if len(kv) != 2 {
			return nil, errors.New("Error parsing authorization (2)")
		}
		name, value := kv[0], kv[1]
		switch strings.ToLower(name) {
		case "username":
			s.username = strings.Trim(value, "\"")
		case "uri":
			s.uri = strings.Trim(value, "\"")
		case "realm":
			s.realm = strings.Trim(value, "\"")
		case "nonce":
			s.nonce = strings.Trim(value, "\"")
		case "response":
			s.response = strings.Trim(value, "\"")
		case "qop":
			s.qop = strings.Trim(value, "\"")
		case "cnonce":
			s.cnonce = strings.Trim(value, "\"")
		case "nc":
			s.nc = strings.Trim(value, "\"")
		case "algorithm":
			s.algorithm = strings.Trim(value, "\"")
		case "opaque":
			s.opaque = strings.Trim(value, "\"")
		default:
			s.otherparams += "," + param
		}
	}
	return s, nil
}

func (b *SipAuthorizationBody) String() string {
	rval := "Digest username=\"" + b.username + "\",realm=\"" + b.realm + "\",nonce=\"" + b.nonce +
		"\",uri=\"" + b.uri + "\",response=\"" + b.response + "\""
	if b.algorithm != "" {
		rval += ",algorithm=\"" + b.algorithm + "\""
	}
	if b.qop != "" {
		rval += ",nc=\"" + b.nc + "\",cnonce=\"" + b.cnonce + "\",qop=" + b.qop
	}
	if b.opaque != "" {
		rval += ",opaque=\"" + b.opaque + "\""
	}
	return rval + b.otherparams
}

func (b *SipAuthorizationBody) GetCopy() *SipAuthorizationBody {
	rval := *b
	return &rval
}

func (b *SipAuthorizationBody) GetUsername() string {
	return b.username
}

func (b *SipAuthorizationBody) Verify(passwd, method, entityBody string) bool {
	alg := sippy_security.GetAlgorithm(b.algorithm)
	if alg == nil {
		return false
	}
	HA1 := DigestCalcHA1(alg, b.algorithm, b.username, b.realm, passwd, b.nonce, b.cnonce)
	return b.VerifyHA1(HA1, method, entityBody)
}

func (b *SipAuthorizationBody) VerifyHA1(HA1, method, entityBody string) bool {
	now, _ := sippy_time.NewMonoTime()
	alg := sippy_security.GetAlgorithm(b.algorithm)
	if alg == nil {
		return false
	}
	if b.qop != "" && b.qop != "auth" {
		return false
	}
	if !sippy_security.HashOracle.ValidateChallenge(b.nonce, alg.Mask, now.Monot()) {
		return false
	}
	response := DigestCalcResponse(alg, HA1, b.nonce, b.nc, b.cnonce, b.qop, method, b.uri, entityBody)
	return response == b.response
}

func (a *SipAuthorization) String() string {
	return a.Name() + ": " + a.StringBody()
}

func (a *SipAuthorization) LocalStr(*sippy_net.HostPort, bool) string {
	return a.String()
}

func (a *SipAuthorization) StringBody() string {
	if a.body != nil {
		return a.body.String()
	}
	return a.stringBody
}

func (a *SipAuthorization) GetCopy() *SipAuthorization {
	if a == nil {
		return nil
	}
	var rval SipAuthorization = *a
	if a.body != nil {
		rval.body = a.body.GetCopy()
	}
	return &rval
}

func (a *SipAuthorization) parse() error {
	body, err := parseSipAuthorizationBody(a.stringBody)
	if err != nil {
		return err
	}
	a.body = body
	return nil
}

func (a *SipAuthorization) GetBody() (*SipAuthorizationBody, error) {
	if a.body == nil {
		if err := a.parse(); err != nil {
			return nil, err
		}
	}
	return a.body, nil
}

func (a *SipAuthorization) GetUsername() (string, error) {
	body, err := a.GetBody()
	if err != nil {
		return "", err
	}
	return body.GetUsername(), nil
}

func (b *SipAuthorizationBody) GenResponse(password, method, entityBody string) {
	alg := sippy_security.GetAlgorithm(b.algorithm)
	if alg == nil {
		return
	}
	HA1 := DigestCalcHA1(alg, b.algorithm, b.username, b.realm, password,
		b.nonce, b.cnonce)
	b.response = DigestCalcResponse(alg, HA1, b.nonce,
		b.nc, b.cnonce, b.qop, method, b.uri, entityBody)
}

func (a *SipAuthorization) GetCopyAsIface() SipHeader {
	return a.GetCopy()
}

func DigestCalcHA1(alg *sippy_security.Algorithm, pszAlg, pszUserName, pszRealm, pszPassword, pszNonce, pszCNonce string) string {
	s := pszUserName + ":" + pszRealm + ":" + pszPassword
	hash := alg.NewHash()
	hash.Write([]byte(s))
	HA1 := hash.Sum(nil)
	if strings.HasSuffix(pszAlg, "-sess") {
		s2 := []byte(hex.EncodeToString(HA1))
		s2 = append(s2, []byte(":"+pszNonce+":"+pszCNonce)...)
		hash = alg.NewHash()
		hash.Write([]byte(s2))
		HA1 = hash.Sum(nil)
	}
	return hex.EncodeToString(HA1)
}

func DigestCalcResponse(alg *sippy_security.Algorithm, HA1, pszNonce, pszNonceCount, pszCNonce, pszQop, pszMethod, pszDigestUri, pszHEntity string) string {
	s := pszMethod + ":" + pszDigestUri
	if pszQop == "auth-int" {
		hash := alg.NewHash()
		hash.Write([]byte(pszHEntity))
		sum := hash.Sum(nil)
		s += ":" + hex.EncodeToString(sum[:])
	}
	hash := alg.NewHash()
	hash.Write([]byte(s))
	sum := hash.Sum(nil)
	HA2 := hex.EncodeToString(sum[:])
	s = HA1 + ":" + pszNonce + ":"
	if pszNonceCount != "" && pszCNonce != "" { // pszQop:
		s += pszNonceCount + ":" + pszCNonce + ":" + pszQop + ":"
	}
	s += HA2
	hash = alg.NewHash()
	hash.Write([]byte(s))
	sum = hash.Sum(nil)
	return hex.EncodeToString(sum[:])
}
