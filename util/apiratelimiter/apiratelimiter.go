package apiratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type APIRateLimiter  interface {
	GetVisitor(ip string) *rate.Limiter
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type apiRateLimiter struct {
	limit    int
	burst    int
	visitors map[string]*visitor
	mu       *sync.RWMutex
}

func NewAPIRateLimiter (limit int, burst int) *apiRateLimiter {
	// Create a map to hold the rate limiters for each visitor and a mutex.
	var visitors = make(map[string]*visitor)
	var mu sync.RWMutex
	return &apiRateLimiter{
		limit:    limit,
		burst:    burst,
		visitors: visitors,
		mu:       &mu,
	}
}

// Retrieve and return the rate limiter for the current visitor if it
// already exists. Otherwise create a new rate limiter and add it to
// the visitors map, using the IP address as the key.
func (r *apiRateLimiter) GetVisitor(ip string) *rate.Limiter {
	r.mu.RLock()
	v, exists := r.visitors[ip]

	if !exists {
		r.mu.RUnlock()
		limiter := rate.NewLimiter(rate.Limit(r.limit), r.burst)
		// Include the current time when creating a new visitor.
		r.mu.Lock()
		r.visitors[ip] = &visitor{limiter, time.Now()}
		r.mu.Unlock()
		return limiter
	}

	// Update the last seen time for the visitor
	now := time.Now()
	shouldUpdate := v.lastSeen.Before(now.Add(time.Duration(-1) * time.Minute))
	r.mu.RUnlock()
	if shouldUpdate {
		r.mu.Lock()
		v.lastSeen = now
		r.mu.Unlock()
	}

	return v.limiter
}

// Every duration check the map for visitors that haven't been seen for
// a duration and delete the entries.
func (r *apiRateLimiter) CleanupVisitors(freq time.Duration, duration time.Duration) {
	for {
		time.Sleep(freq)

		r.mu.Lock()
		for ip, v := range r.visitors {
			if time.Since(v.lastSeen) > duration {
				delete(r.visitors, ip)
			}
		}
		r.mu.Unlock()
	}
}