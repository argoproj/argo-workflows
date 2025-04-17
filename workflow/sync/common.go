package sync

import (
	"time"

	"github.com/upper/db/v4"
)

type semaphore interface {
	acquire(holderKey string, session db.Session) bool
	checkAcquire(holderKey string, session db.Session) (bool, bool, string)
	tryAcquire(holderKey string, session db.Session) (bool, string)
	release(key string) bool
	addToQueue(holderKey string, priority int32, creationTime time.Time, session db.Session)
	removeFromQueue(holderKey string)
	getCurrentHolders() []string
	getCurrentPending() []string
	getName() string
	getLimit() int
	probeWaiting()
}

// expose for overriding in tests
var nowFn = time.Now
