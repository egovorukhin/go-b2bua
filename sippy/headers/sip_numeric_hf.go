package sippy_header

import (
	"strconv"
)

type SipNumericHF struct {
	stringBody string
	Number     int
	parsed     bool
}

func newSipNumericHF(num int) SipNumericHF {
	return SipNumericHF{
		Number: num,
		parsed: true,
	}
}

func createSipNumericHF(body string) SipNumericHF {
	return SipNumericHF{
		stringBody: body,
		parsed:     false,
	}
}

func (s *SipNumericHF) StringBody() string {
	if s.parsed {
		return strconv.Itoa(s.Number)
	}
	return s.stringBody
}

func (s *SipNumericHF) parse() error {
	if !s.parsed {
		var err error
		s.Number, err = strconv.Atoi(s.stringBody)
		if err != nil {
			return err
		}
		s.parsed = true
	}
	return nil
}

func (s *SipNumericHF) GetBody() (*SipNumericHF, error) {
	if !s.parsed {
		if err := s.parse(); err != nil {
			return nil, err
		}
	}
	return s, nil
}
