package sippy_header

import (
	"strconv"

	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type SipCSeqBody struct {
	CSeq   int
	Method string
}

type SipCSeq struct {
	normalName
	stringBody string
	body       *SipCSeqBody
}

func NewSipCSeq(cseq int, method string) *SipCSeq {
	return &SipCSeq{
		normalName: sipCseqName,
		body:       newSipCSeqBody(cseq, method),
	}
}

func newSipCSeqBody(cseq int, method string) *SipCSeqBody {
	return &SipCSeqBody{
		CSeq:   cseq,
		Method: method,
	}
}

var sipCseqName normalName = newNormalName("CSeq")

func CreateSipCSeq(body string) []SipHeader {
	return []SipHeader{
		&SipCSeq{
			normalName: sipCseqName,
			stringBody: body,
		},
	}
}

func (s *SipCSeq) parse() error {
	arr := sippy_utils.FieldsN(s.stringBody, 2)
	cseq, err := strconv.Atoi(arr[0])
	if err != nil {
		return err
	}
	body := &SipCSeqBody{
		CSeq: cseq,
	}
	if len(arr) == 2 {
		body.Method = arr[1]
	}
	s.body = body
	return nil
}

func (s *SipCSeq) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipCSeq) GetBody() (*SipCSeqBody, error) {
	if s.body == nil {
		if err := s.parse(); err != nil {
			return nil, err
		}
	}
	return s.body, nil
}

func (s *SipCSeq) GetCopy() *SipCSeq {
	tmp := *s
	if s.body != nil {
		body := *s.body
		tmp.body = &body
	}
	return &tmp
}

func (s *SipCSeq) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipCSeq) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipCSeq) StringBody() string {
	if s.body != nil {
		return s.body.String()
	}
	return s.stringBody
}

func (s *SipCSeqBody) String() string {
	return strconv.Itoa(s.CSeq) + " " + s.Method
}
