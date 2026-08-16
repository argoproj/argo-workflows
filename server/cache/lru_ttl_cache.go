package cache

import (
	"time"

	"k8s.io/utils/lru"
)

var currentTime = time.Now

type lruTTLCache struct {
	timeout time.Duration
	cache   *lru.Cache
}

type item struct {
	expiryTime time.Time
	value      any
}

func NewLRUTtlCache(timeout time.Duration, size int) Interface {
	return &lruTTLCache{
		timeout: timeout,
		cache:   lru.New(size),
	}
}

func (c *lruTTLCache) Get(key string) (any, bool) {
	if data, ok := c.cache.Get(key); ok {
		item := data.(*item)
		if currentTime().Before(item.expiryTime) {
			return item.value, true
		}
		c.cache.Remove(key)
	}
	return nil, false
}

func (c *lruTTLCache) Add(key string, value any) {
	c.cache.Add(key, &item{
		expiryTime: currentTime().Add(c.timeout),
		value:      value,
	})
}
