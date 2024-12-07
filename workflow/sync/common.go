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
	resize(n int) bool
}
