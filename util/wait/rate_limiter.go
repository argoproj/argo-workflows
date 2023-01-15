package wait

import (
	"sync"
	"time"
)

type RateLimiter struct {
	last     time.Time
	interval time.Duration
	mutex    *sync.Mutex
}

func NewRateLimiter(interval time.Duration) RateLimiter {
	return RateLimiter{last: time.Now().Add(-interval), interval: interval, mutex: &sync.Mutex{}}
}

func (d *RateLimiter) Wait() {
	d.mutex.Lock()
	canRunAt := d.last.Add(d.interval)
	now := time.Now()
	diff := canRunAt.Sub(now)
	if diff > 0 {
		time.Sleep(diff)
	}
	d.last = now
	d.mutex.Unlock()
}
