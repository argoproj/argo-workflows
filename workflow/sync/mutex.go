package sync

import (
	"sync"
	"time"
)

type Mutex struct {
	name  string
	mutex *Semaphore
	lock  *sync.Mutex
}

// NewMutex creates new mutex lock object
// name of the mutex
// callbackFunc is a release notification function.
func NewMutex(name string, callbackFunc func(string)) *Mutex {
	return &Mutex{
		name:  name,
		lock:  &sync.Mutex{},
		mutex: NewSemaphore(name, 1, callbackFunc, "mutex"),
	}
}

func (m *Mutex) getName() string {
	return m.name
}

func (m *Mutex) getLimit() int {
	return m.mutex.limit
}

func (m *Mutex) getCurrentHolders() []string {
	return m.mutex.getCurrentHolders()
}

func (m *Mutex) resize(n int) bool {
	return false
}

func (m *Mutex) release(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.mutex.release(key)
}

func (m *Mutex) acquire(holderKey string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.mutex.acquire(holderKey)
}

func (m *Mutex) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.mutex.addToQueue(holderKey, priority, creationTime)
}

func (m *Mutex) tryAcquire(holderKey string) (bool, string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.mutex.tryAcquire(holderKey)
}
