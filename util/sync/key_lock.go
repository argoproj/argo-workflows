package sync

import "sync"

type KeyLock interface {
	Lock(key string)
	Unlock(key string)
}

type keyLock struct {
	lock  *sync.RWMutex
	locks map[string]*sync.Mutex
}

func (l *keyLock) Lock(key string) {
	l.lock.RLock()
	lock, ok := l.locks[key]
	l.lock.RUnlock()
	if !ok {
		lock = &sync.Mutex{}
		l.lock.Lock()
		l.locks[key] = lock
		l.lock.Unlock()
	}
	lock.Lock()
}

func (l *keyLock) Unlock(key string) {
	l.lock.RLock()
	lock := l.locks[key]
	l.lock.RUnlock()
	lock.Unlock()
	l.lock.Lock()
	delete(l.locks, key)
	l.lock.Unlock()
}

func NewKeyLock() KeyLock {
	return &keyLock{
		lock:  &sync.RWMutex{},
		locks: make(map[string]*sync.Mutex),
	}
}
