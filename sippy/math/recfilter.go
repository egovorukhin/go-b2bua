package sippy_math

type RecFilter interface {
	Apply(float64) float64
	GetLastval() float64
}

type rec_filter struct {
	lastval float64
	a       float64
	b       float64
}

func NewRecFilter(fcoef float64, initval float64) (s *rec_filter) {
	return &rec_filter{
		lastval: initval,
		a:       1.0 - fcoef,
		b:       fcoef,
	}
}

func (s *rec_filter) Apply(x float64) float64 {
	s.lastval = s.a*x + s.b*s.lastval
	return s.lastval
}

func (s *rec_filter) GetLastval() float64 {
	return s.lastval
}
