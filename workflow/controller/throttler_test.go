package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/util/workqueue"
)

func TestNoParallelismSamePriority(t *testing.T) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	throttler := NewThrottler(0, queue)

	throttler.Add("c", 0, time.Now().Add(2*time.Hour))
	throttler.Add("b", 0, time.Now().Add(1*time.Hour))
	throttler.Add("a", 0, time.Now())

	next, ok := throttler.Next("b")
	assert.True(t, ok)
	assert.Equal(t, "a", next)

	next, ok = throttler.Next("c")
	assert.True(t, ok)
	assert.Equal(t, "b", next)
}

func TestWithParallelismLimitAndPriority(t *testing.T) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	throttler := NewThrottler(2, queue)

	throttler.Add("a", 1, time.Now())
	throttler.Add("b", 2, time.Now())
	throttler.Add("c", 3, time.Now())
	throttler.Add("d", 4, time.Now())

	next, ok := throttler.Next("a")
	assert.True(t, ok)
	assert.Equal(t, "d", next)

	next, ok = throttler.Next("a")
	assert.True(t, ok)
	assert.Equal(t, "c", next)

	_, ok = throttler.Next("a")
	assert.False(t, ok)

	next, ok = throttler.Next("c")
	assert.True(t, ok)
	assert.Equal(t, "c", next)

	throttler.Remove("c")

	assert.Equal(t, 1, queue.Len())
	queued, _ := queue.Get()
	assert.Equal(t, "b", queued)
}

func TestChangeParallelism(t *testing.T) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	throttler := NewThrottler(1, queue)

	throttler.Add("a", 1, time.Now())
	throttler.Add("b", 2, time.Now())
	throttler.Add("c", 3, time.Now())
	throttler.Add("d", 4, time.Now())

	next, ok := throttler.Next("a")
	assert.True(t, ok)
	assert.Equal(t, "d", next)

	_, ok = throttler.Next("b")
	assert.False(t, ok)

	_, ok = throttler.Next("c")
	assert.False(t, ok)

	throttler.SetParallelism(3)

	assert.Equal(t, 2, queue.Len())
	queued, _ := queue.Get()
	assert.Equal(t, "c", queued)
	queued, _ = queue.Get()
	assert.Equal(t, "b", queued)
}
