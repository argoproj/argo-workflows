package sync

import "time"

type semaphore interface {
	acquire(holderKey string) bool
	checkAcquire(holderKey string) (bool, bool, string)
	tryAcquire(holderKey string) (bool, string)
	release(key string) bool
	addToQueue(holderKey string, priority int32, creationTime time.Time)
	removeFromQueue(holderKey string)
	getCurrentHolders() []string
	getCurrentPending() []string
	getName() string
	getLimit() int
	getLimitTimestamp() time.Time
	resetLimitTimestamp()
	resize(n int) bool
	probeWaiting()
}

// expose for overriding in tests
var nowFn = time.Now
