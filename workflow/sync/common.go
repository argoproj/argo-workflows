package sync

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type semaphore interface {
	acquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) (bool, error)
	// reacquire re-establishes a recorded holder at controller startup.
	//
	// For an in-memory lock it force-registers the holder, ignoring the current
	// limit, so the in-memory count reflects persisted reality even when recorded
	// holders exceed a (since lowered) limit - new acquisitions then correctly
	// wait until the count drains below the limit, rather than dropping a holder
	// (a double-acquire) or poisoning the lock over a routine limit change.
	//
	// For a database-backed lock the database is the single source of truth:
	// reacquire mutates nothing and only asserts the recorded hold still exists
	// there. An error means the hold could not be verified - either the held row
	// is gone (e.g. expired while the controller was down) or the database could
	// not be queried - and the caller fails the holding workflow.
	reacquire(ctx context.Context, holderKey string, tx *sqldb.SessionProxy) error
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
