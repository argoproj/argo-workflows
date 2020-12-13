package ticker

import (
	"time"
)

type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(time.Duration)
}

type realTicker struct {
	t *time.Ticker
}

func (rt *realTicker) Stop() {
	rt.t.Stop()
}

func (rt *realTicker) Reset(d time.Duration) {
	rt.t.Reset(d)
}

func (rt *realTicker) C() <-chan time.Time {
	return rt.t.C
}

func NewTicker(d time.Duration) Ticker {
	return &realTicker{
		t: time.NewTicker(d),
	}
}
