package sippy_log

import (
	"github.com/egovorukhin/go-b2bua/sippy/time"
	"os"
	"strconv"
	"syscall"
	"time"
)

type SipLogger interface {
	Write(rtime *sippy_time.MonoTime, callId string, msg string)
}

type sipLogger struct {
	fname string
	id    string
	fd    *os.File
}

func NewSipLogger(id, fname string) (SipLogger, error) {
	s := &sipLogger{
		fname: fname,
		id:    id,
	}
	err := s.Reopen()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func _fmt0Xd(v, width int) string {
	ret := strconv.Itoa(v)
	for len(ret) < width {
		ret = "0" + ret
	}
	return ret
}

func FormatDate(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + t.Month().String()[:3] + " " +
		_fmt0Xd(t.Hour(), 2) + ":" + _fmt0Xd(t.Minute(), 2) + ":" + _fmt0Xd(t.Second(), 2) + "." +
		_fmt0Xd(t.Nanosecond()/1000000, 3)
}

func (s *sipLogger) Write(rtime *sippy_time.MonoTime, call_id string, msg string) {
	var t time.Time
	if rtime != nil {
		t = rtime.Realt()
	} else {
		t = time.Now()
	}
	//buf := fmt.Sprintf("%d %s %02d:%02d:%06.3f/%s/%s: %s\n",
	buf := FormatDate(t) + "/" + call_id + "/" + s.id + ": " + msg
	fileno := int(s.fd.Fd())
	syscall.Flock(fileno, syscall.LOCK_EX)
	defer syscall.Flock(fileno, syscall.LOCK_UN)
	s.fd.Write([]byte(buf))
}

func (s *sipLogger) Reopen() error {
	var err error
	if s.fd == nil {
		s.fd.Close()
		s.fd = nil
	}
	s.fd, err = os.OpenFile(s.fname, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	return err
}
