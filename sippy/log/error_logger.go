package sippy_log

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/fmt"
)

type ErrorLogger interface {
	ErrorAndTraceback(interface{})
	Error(...interface{})
	Debug(...interface{})
	Errorf(string, ...interface{})
	Debugf(string, ...interface{})
}

type errorLogger struct {
	lock sync.Mutex
}

func NewErrorLogger() *errorLogger {
	return &errorLogger{}
}

func (l *errorLogger) ErrorAndTraceback(err interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Error(err)
	buf := make([]byte, 16384)
	n := runtime.Stack(buf, false)
	for _, s := range strings.Split(string(buf[:n]), "\n") {
		l.Error(s)
	}
}

func (l *errorLogger) Debug(params ...interface{}) {
	l.write("DEBUG:", params...)
}

func (l *errorLogger) Debugf(format string, params ...interface{}) {
	l.write("DEBUG:", sippy_fmt.Sprintf(format, params...))
}

func (l *errorLogger) Error(params ...interface{}) {
	l.write("ERROR:", params...)
}

func (l *errorLogger) Errorf(format string, params ...interface{}) {
	l.write("ERROR:", sippy_fmt.Sprintf(format, params...))
}

func (*errorLogger) Reopen() {
}

func (*errorLogger) write(prefix string, params ...interface{}) {
	t := time.Now()
	buf := []interface{}{FormatDate(t), " ", prefix}
	for _, it := range params {
		buf = append(buf, " ", it)
	}
	buf = append(buf, "\n")
	fmt.Fprint(os.Stderr, buf...)
}
