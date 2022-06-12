package cache

import (
	"time"

	"k8s.io/utils/lru"
)

var currentTime = time.Now

type lruTtlCache struct {
	timeout time.Duration
	cache   *lru.Cache
}

type item struct {
	expiryTime time.Time
	value      any
}

func NewLRUTtlCache(timeout time.Duration, size int) *lruTtlCache {
	return &lruTtlCache{
		timeout: timeout,
		cache:   lru.New(size),
	}
}

func (c *lruTtlCache) Get(key string) (any, bool) {
	if data, ok := c.cache.Get(key); ok {
		item := data.(*item)
		if currentTime().Before(item.expiryTime) {
			return item.value, true
		}
		c.cache.Remove(key)
	}
	return nil, false
}

func (c *lruTtlCache) Add(key string, value any) {
	c.cache.Add(key, &item{
		expiryTime: currentTime().Add(c.timeout),
		value:      value,
	})
}
