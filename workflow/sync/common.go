package sync

import (
	"time"
)

type semaphore interface {
	acquire(holderKey string, tx *transaction) bool
	checkAcquire(holderKey string, tx *transaction) (bool, bool, string)
	tryAcquire(holderKey string, tx *transaction) (bool, string)
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
