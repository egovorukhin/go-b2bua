package sippy_header

type SipProxyAuthenticate struct {
	*SipWWWAuthenticate
}

var sipProxyAuthenticateName normalName = newNormalName("Proxy-Authenticate")

func CreateSipProxyAuthenticate(body string) []SipHeader {
	super := createSipWWWAuthenticateObj(body)
	super.normalName = sipProxyAuthenticateName
	super.aClass = func(body *SipAuthorizationBody) SipHeader { return NewSipProxyAuthorizationWithBody(body) }
	return []SipHeader{
		&SipProxyAuthenticate{
			SipWWWAuthenticate: super,
		},
	}
}

func (s *SipProxyAuthenticate) GetCopy() *SipProxyAuthenticate {
	cself := &SipProxyAuthenticate{
		SipWWWAuthenticate: s.SipWWWAuthenticate.GetCopy(),
	}
	cself.normalName = sipProxyAuthenticateName
	return cself
}

func (s *SipProxyAuthenticate) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}
