package sync

import (
	log "github.com/sirupsen/logrus"
	"time"
)

type Mutex struct {
	name  string
	mutex *Semaphore
	log               *log.Entry
}

func NewMutex(name string, callbackFunc func(string)) *Mutex {
	return &Mutex{
		name:  name,
		mutex: NewSemaphore(name, 1, callbackFunc, LockTypeMutex),
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
	return m.mutex.release(key)
}

func (m *Mutex) acquire(holderKey string) bool {
	return m.mutex.acquire(holderKey)
}

func (m *Mutex) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	m.mutex.addToQueue(holderKey,priority,creationTime)
}

func (m *Mutex) tryAcquire(holderKey string) (bool, string) {
	return m.mutex.tryAcquire(holderKey)
}
