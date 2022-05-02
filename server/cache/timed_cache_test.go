package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTimedCache(t *testing.T) {

	t.Run("NewTimedCache should return a new instance", func(t *testing.T) {
		cache := NewTimedCache[string, string](time.Second, 1)
		assert.NotNil(t, cache)
	})

	t.Run("TimedCache should cache based on LRU size", func(t *testing.T) {
		cache := NewTimedCache[string, string](time.Second*10, 2)
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
		cache := NewTimedCache[string, string](time.Millisecond*5, 2)
		cache.Add("one", "one")

		_, ok := cache.Get("one")
		assert.True(t, ok)

		time.Sleep(time.Millisecond * 10)

		// "one" should not be available since timeout is 5 ms
		_, ok = cache.Get("one")
		assert.False(t, ok)

	})

}
