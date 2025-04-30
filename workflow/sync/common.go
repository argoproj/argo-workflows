package sync

import (
	"time"
)

type semaphore interface {
	acquire(holderKey string, tx *transaction) bool
	checkAcquire(holderKey string, tx *transaction) (bool, bool, string)
	tryAcquire(holderKey string, tx *transaction) (bool, string)
	release(key string) bool
	addToQueue(holderKey string, priority int32, creationTime time.Time, tx *transaction)
	removeFromQueue(holderKey string)
	getCurrentHolders() ([]string, error)
	getCurrentPending() ([]string, error)
	getName() string
	getLimit() int
	probeWaiting()
}

// expose for overriding in tests
var nowFn = time.Now
