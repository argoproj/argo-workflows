package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTimedCache(t *testing.T) {
	t.Run("NewLRUTtlCache should return a new instance", func(t *testing.T) {
		cache := NewLRUTtlCache(time.Second, 1)
		assert.NotNil(t, cache)
	})

	t.Run("TimedCache should cache based on LRU size", func(t *testing.T) {
		cache := NewLRUTtlCache(time.Second*10, 2)
		cache.Add("one", "one")
		cache.Add("two", "two")

		// Both "one" and "two" should be available since maxSize is 2
		_, ok := cache.Get("one")
		assert.True(t, ok)

		_, ok = cache.Get("two")
		assert.True(t, ok)

		// "three" should be available since its newly added
		cache.Add("three", "three")
		_, ok = cache.Get("three")
		assert.True(t, ok)

		// "one" should not be available since maxSize is 2
		_, ok = cache.Get("one")
		assert.False(t, ok)
	})

	t.Run("TimedCache should cache based on timeout", func(t *testing.T) {
		tempCurrentTime := currentTime

		cache := NewLRUTtlCache(time.Minute*1, 2)

		currentTime = getTimeFunc(0, 0)
		cache.Add("one", "one")

		currentTime = getTimeFunc(0, 30)
		_, ok := cache.Get("one")
		assert.True(t, ok)

		currentTime = getTimeFunc(1, 30)
		// "one" should not be available since timeout is 1 min
		_, ok = cache.Get("one")
		assert.False(t, ok)
		currentTime = tempCurrentTime
	})
}

func getTimeFunc(minutes int, sec int) func() time.Time {
	return func() time.Time {
		return time.Date(0, 0, 0, 0, minutes, sec, 0, time.UTC)
	}
}
