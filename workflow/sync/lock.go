package sync

import "sync"

type KeyLock interface {
	Lock(key string)
	Unlock(key string)
}

type keyLock struct {
	lock  *sync.Mutex
	locks map[string]*sync.Mutex
}

func (l *keyLock) Lock(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	lock, ok := l.locks[key]
	if !ok {
		l.locks[key] = &sync.Mutex{}
		lock = l.locks[key]
	}
	lock.Lock()
}

func (l *keyLock) Unlock(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.locks[key].Unlock()
}

func NewKeyLock() KeyLock {
	return &keyLock{
		lock:  &sync.Mutex{},
		locks: make(map[string]*sync.Mutex),
	}
}
