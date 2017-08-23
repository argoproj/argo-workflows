package cache

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcachedStore struct {
	*memcache.Client
	defaultExpiration time.Duration
}

func NewMemcachedStore(hostList []string, defaultExpiration time.Duration) *MemcachedStore {
	return &MemcachedStore{memcache.New(hostList...), defaultExpiration}
}

func (c *MemcachedStore) Set(key string, value interface{}, expires time.Duration) error {
	return c.invoke((*memcache.Client).Set, key, value, expires)
}

func (c *MemcachedStore) Add(key string, value interface{}, expires time.Duration) error {
	return c.invoke((*memcache.Client).Add, key, value, expires)
}

func (c *MemcachedStore) Replace(key string, value interface{}, expires time.Duration) error {
	return c.invoke((*memcache.Client).Replace, key, value, expires)
}

func (c *MemcachedStore) Get(key string, value interface{}) error {
	item, err := c.Client.Get(key)
	if err != nil {
		return convertMemcacheError(err)
	}
	return deserialize(item.Value, value)
}

func (c *MemcachedStore) Delete(key string) error {
	return convertMemcacheError(c.Client.Delete(key))
}

func (c *MemcachedStore) Increment(key string, delta uint64) (uint64, error) {
	newValue, err := c.Client.Increment(key, delta)
	return newValue, convertMemcacheError(err)
}

func (c *MemcachedStore) Decrement(key string, delta uint64) (uint64, error) {
	newValue, err := c.Client.Decrement(key, delta)
	return newValue, convertMemcacheError(err)
}

func (c *MemcachedStore) Flush() error {
	return ErrNotSupport
}

func (c *MemcachedStore) invoke(storeFn func(*memcache.Client, *memcache.Item) error,
	key string, value interface{}, expire time.Duration) error {

	switch expire {
	case DEFAULT:
		expire = c.defaultExpiration
	case FOREVER:
		expire = time.Duration(0)
	}

	b, err := serialize(value)
	if err != nil {
		return err
	}
	return convertMemcacheError(storeFn(c.Client, &memcache.Item{
		Key:        key,
		Value:      b,
		Expiration: int32(expire / time.Second),
	}))
}

func convertMemcacheError(err error) error {
	switch err {
	case nil:
		return nil
	case memcache.ErrCacheMiss:
		return ErrCacheMiss
	case memcache.ErrNotStored:
		return ErrNotStored
	}

	return err
}
