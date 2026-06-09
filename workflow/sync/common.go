package sync

import (
	"time"
)

type semaphore interface {
	acquire(holderKey string, tx *transaction) (bool, error)
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
	reacquire(holderKey string, tx *transaction) error
	checkAcquire(holderKey string, tx *transaction) (bool, bool, string)
	tryAcquire(holderKey string, tx *transaction) (bool, string, error)
	release(key string) bool
	addToQueue(holderKey string, priority int32, creationTime time.Time) error
	removeFromQueue(holderKey string) error
	getCurrentHolders() ([]string, error)
	getCurrentPending() ([]string, error)
	getName() string
	getLimit() int // Testing only
	probeWaiting()
	lock() bool
	unlock()
}

// expose for overriding in tests
var nowFn = time.Now
