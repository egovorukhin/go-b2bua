package sippy

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

func extractToNextToken(s string, match string, invert bool) (string, string) {
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
	OtherParams   string
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
	ul := &UpdateLookupOpts{
		RemoteIP:   arr[0],
		RemotePort: arr[1],
	}
	arr = sippy_utils.FieldsN(arr[2], 2)
	ul.FromTag = arr[0]
	if len(arr) > 1 {
		arr2 := sippy_utils.FieldsN(arr[1], 3)
		switch len(arr2) {
		case 1:
			ul.ToTag = arr2[0]
		case 2:
			ul.NotifySocket, ul.NotifyTag = arr2[0], arr2[1]
		default:
			ul.ToTag, ul.NotifySocket, ul.NotifyTag = arr2[0], arr2[1], arr2[2]
		}
	}
	for len(s) > 0 {
		var val string
		if s[0] == 'R' {
			val, s = extractToNextToken(s[1:], "1234567890.", false)
			val = strings.TrimSpace(val)
			if len(val) > 0 {
				ul.DestinationIP = val
			}
		}
		switch s[0] {
		case 'L':
			val, s = extractToNextToken(s[1:], "1234567890.", false)
			val = strings.TrimSpace(val)
			if len(val) > 0 {
				ul.LocalIP = val
			}
		case 'c':
			val, s = extractToNextToken(s[1:], "1234567890,", false)
			val = strings.TrimSpace(val)
			if len(val) > 0 {
				ul.Codecs = strings.Split(val, ",")
			}
		default:
			val, s = extractToNextToken(s, "cR", true)
			if len(val) > 0 {
				ul.OtherParams += val
			}
		}
	}
	return ul, nil
}

func (ul *UpdateLookupOpts) Getstr(call_id string, swaptags, skipnotify bool) (string, error) {
	var s string
	if ul.DestinationIP != "" {
		s += "R" + ul.DestinationIP
	}
	if ul.LocalIP != "" {
		s += "L" + ul.LocalIP
	}
	if ul.Codecs != nil {
		s += "c" + strings.Join(ul.Codecs, ",")
	}
	s += ul.OtherParams
	s += " " + call_id
	if ul.RemoteIP != "" {
		s += " " + ul.RemoteIP
	}
	if ul.RemotePort != "" {
		s += " " + ul.RemotePort
	}
	FromTag, ToTag := ul.FromTag, ul.ToTag
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
		if ul.NotifySocket != "" {
			s += " " + ul.NotifySocket
		}
		if ul.NotifyTag != "" {
			s += " " + ul.NotifyTag
		}
	}
	return s, nil
}

type RtpProxyCmd struct {
	Type        byte
	ULOpts      *UpdateLookupOpts
	CommandOpts string
	CallId      string
	Args        string
	Nretr       int
}

func NewRtpProxyCmd(cmd string) (*RtpProxyCmd, error) {
	rpc := &RtpProxyCmd{
		Type: strings.ToUpper(cmd[:1])[0],
	}
	switch rpc.Type {
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
		var commandOpts, args string
		arr := sippy_utils.FieldsN(cmd, 3)
		if len(arr) != 3 {
			return nil, errors.New("The command must have at least three parts")
		}
		commandOpts, rpc.CallId, args = arr[0], arr[1], arr[2]
		switch rpc.Type {
		case 'U':
			fallthrough
		case 'L':
			var err error
			rpc.ULOpts, err = NewUpdateLookupOpts(commandOpts[1:], args)
			if err != nil {
				return nil, err
			}
		default:
			rpc.Args = args
			rpc.CommandOpts = commandOpts[1:]
		}
	case 'G':
		if !unicode.IsSpace([]rune(cmd)[1]) {
			cparts := sippy_utils.FieldsN(cmd[1:], 2)
			if len(cparts) > 1 {
				rpc.CommandOpts, rpc.Args = cparts[0], cparts[1]
			} else {
				rpc.CommandOpts = cparts[0]
			}
		} else {
			rpc.Args = strings.TrimSpace(cmd[1:])
		}
	default:
		rpc.CommandOpts = cmd[1:]
	}
	return rpc, nil
}

func (rpc *RtpProxyCmd) String() string {
	s := string([]byte{rpc.Type})
	if rpc.ULOpts != nil {
		_s, err := rpc.ULOpts.Getstr(rpc.CallId, false, false)
		if err != nil {
			panic(err)
		}
		s += _s
	} else {
		if rpc.CommandOpts != "" {
			s += rpc.CommandOpts
		}
		if rpc.CallId != "" {
			s += " " + rpc.CallId
		}
	}
	if rpc.Args != "" {
		s += " " + rpc.Args
	}
	return s
}

type RtppStats struct {
	spookyPrefix  string
	allNames      []string
	Verbose       bool
	dict          map[string]int64
	dictLock      sync.Mutex
	totalDuration float64
}

func NewRtppStats(snames []string) *RtppStats {
	s := &RtppStats{
		Verbose:      false,
		spookyPrefix: "",
		dict:         make(map[string]int64),
		allNames:     snames,
	}
	for _, sname := range snames {
		if sname != "total_duration" {
			s.dict[s.spookyPrefix+sname] = 0
		}
	}
	return s
}

func (s *RtppStats) AllNames() []string {
	return s.allNames
}

/*
def __iadd__(s, other):

	for sname in s.all_names:
	    aname = s.spookyPrefix + sname
	    s.__dict__[aname] += other.__dict__[aname]
	return s
*/
func (s *RtppStats) ParseAndAdd(rstr string) error {
	rparts := sippy_utils.FieldsN(rstr, len(s.allNames))
	for i, name := range s.allNames {
		if name == "total_duration" {
			rval, err := strconv.ParseFloat(rparts[i], 64)
			if err != nil {
				return err
			}
			s.totalDuration += rval
		} else {
			rval, err := strconv.ParseInt(rparts[i], 10, 64)
			if err != nil {
				return err
			}
			aname := s.spookyPrefix + s.allNames[i]
			s.dictLock.Lock()
			s.dict[aname] += rval
			s.dictLock.Unlock()
		}
	}
	return nil
}

func (s *RtppStats) String() string {
	rvals := make([]string, 0, len(s.allNames))
	for _, sname := range s.allNames {
		var rval string

		if sname == "total_duration" {
			rval = strconv.FormatFloat(s.totalDuration, 'f', -1, 64)
		} else {
			aname := s.spookyPrefix + sname
			s.dictLock.Lock()
			rval = strconv.FormatInt(s.dict[aname], 10)
			s.dictLock.Unlock()
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
