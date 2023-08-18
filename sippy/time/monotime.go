package sippy_time

//
// #cgo linux LDFLAGS: -lrt
//
// #include <time.h>
//
// typedef struct timespec timespec_struct;
//
import "C"
import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/sippy/go-b2bua/sippy/fmt"
	"github.com/sippy/go-b2bua/sippy/math"
)

const (
	CLOCK_REALTIME  = C.CLOCK_REALTIME
	CLOCK_MONOTONIC = C.CLOCK_MONOTONIC
)

type MonoTime struct {
	monot time.Time
	realt time.Time
}

type GoroutineCtx interface {
	Apply(realt, monot time.Time) time.Duration
}

type monoGlobals struct {
	monot_max time.Time
	realt_flt sippy_math.RecFilter
}

var globals *monoGlobals

func init() {
	t, _ := newMonoTime()
	globals = &monoGlobals{
		monot_max: t.monot,
		realt_flt: sippy_math.NewRecFilter(0.99, t.realt.Sub(t.monot).Seconds()),
	}
}

func (s *monoGlobals) Apply(realt, monot time.Time) time.Duration {
	diff_flt := s.realt_flt.Apply(realt.Sub(monot).Seconds())
	if s.monot_max.Before(monot) {
		s.monot_max = monot
	}
	return time.Duration(diff_flt * float64(time.Second))
}

func NewMonoTimeFromString(s string) (*MonoTime, error) {
	parts := strings.SplitN(s, "-", 2)
	realt0, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, err
	}
	realt := FloatToTime(realt0)
	if len(parts) == 1 {
		return NewMonoTime1(realt)
	}
	monot0, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, err
	}
	monot := FloatToTime(monot0)
	return NewMonoTime2(monot, realt), nil
}

func NewMonoTime1(realt time.Time) (*MonoTime, error) {
	monot := TimeToFloat(realt) - globals.realt_flt.GetLastval()
	s := &MonoTime{
		realt: realt,
		monot: FloatToTime(monot),
	}
	if s.monot.After(globals.monot_max) {
		var ts C.timespec_struct
		if res, _ := C.clock_gettime(C.CLOCK_REALTIME, &ts); res != 0 {
			return nil, errors.New("Cannot read realtime clock")
		}
		monot_now := time.Unix(int64(ts.tv_sec), int64(ts.tv_nsec))
		if monot_now.After(globals.monot_max) {
			globals.monot_max = monot_now
		}
		s.monot = globals.monot_max
	}
	return s, nil
}

func NewMonoTime2(monot time.Time, realt time.Time) *MonoTime {
	return &MonoTime{
		monot: monot,
		realt: realt,
	}
}

func newMonoTime() (s *MonoTime, err error) {
	var ts C.timespec_struct

	if res, _ := C.clock_gettime(C.CLOCK_MONOTONIC, &ts); res != 0 {
		return nil, errors.New("Cannot read monolitic clock")
	}
	monot := time.Unix(int64(ts.tv_sec), int64(ts.tv_nsec))

	if res, _ := C.clock_gettime(C.CLOCK_REALTIME, &ts); res != 0 {
		return nil, errors.New("Cannot read realtime clock")
	}
	realt := time.Unix(int64(ts.tv_sec), int64(ts.tv_nsec))
	return &MonoTime{
		monot: monot,
		realt: realt,
	}, nil
}

func ClockGettime(cid C.clockid_t) (time.Time, error) {
	var ts C.timespec_struct

	if res, _ := C.clock_gettime(cid, &ts); res != 0 {
		return time.Unix(0, 0), errors.New("Cannot read clock")
	}
	return time.Unix(int64(ts.tv_sec), int64(ts.tv_nsec)), nil
}

func NewMonoTime() (s *MonoTime, err error) {
	t, err := newMonoTime()
	if err != nil {
		return nil, err
	}
	diff_flt := globals.Apply(t.realt, t.monot)
	t.realt = t.monot.Add(diff_flt)
	return t, nil
}

func (s *MonoTime) Ftime() string {
	t := RoundTime(s.realt).UTC()
	return sippy_fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d+00", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func (s *MonoTime) Fptime() string {
	t := s.realt.UTC()
	return sippy_fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d.%06d+00", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000)
}

func (s *MonoTime) Monot() time.Time {
	return s.monot
}

func (s *MonoTime) Realt() time.Time {
	return s.realt
}

func (s *MonoTime) Add(d time.Duration) *MonoTime {
	return NewMonoTime2(s.monot.Add(d), s.realt.Add(d))
}

func (s *MonoTime) Sub(t *MonoTime) time.Duration {
	return s.monot.Sub(t.monot)
}

func (s *MonoTime) After(t *MonoTime) bool {
	return s.monot.After(t.monot)
}

func (s *MonoTime) Before(t *MonoTime) bool {
	return s.monot.Before(t.monot)
}

func (s *MonoTime) OffsetFromNow() (time.Duration, error) {
	var ts C.timespec_struct

	if res, _ := C.clock_gettime(C.CLOCK_MONOTONIC, &ts); res != 0 {
		return 0, errors.New("Cannot read monolitic clock")
	}
	monot_now := time.Unix(int64(ts.tv_sec), int64(ts.tv_nsec))
	return monot_now.Sub(s.monot), nil
}

func (s *MonoTime) GetOffsetCopy(offset time.Duration) *MonoTime {
	return &MonoTime{
		monot: s.monot.Add(offset),
		realt: s.realt.Add(offset),
	}
}
