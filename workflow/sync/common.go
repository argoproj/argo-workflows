package sync

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type semaphore interface {
	acquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) (bool, error)
	// reacquire re-establishes a recorded holder at controller startup, ignoring
	// the current limit. Unlike acquire it always represents the hold, so the
	// in-memory count reflects persisted reality even when recorded holders
	// exceed a (since lowered) limit - new acquisitions then correctly wait until
	// the count drains below the limit, rather than dropping a holder (a
	// double-acquire) or poisoning the lock over a routine limit change.
	reacquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy)
	checkAcquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) (bool, bool, string)
	tryAcquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) (bool, string, error)
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
