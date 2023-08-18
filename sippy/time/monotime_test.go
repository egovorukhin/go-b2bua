package sippy_time

import (
	"runtime"
	"testing"
)

func TestMonoTime(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	for i := 0; i < 100000; i++ {
		m1, _ := NewMonoTime()
		m2, _ := NewMonoTime()
		if i == 0 {
			t.Log(m1.Fptime(), m2.Fptime())
		}
	}
}
