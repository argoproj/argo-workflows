package sync

import (
	"testing"
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/stretchr/testify/assert"
)

func Test_NamespaceBucket(t *testing.T) {
	assert.Equal(t, "a", NamespaceBucket("a/b"))
}

func TestNoParallelismSamePriority(t *testing.T) {
	throttler := NewThrottler(0, SingleBucket, nil)

	throttler.Add("c", 0, time.Now().Add(2*time.Hour))
	throttler.Add("b", 0, time.Now().Add(1*time.Hour))
	throttler.Add("a", 0, time.Now())

	assert.True(t, throttler.Admit("a"))
	assert.True(t, throttler.Admit("b"))
	assert.True(t, throttler.Admit("c"))
}

func TestNoParallelismMultipleBuckets(t *testing.T) {
	throttler := NewThrottler(1, func(key Key) BucketKey {
		namespace, _, _ := cache.SplitMetaNamespaceKey(key)
		return namespace
	}, func(key string) {})

	throttler.Add("a/0", 0, time.Now())
	throttler.Add("a/1", 0, time.Now())
	throttler.Add("b/0", 0, time.Now())
	throttler.Add("b/1", 0, time.Now())

	assert.True(t, throttler.Admit("a/0"))
	assert.False(t, throttler.Admit("a/1"))
	assert.True(t, throttler.Admit("b/0"))
	throttler.Remove("a/0")
	assert.True(t, throttler.Admit("a/1"))
}

func TestWithParallelismLimitAndPriority(t *testing.T) {
	queuedKey := ""
	throttler := NewThrottler(2, SingleBucket, func(key string) { queuedKey = key })

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
