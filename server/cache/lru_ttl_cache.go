package cache

import (
	"time"

	"k8s.io/utils/lru"
)

type lruTtlCache struct {
	timeout time.Duration
	cache   *lru.Cache
}

type timeValueHolder struct {
	createTime time.Time
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
		holder := data.(*timeValueHolder)
		deadline := holder.createTime.Add(c.timeout)
		if getCurrentTime().Before(deadline) {
			return holder.value, true
		}
		c.cache.Remove(key)
	}
	return nil, false
}

func (c *lruTtlCache) Add(key string, value any) {
	c.cache.Add(key, &timeValueHolder{
		createTime: getCurrentTime(),
		value:      value,
	})
}

func getCurrentTime() time.Time {
	return time.Now().UTC()
}
