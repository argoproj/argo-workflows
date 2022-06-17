package apiratelimiter

import (
	"sync"

	"golang.org/x/time/rate"
)

type ApiRateLimiter interface {
	GetVisitor(ip string) *rate.Limiter
}

type apiRateLimiter struct {
	limit    int
	burst    int
	visitors map[string]*rate.Limiter
	mu       *sync.Mutex
}

func NewApiRateLimiter(limit int, burst int) *apiRateLimiter {
	// Create a map to hold the rate limiters for each visitor and a mutex.
	var visitors = make(map[string]*rate.Limiter)
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

	limiter, exists := r.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(r.limit), r.burst)
		r.visitors[ip] = limiter
	}

	return limiter
}