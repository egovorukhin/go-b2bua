package sippy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/egovorukhin/go-b2bua/sippy/net"
	"github.com/egovorukhin/go-b2bua/sippy/types"
)

type msgBody struct {
	mtype          string
	sdp            sippy_types.Sdp
	string_content string
	needs_update   bool
	parsed         bool
}

func NewMsgBody(content, mtype string) *msgBody {
	return &msgBody{
		mtype:          mtype,
		sdp:            nil,
		string_content: content,
		needs_update:   true,
		parsed:         false,
	}
}

func (s *msgBody) GetSdp() (sippy_types.Sdp, error) {
	err := s.parse()
	if err != nil {
		return nil, err
	}
	if s.sdp == nil {
		return nil, errors.New("Not an SDP message")
	}
	return s.sdp, nil
}

func (s *msgBody) parse() error {
	if s.parsed {
		return nil
	}
	if strings.HasPrefix(s.mtype, "multipart/mixed;") {
		arr := strings.SplitN(s.mtype, ";", 2)
		mtheaders := arr[1]
		var mth_boundary *string = nil
		for _, s := range strings.Split(mtheaders, ";") {
			arr = strings.SplitN(s, "=", 2)
			if arr[0] == "boundary" && len(arr) == 2 {
				mth_boundary = &arr[1]
				break
			}
		}
		if mth_boundary == nil {
			return errors.New("Error parsing the multipart message")
		}
		boundary := "--" + *mth_boundary
		for _, subsection := range strings.Split(s.string_content, boundary) {
			subsection = strings.TrimSpace(subsection)
			if subsection == "" {
				continue
			}
			boff, bdel := -1, ""
			for _, bdel = range []string{"\r\n\r\n", "\r\r", "\n\n"} {
				boff = strings.Index(subsection, bdel)
				if boff != -1 {
					break
				}
			}
			if boff == -1 {
				continue
			}
			mbody := subsection[boff+len(bdel):]
			mtype := ""
			for _, line := range strings.FieldsFunc(subsection[:boff], func(c rune) bool { return c == '\n' || c == '\r' }) {
				tmp := strings.ToLower(strings.TrimSpace(line))
				if strings.HasPrefix(tmp, "content-type:") {
					arr = strings.SplitN(tmp, ":", 2)
					mtype = strings.TrimSpace(arr[1])
				}
			}
			if mtype == "" {
				continue
			}
			if mtype == "application/sdp" {
				s.mtype = mtype
				s.string_content = mbody
				break
			}
		}
	}
	if s.mtype == "application/sdp" {
		sdp, err := ParseSdpBody(s.string_content)
		if err == nil {
			s.sdp = sdp
		} else {
			return fmt.Errorf("error parsing the SDP: %s", err.Error())
		}
	}
	s.parsed = true
	return nil
}

func (s *msgBody) String() string {
	if s.sdp != nil {
		s.string_content = s.sdp.String()
	}
	return s.string_content
}

func (s *msgBody) LocalStr(local_hostport *sippy_net.HostPort) string {
	if s.sdp != nil {
		return s.sdp.LocalStr(local_hostport)
	}
	return s.String()
}

func (s *msgBody) GetCopy() sippy_types.MsgBody {
	if s == nil {
		return nil
	}
	var sdp sippy_types.Sdp
	if s.sdp != nil {
		sdp = s.sdp.GetCopy()
	}
	return &msgBody{
		mtype:          s.mtype,
		sdp:            sdp,
		string_content: s.string_content,
		needs_update:   true,
		parsed:         s.parsed,
	}
}

func (s *msgBody) GetMtype() string {
	return s.mtype
}

func (s *msgBody) NeedsUpdate() bool {
	return s.needs_update
}

func (s *msgBody) SetNeedsUpdate(v bool) {
	s.needs_update = v
}

func (s *msgBody) AppendAHeader(hdr string) {
	if s.sdp != nil {
		s.sdp.AppendAHeader(hdr)
	} else {
		s.string_content += "a=" + hdr + "\r\n"
	}
}
