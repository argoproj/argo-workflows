package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoParallelismSamePriority(t *testing.T) {
	throttler := NewThrottler(0, nil)

	throttler.Add("c", 0, time.Now().Add(2*time.Hour))
	throttler.Add("b", 0, time.Now().Add(1*time.Hour))
	throttler.Add("a", 0, time.Now())

	assert.True(t, throttler.Admit("a"))
	assert.True(t, throttler.Admit("b"))
	assert.True(t, throttler.Admit("c"))
}

func TestWithParallelismLimitAndPriority(t *testing.T) {
	queuedKey := ""
	throttler := NewThrottler(2, func(key string) { queuedKey = key })

	throttler.Add("a", 1, time.Now())
	throttler.Add("b", 2, time.Now())
	throttler.Add("c", 3, time.Now())
	throttler.Add("d", 4, time.Now())

	assert.True(t, throttler.Admit("a"), "is started, even though low priority")
	assert.True(t, throttler.Admit("b"), "is started, even though low priority")
	assert.False(t, throttler.Admit("c"), "cannot start")
	assert.False(t, throttler.Admit("d"), "cannot start")
	assert.Equal(t, "b", queuedKey)
	queuedKey = ""

	throttler.Remove("a")
	assert.True(t, throttler.Admit("b"), "stays running")
	assert.True(t, throttler.Admit("d"), "top priority")
	assert.False(t, throttler.Admit("c"))
	assert.Equal(t, "d", queuedKey)
	queuedKey = ""

	throttler.Remove("b")
	assert.True(t, throttler.Admit("d"), "top priority")
	assert.True(t, throttler.Admit("c"), "now running too")
	assert.Equal(t, "c", queuedKey)
}
