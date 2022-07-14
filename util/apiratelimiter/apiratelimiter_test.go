package apiratelimiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAPIRateLimiter (t *testing.T) {
	ratelimiter := NewAPIRateLimiter (1, 1)
	visitor1 := ratelimiter.GetVisitor("123")
	visitor2 := ratelimiter.GetVisitor("123")

	assert.Equal(t, visitor1, visitor2)
	assert.True(t, visitor1.Allow())
	assert.False(t, visitor1.Allow())

	go ratelimiter.CleanupVisitors(1*time.Millisecond, 0*time.Second)
	time.Sleep(10 * time.Millisecond)
	visitor3 := ratelimiter.GetVisitor("123")
	assert.NotEqual(t, visitor1, visitor3)
}
