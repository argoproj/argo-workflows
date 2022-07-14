package apiratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type APIRateLimiter interface {
	GetVisitor(ip string) *rate.Limiter
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type apiRateLimiter struct {
	limit    int
	burst    int
	visitors *sync.Map
}

func NewAPIRateLimiter(limit int, burst int) *apiRateLimiter {
	// Create a map to hold the rate limiters for each visitor and a mutex.
	var visitors sync.Map
	return &apiRateLimiter{
		limit:    limit,
		burst:    burst,
		visitors: &visitors,
	}
}

// Retrieve and return the rate limiter for the current visitor if it
// already exists. Otherwise create a new rate limiter and add it to
// the visitors map, using the IP address as the key.
func (r *apiRateLimiter) GetVisitor(ip string) *rate.Limiter {
	value, exists := r.visitors.Load(ip)

	if !exists {
		limiter := rate.NewLimiter(rate.Limit(r.limit), r.burst)
		// Include the current time when creating a new visitor.
		r.visitors.Store(ip, &visitor{limiter, time.Now()})
		return limiter
	}

	v := value.(*visitor)

	// Update the last seen time for the visitor
	now := time.Now()
	shouldUpdate := v.lastSeen.Before(now.Add(time.Duration(-1) * time.Minute))
	if shouldUpdate {
		limiter := rate.NewLimiter(rate.Limit(r.limit), r.burst)
		r.visitors.Store(ip, &visitor{limiter, now})
		return limiter
	}

	return v.limiter
}

// Every duration check the map for visitors that haven't been seen for
// a duration and delete the entries.
func (r *apiRateLimiter) CleanupVisitors(freq time.Duration, duration time.Duration) {
	for {
		time.Sleep(freq)

		r.visitors.Range(func(key, value interface{}) bool {
			v := value.(*visitor)
			if time.Since(v.lastSeen) > duration {
				r.visitors.Delete(key)
			}
			return true
		})
	}
}