package sippy_header

import (
	"errors"
	"strconv"

	"github.com/sippy/go-b2bua/sippy/net"
	"github.com/sippy/go-b2bua/sippy/utils"
)

type SipRAckBody struct {
	CSeq   int
	RSeq   int
	Method string
}

type SipRAck struct {
	normalName
	stringBody string
	body       *SipRAckBody
}

func NewSipRAck(rseq, cseq int, method string) *SipRAck {
	return &SipRAck{
		normalName: sipRackName,
		body:       newSipRAckBody(rseq, cseq, method),
	}
}

func newSipRAckBody(rseq, cseq int, method string) *SipRAckBody {
	return &SipRAckBody{
		RSeq:   rseq,
		CSeq:   cseq,
		Method: method,
	}
}

var sipRackName = newNormalName("RAck")

func CreateSipRAck(body string) []SipHeader {
	return []SipHeader{
		&SipRAck{
			normalName: sipRackName,
			stringBody: body,
		},
	}
}

func (s *SipRAck) parse() error {
	arr := sippy_utils.FieldsN(s.stringBody, 3)
	if len(arr) != 3 {
		return errors.New("Malformed RAck field")
	}
	rseq, err := strconv.Atoi(arr[0])
	if err != nil {
		return err
	}
	cseq, err := strconv.Atoi(arr[1])
	if err != nil {
		return err
	}
	s.body = &SipRAckBody{
		CSeq:   cseq,
		RSeq:   rseq,
		Method: arr[2],
	}
	return nil
}

func (s *SipRAck) GetCopyAsIface() SipHeader {
	return s.GetCopy()
}

func (s *SipRAck) GetBody() (*SipRAckBody, error) {
	if s.body == nil {
		if err := s.parse(); err != nil {
			return nil, err
		}
	}
	return s.body, nil
}

func (s *SipRAck) GetCopy() *SipRAck {
	tmp := *s
	if s.body != nil {
		body := *s.body
		tmp.body = &body
	}
	return &tmp
}

func (s *SipRAck) LocalStr(*sippy_net.HostPort, bool) string {
	return s.String()
}

func (s *SipRAck) String() string {
	return s.Name() + ": " + s.StringBody()
}

func (s *SipRAck) StringBody() string {
	if s.body != nil {
		return s.body.String()
	}
	return s.stringBody
}

func (s *SipRAckBody) String() string {
	return strconv.Itoa(s.RSeq) + " " + strconv.Itoa(s.CSeq) + " " + s.Method
}
