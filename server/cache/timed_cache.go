package cache

import (
	"time"

	"k8s.io/utils/lru"
)

type timedCache[Key comparable, Value any] struct {
	timeout time.Duration
	*lru.Cache
}

type timeValueHolder struct {
	createTime time.Time
	value      interface{}
}

func NewTimedCache[key comparable, value any](timeout time.Duration, size int) *timedCache[key, value] {
	return &timedCache[key, value]{
		timeout: timeout,
		Cache:   lru.New(size),
	}
}

func (c *timedCache[Key, Value]) Get(key Key) (Value, bool) {
	if data, ok := c.Cache.Get(key); ok {
		holder := data.(*timeValueHolder)
		deadline := holder.createTime.Add(c.timeout)
		if c.getCurrentTime().Before(deadline) {
			if value, ok := holder.value.(Value); ok {
				return value, true
			}
		}
		c.Cache.Remove(key)
	}
	return *new(Value), false
}

func (c *timedCache[Key, Value]) Add(key Key, value Value) {
	c.Cache.Add(key, &timeValueHolder{
		createTime: c.getCurrentTime(),
		value:      value,
	})
}

func (c *timedCache[Key, Value]) getCurrentTime() time.Time {
	return time.Now().UTC()
}
