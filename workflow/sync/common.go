package sync

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type semaphore interface {
	acquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) bool
	checkAcquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) (bool, bool, string)
	tryAcquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) (bool, string)
	release(ctx context.Context, key string) bool
	addToQueue(ctx context.Context, holderKey string, priority int32, creationTime time.Time) error
	removeFromQueue(ctx context.Context, holderKey string) error
	getCurrentHolders(ctx context.Context) ([]string, error)
	getCurrentPending(ctx context.Context) ([]string, error)
	getLimit(ctx context.Context) int // Testing only
	probeWaiting(ctx context.Context)
	lock(ctx context.Context) bool
	unlock(ctx context.Context)
}

// expose for overriding in tests
var nowFn = time.Now
