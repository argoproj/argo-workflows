package sync

import (
	"sync"
	"time"
)

type PriorityMutex struct {
	name  string
	mutex *PrioritySemaphore
	lock  *sync.Mutex
}

func (m *PriorityMutex) getCurrentPending() []string {
	return m.mutex.getCurrentPending()
}

var _ Semaphore = &PriorityMutex{}

// NewMutex creates new mutex lock object
// name of the mutex
// callbackFunc is a release notification function.
func NewMutex(name string, nextWorkflow NextWorkflow) *PriorityMutex {
	return &PriorityMutex{
		name:  name,
		lock:  &sync.Mutex{},
		mutex: NewSemaphore(name, 1, nextWorkflow, "mutex"),
	}
}

func (m *PriorityMutex) getName() string {
	return m.name
}

func (m *PriorityMutex) getLimit() int {
	return m.mutex.limit
}

func (m *PriorityMutex) getCurrentHolders() []string {
	return m.mutex.getCurrentHolders()
}

func (m *PriorityMutex) resize(n int) bool {
	return false
}

func (m *PriorityMutex) release(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.mutex.release(key)
}

func (m *PriorityMutex) acquire(holderKey string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.mutex.acquire(holderKey)
}

func (m *PriorityMutex) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.mutex.addToQueue(holderKey, priority, creationTime)
}

func (m *PriorityMutex) removeFromQueue(holderKey string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.mutex.removeFromQueue(holderKey)
}

func (m *PriorityMutex) tryAcquire(holderKey string) (bool, string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.mutex.tryAcquire(holderKey)
}
