package sync

import (
	"context"
	"time"
)

type semaphore interface {
	acquire(ctx context.Context, holderKey string, tx *transaction) bool
	checkAcquire(ctx context.Context, holderKey string, tx *transaction) (bool, bool, string)
	tryAcquire(ctx context.Context, holderKey string, tx *transaction) (bool, string)
	release(ctx context.Context, key string) bool
	addToQueue(ctx context.Context, holderKey string, priority int32, creationTime time.Time) error
	removeFromQueue(ctx context.Context, holderKey string) error
	getCurrentHolders(ctx context.Context) ([]string, error)
	getCurrentPending(ctx context.Context) ([]string, error)
	getName() string
	getLimit(ctx context.Context) int // Testing only
	probeWaiting(ctx context.Context)
	lock(ctx context.Context) bool
	unlock(ctx context.Context)
}

// expose for overriding in tests
var nowFn = time.Now
