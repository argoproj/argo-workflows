package sync

import "sync"

type KeyLock interface {
	Lock(key string)
	Unlock(key string)
}

type keyLock struct {
	guard sync.RWMutex
	locks map[string]*sync.Mutex
}

func NewKeyLock() KeyLock {
	return &keyLock{
		guard: sync.RWMutex{},
		locks: map[string]*sync.Mutex{},
	}
}

func (l *keyLock) getLock(key string) *sync.Mutex {
	l.guard.RLock()
	if lock, ok := l.locks[key]; ok {
		l.guard.RUnlock()
		return lock
	}

	l.guard.RUnlock()
	l.guard.Lock()

	if lock, ok := l.locks[key]; ok {
		l.guard.Unlock()
		return lock
	}

	lock := &sync.Mutex{}
	l.locks[key] = lock
	l.guard.Unlock()
	return lock
}

func (l *keyLock) Lock(key string) {
	l.getLock(key).Lock()
}

func (l *keyLock) Unlock(key string) {
	l.getLock(key).Unlock()
}
