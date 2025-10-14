package sync

import (
	"context"
	"time"
)

type limitProvider interface {
	get(ctx context.Context, key string) (int, bool, error)
}

var _ limitProvider = &cachedLimit{}

type cachedLimit struct {
	limit          int
	limitTimestamp time.Time
	TTL            time.Duration
	getter         GetSyncLimit
}

func newCachedLimit(getter GetSyncLimit, TTL time.Duration) *cachedLimit {
	return &cachedLimit{
		limit:          0,
		limitTimestamp: time.Time{}, // very long ago, so first use will update
		TTL:            TTL,
		getter:         getter,
	}
}

func (c *cachedLimit) get(ctx context.Context, key string) (int, bool, error) {
	changed := false
	if nowFn().Sub(c.limitTimestamp) >= c.TTL {
		limit, err := c.getter(ctx, key)
		if err != nil {
			return c.limit, false, err
		}
		if limit != c.limit {
			c.limit = limit
			changed = true
		}
		c.limitTimestamp = nowFn()
	}
	return c.limit, changed, nil
}
