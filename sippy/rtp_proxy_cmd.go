package sippy

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/sippy/go-b2bua/sippy/utils"
)

func extract_to_next_token(s string, match string, invert bool) (string, string) {
	i := 0
	for i < len(s) {
		if (!invert && strings.IndexByte(match, s[i]) == -1) || (invert && strings.IndexByte(match, s[i]) != -1) {
			break
		}
		i++
	}
	if i == 0 {
		return "", s
	}
	if i == len(s) {
		return s, ""
	}
	return s[:i], s[i:]
}

type UpdateLookupOpts struct {
	DestinationIP string
	LocalIP       string
	Codecs        []string
	Otherparams   string
	RemoteIP      string
	RemotePort    string
	FromTag       string
	ToTag         string
	NotifySocket  string
	NotifyTag     string
}

func NewUpdateLookupOpts(s, args string) (*UpdateLookupOpts, error) {
	arr := sippy_utils.FieldsN(args, 3)
	if len(arr) != 3 {
		return nil, errors.New("The lookup opts must have at least three arguments")
	}
	s := &UpdateLookupOpts{
		RemoteIP:   arr[0],
		RemotePort: arr[1],
	}
	arr = sippy_utils.FieldsN(arr[2], 2)
	s.FromTag = arr[0]
	if len(arr) > 1 {
		arr2 := sippy_utils.FieldsN(arr[1], 3)
		switch len(arr2) {
		case 1:
			s.ToTag = arr2[0]
		case 2:
			s.NotifySocket, s.NotifyTag = arr2[0], arr2[1]
		default:
			s.ToTag, s.NotifySocket, s.NotifyTag = arr2[0], arr2[1], arr2[2]
		}
	}
	for len(s) > 0 {
		var val string
		if s[0] == 'R' {
			val, s = extract_to_next_token(s[1:], "1234567890.", false)
			val = strings.TrimSpace(val)
			if len(val) > 0 {
				s.DestinationIP = val
			}
		}
		switch s[0] {
		case 'L':
			val, s = extract_to_next_token(s[1:], "1234567890.", false)
			val = strings.TrimSpace(val)
			if len(val) > 0 {
				s.LocalIP = val
			}
		case 'c':
			val, s = extract_to_next_token(s[1:], "1234567890,", false)
			val = strings.TrimSpace(val)
			if len(val) > 0 {
				s.Codecs = strings.Split(val, ",")
			}
		default:
			val, s = extract_to_next_token(s, "cR", true)
			if len(val) > 0 {
				s.Otherparams += val
			}
		}
	}
	return s, nil
}

func (s *UpdateLookupOpts) Getstr(call_id string, swaptags, skipnotify bool) (string, error) {
	s := ""
	if s.DestinationIP != "" {
		s += "R" + s.DestinationIP
	}
	if s.LocalIP != "" {
		s += "L" + s.LocalIP
	}
	if s.Codecs != nil {
		s += "c" + strings.Join(s.Codecs, ",")
	}
	s += s.Otherparams
	s += " " + call_id
	if s.RemoteIP != "" {
		s += " " + s.RemoteIP
	}
	if s.RemotePort != "" {
		s += " " + s.RemotePort
	}
	FromTag, ToTag := s.FromTag, s.ToTag
	if swaptags {
		if ToTag == "" {
			return "", errors.New("UpdateLookupOpts::Getstr(swaptags = True): ToTag is not set")
		}
		ToTag, FromTag = FromTag, ToTag
	}
	if FromTag != "" {
		s += " " + FromTag
	}
	if ToTag != "" {
		s += " " + ToTag
	}
	if !skipnotify {
		if s.NotifySocket != "" {
			s += " " + s.NotifySocket
		}
		if s.NotifyTag != "" {
			s += " " + s.NotifyTag
		}
	}
	return s, nil
}

type Rtp_proxy_cmd struct {
	Type        byte
	ULOpts      *UpdateLookupOpts
	CommandOpts string
	CallId      string
	Args        string
	Nretr       int
}

func NewRtp_proxy_cmd(cmd string) (*Rtp_proxy_cmd, error) {
	s := &Rtp_proxy_cmd{
		Type: strings.ToUpper(cmd[:1])[0],
	}
	switch s.Type {
	case 'U':
		fallthrough
	case 'L':
		fallthrough
	case 'D':
		fallthrough
	case 'P':
		fallthrough
	case 'S':
		fallthrough
	case 'R':
		fallthrough
	case 'C':
		fallthrough
	case 'Q':
		var command_opts, args string
		arr := sippy_utils.FieldsN(cmd, 3)
		if len(arr) != 3 {
			return nil, errors.New("The command must have at least three parts")
		}
		command_opts, s.CallId, args = arr[0], arr[1], arr[2]
		switch s.Type {
		case 'U':
			fallthrough
		case 'L':
			var err error
			s.ULOpts, err = NewUpdateLookupOpts(command_opts[1:], args)
			if err != nil {
				return nil, err
			}
		default:
			s.Args = args
			s.CommandOpts = command_opts[1:]
		}
	case 'G':
		if !unicode.IsSpace([]rune(cmd)[1]) {
			cparts := sippy_utils.FieldsN(cmd[1:], 2)
			if len(cparts) > 1 {
				s.CommandOpts, s.Args = cparts[0], cparts[1]
			} else {
				s.CommandOpts = cparts[0]
			}
		} else {
			s.Args = strings.TrimSpace(cmd[1:])
		}
	default:
		s.CommandOpts = cmd[1:]
	}
	return s, nil
}

func (s *Rtp_proxy_cmd) String() string {
	s := string([]byte{s.Type})
	if s.ULOpts != nil {
		_s, err := s.ULOpts.Getstr(s.CallId, false, false)
		if err != nil {
			panic(err)
		}
		s += _s
	} else {
		if s.CommandOpts != "" {
			s += s.CommandOpts
		}
		if s.CallId != "" {
			s += " " + s.CallId
		}
	}
	if s.Args != "" {
		s += " " + s.Args
	}
	return s
}

type Rtpp_stats struct {
	spookyprefix   string
	all_names      []string
	Verbose        bool
	dict           map[string]int64
	dict_lock      sync.Mutex
	total_duration float64
}

func NewRtpp_stats(snames []string) *Rtpp_stats {
	s := &Rtpp_stats{
		Verbose:      false,
		spookyprefix: "",
		dict:         make(map[string]int64),
		all_names:    snames,
	}
	for _, sname := range snames {
		if sname != "total_duration" {
			s.dict[s.spookyprefix+sname] = 0
		}
	}
	return s
}

func (s *Rtpp_stats) AllNames() []string {
	return s.all_names
}

/*
def __iadd__(s, other):

	for sname in s.all_names:
	    aname = s.spookyprefix + sname
	    s.__dict__[aname] += other.__dict__[aname]
	return s
*/
func (s *Rtpp_stats) ParseAndAdd(rstr string) error {
	rparts := sippy_utils.FieldsN(rstr, len(s.all_names))
	for i, name := range s.all_names {
		if name == "total_duration" {
			rval, err := strconv.ParseFloat(rparts[i], 64)
			if err != nil {
				return err
			}
			s.total_duration += rval
		} else {
			rval, err := strconv.ParseInt(rparts[i], 10, 64)
			if err != nil {
				return err
			}
			aname := s.spookyprefix + s.all_names[i]
			s.dict_lock.Lock()
			s.dict[aname] += rval
			s.dict_lock.Unlock()
		}
	}
	return nil
}

func (s *Rtpp_stats) String() string {
	rvals := make([]string, 0, len(s.all_names))
	for _, sname := range s.all_names {
		var rval string

		if sname == "total_duration" {
			rval = strconv.FormatFloat(s.total_duration, 'f', -1, 64)
		} else {
			aname := s.spookyprefix + sname
			s.dict_lock.Lock()
			rval = strconv.FormatInt(s.dict[aname], 10)
			s.dict_lock.Unlock()
		}
		if s.Verbose {
			rval = sname + "=" + rval
		}
		rvals = append(rvals, rval)
	}
	return strings.Join(rvals, " ")
}

/*
if __name__ == '__main__':
    rc = Rtp_proxy_cmd('G nsess_created total_duration')
    print(rc)
    print(rc.args)
    print(rc.command_opts)
    rc = Rtp_proxy_cmd('Gv nsess_created total_duration')
    print(rc)
    print(rc.args)
    print(rc.command_opts)
*/
