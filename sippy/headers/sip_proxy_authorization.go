package sippy_header

type SipProxyAuthorization struct {
	*SipAuthorization
}

var sipProxyAuthorizationName normalName = newNormalName("Proxy-Authorization")

func NewSipProxyAuthorizationWithBody(body *SipAuthorizationBody) *SipProxyAuthorization {
	super := NewSipAuthorizationWithBody(body)
	super.normalName = sipProxyAuthorizationName
	return &SipProxyAuthorization{
		SipAuthorization: super,
	}
}

func NewSipProxyAuthorization(realm, nonce, uri, username, algorithm string) *SipProxyAuthorization {
	super := NewSipAuthorization(realm, nonce, uri, username, algorithm)
	super.normalName = sipProxyAuthorizationName
	return &SipProxyAuthorization{
		SipAuthorization: super,
	}
}

func CreateSipProxyAuthorization(body string) []SipHeader {
	super := createSipAuthorizationObj(body)
	super.normalName = sipProxyAuthorizationName
	return []SipHeader{&SipProxyAuthorization{
		SipAuthorization: super,
	},
	}
}
