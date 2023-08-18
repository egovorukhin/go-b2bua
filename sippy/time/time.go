package sippy_time

import (
	"math"
	"time"
)

func FloatToDuration(v float64) time.Duration {
	return time.Duration(v * float64(time.Second))
}

func RoundTime(t time.Time) time.Time {
	s := t.Unix()
	if t.Nanosecond() >= 5e8 {
		s += 1
	}
	return time.Unix(s, 0)
}

func FloatToTime(t float64) time.Time {
	i, f := math.Modf(t)
	return time.Unix(int64(i), int64(f*1e9))
}

func TimeToFloat(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e9
}
