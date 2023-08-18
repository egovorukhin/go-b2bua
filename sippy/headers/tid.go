package sippy_header

import (
	"github.com/egovorukhin/go-b2bua/sippy/fmt"
)

type TID struct {
	CallId     string
	CSeq       string
	CSeqMethod string
	FromTag    string
	ToTag      string
	Branch     string
}

func NewTID(callId, cseq, cseqMethod, fromTag, toTag, viaBranch string) *TID {
	s := &TID{
		CallId:     callId,
		CSeq:       cseq,
		CSeqMethod: cseqMethod,
		FromTag:    fromTag,
		ToTag:      toTag,
		Branch:     viaBranch,
	}
	return s
}

func (s *TID) String() string {
	return sippy_fmt.Sprintf("call_id: '%s', cseq: '%s', cseq_method: '%s', from_tag: '%s', to_tag: '%s', branch: '%s'", s.CallId, s.CSeq, s.CSeqMethod, s.FromTag, s.ToTag, s.Branch)
}
