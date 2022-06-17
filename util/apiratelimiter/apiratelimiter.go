package apiratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ApiRateLimiter interface {
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
	mu       *sync.Mutex
}

func NewApiRateLimiter(limit int, burst int) *apiRateLimiter {
	// Create a map to hold the rate limiters for each visitor and a mutex.
	var visitors = make(map[string]*visitor)
	var mu sync.Mutex
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
	r.mu.Lock()
	defer r.mu.Unlock()

	v, exists := r.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(r.limit), r.burst)
		// Include the current time when creating a new visitor.
		r.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 5 minutes and delete the entries.
func (r *apiRateLimiter) CleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		r.mu.Lock()
		for ip, v := range r.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(r.visitors, ip)
			}
		}
		r.mu.Unlock()
	}
}