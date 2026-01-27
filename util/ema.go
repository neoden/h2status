package util

// EMA implements exponential moving average smoothing.
// Alpha is calculated from period: α = 2 / (period + 1)
type EMA struct {
	alpha   float64
	value   float64
	primed  bool
	samples int
	period  int
}

// NewEMA creates a new EMA smoother with the given period.
// Period corresponds to the "smoothing_interval_seconds" config parameter.
func NewEMA(period int) *EMA {
	if period < 1 {
		period = 1
	}
	return &EMA{
		alpha:  2.0 / float64(period+1),
		period: period,
	}
}

// Update adds a new sample and returns the smoothed value.
func (e *EMA) Update(value float64) float64 {
	if !e.primed {
		e.value = value
		e.primed = true
		e.samples = 1
	} else {
		e.value = e.alpha*value + (1-e.alpha)*e.value
		e.samples++
	}
	return e.value
}

// Value returns the current smoothed value.
func (e *EMA) Value() float64 {
	return e.value
}

// Ready returns true after at least one smoothing has occurred.
func (e *EMA) Ready() bool {
	return e.samples >= 2
}
