package sync

import (
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type Semaphore interface {
	acquire(holderKey string) bool
	tryAcquire(holderKey string) (bool, string)
	release(key string) bool
	addToQueue(holderKey string, priority int32, creationTime time.Time, syncLockRef *wfv1.Synchronization)
	removeFromQueue(holderKey string)
	getCurrentHolders() []string
	getCurrentPending() []string
	getName() string
	getLimit() int
	resize(n int) bool
}

// SemaphoreStrategyQueue is aware that it supports a semaphore, and implements a specific
// strategy for maintaining order of pending holders
type SemaphoreStrategyQueue interface {
	peek() *item
	pop() *item
	add(key Key, priority int32, creationTime time.Time, syncLockRef *wfv1.Synchronization)
	remove(key Key)
	onRelease(key Key)
	all() []*item
	Len() int
}
