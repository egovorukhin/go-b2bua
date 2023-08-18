package sippy

import (
	"math/rand"
	"sync"
	"time"

	"github.com/egovorukhin/go-b2bua/sippy/log"
	"github.com/egovorukhin/go-b2bua/sippy/utils"
)

type Timeout struct {
	callback      func()
	timeout       time.Duration
	logger        sippy_log.ErrorLogger
	shutdown_chan chan struct{}
	spread        float64
	nticks        int
	lock          sync.Mutex
	cb_lock       sync.Locker
	started       bool
}

func StartTimeoutWithSpread(callback func(), cb_lock sync.Locker, _timeout time.Duration, nticks int, logger sippy_log.ErrorLogger, spread float64) *Timeout {
	s := NewInactiveTimeout(callback, cb_lock, _timeout, nticks, logger)
	s.spread = spread
	s.Start()
	return s
}

func StartTimeout(callback func(), cb_lock sync.Locker, _timeout time.Duration, nticks int, logger sippy_log.ErrorLogger) *Timeout {
	return StartTimeoutWithSpread(callback, cb_lock, _timeout, nticks, logger, 0)
}

func NewInactiveTimeout(callback func(), cb_lock sync.Locker, _timeout time.Duration, nticks int, logger sippy_log.ErrorLogger) *Timeout {
	s := &Timeout{
		callback:      callback,
		timeout:       _timeout,
		nticks:        nticks,
		logger:        logger,
		shutdown_chan: make(chan struct{}),
		spread:        0,
		started:       false,
		cb_lock:       cb_lock,
	}
	return s
}

func (s *Timeout) Start() {
	s.lock.Lock()
	if !s.started && s.callback != nil {
		s.started = true
		go s.run()
	}
	s.lock.Unlock()
}

func (s *Timeout) SpreadRuns(spread float64) {
	s.spread = spread
}

func (s *Timeout) Cancel() {
	close(s.shutdown_chan)
}

func (s *Timeout) run() {
	s._run()
	s.callback = nil
	s.cb_lock = nil
}

func (s *Timeout) _run() {
	var timer *time.Timer
LOOP:
	for s.nticks != 0 {
		if s.nticks > 0 {
			s.nticks--
		}
		t := s.timeout
		if s.spread > 0 {
			t = time.Duration(float64(t) * (1 + s.spread*(1-2*rand.Float64())))
		}
		if timer == nil {
			timer = time.NewTimer(t)
		} else {
			timer.Reset(t)
		}
		select {
		case <-s.shutdown_chan:
			timer.Stop()
			break LOOP
		case <-timer.C:
			sippy_utils.SafeCall(s.callback, s.cb_lock, s.logger)
		}
	}
}
