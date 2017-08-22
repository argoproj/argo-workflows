package lock

import (
	"sync"
	"time"

	"applatix.io/common"
)

type LockGroup struct {
	Name       string
	TtlSeconds int64
	global     sync.Mutex
	lockMap    map[string]*LockEntry
}

type LockEntry struct {
	freshness time.Time
	lock      chan struct{}
}

func (l *LockGroup) Init() {
	l.global.Lock()
	defer l.global.Unlock()
	l.lockMap = make(map[string]*LockEntry)
	if l.TtlSeconds == 0 {
		l.TtlSeconds = 60 * 60 * 12
	}
	l.scheduleLockGC()
	common.DebugLog.Printf("[Lock %v] Lock Group %v is initilized with TTL seconds %v.\n", l.Name, l.Name, l.TtlSeconds)
}

func (l *LockGroup) Size() int {
	l.global.Lock()
	defer l.global.Unlock()
	return len(l.lockMap)
}

func (l *LockGroup) getLock(key string) *LockEntry {
	l.global.Lock()
	defer l.global.Unlock()

	lock, ok := l.lockMap[key]
	if !ok {
		lock = &LockEntry{
			freshness: time.Now(),
			lock:      make(chan struct{}, 1),
		}
		l.lockMap[key] = lock
	}

	return lock
}

func (l *LockGroup) Lock(key string) {
	//common.DebugLog.Printf("[Lock %v] Try Lock Key %v.\n", l.Name, key)
	lock := l.getLock(key)
	lock.lock <- struct{}{}
	lock.freshness = time.Now()
	//common.DebugLog.Printf("[Lock %v] Try Lock Key %v succeed.\n", l.Name, key)
}

func (l *LockGroup) Unlock(key string) {
	//common.DebugLog.Printf("[Lock %v] Try unlock Key %v.\n", l.Name, key)
	lock := l.getLock(key)
	<-lock.lock
	lock.freshness = time.Now()
	//common.DebugLog.Printf("[Lock %v] Try unlock Key %v succeed.\n", l.Name, key)
}

func (l *LockGroup) TryLock(key string, timeout time.Duration) bool {
	//common.DebugLog.Printf("[Lock %v] Try lock Key %v with timeout %v.\n", l.Name, key, timeout.String())
	lock := l.getLock(key)
	if timeout > 0 {
		t := time.After(timeout)
		select {
		case lock.lock <- struct{}{}:
			lock.freshness = time.Now()
			//common.DebugLog.Printf("[Lock %v] Try lock Key %v succeed.\n", l.Name, key)
			return true
		case <-t:
			common.DebugLog.Printf("[Lock %v] Try lock Key %v failed.\n", l.Name, key)
			return false
		}
	} else {
		select {
		case lock.lock <- struct{}{}:
			lock.freshness = time.Now()
			//common.DebugLog.Printf("[Lock %v] Try lock Key %v succeed.\n", l.Name, key)
			return true
		default:
			common.DebugLog.Printf("[Lock %v] Try lock Key %v failed.\n", l.Name, key)
			return false
		}
	}
}

func (l *LockGroup) gc() {
	l.global.Lock()
	defer l.global.Unlock()
	gcKeys := []string{}
	for key, lock := range l.lockMap {
		if time.Now().Sub(lock.freshness).Seconds() > float64(l.TtlSeconds) {
			gcKeys = append(gcKeys, key)
		}
	}

	for _, key := range gcKeys {
		delete(l.lockMap, key)
		common.DebugLog.Printf("[Lock %v] Lock Key %v is GCed.\n", l.Name, key)
	}
}

func (l *LockGroup) scheduleLockGC() {
	ticker := time.NewTicker(time.Second * time.Duration(l.TtlSeconds))
	go func() {
		for _ = range ticker.C {
			common.DebugLog.Printf("[Lock %v] Lock Key GC starting.\n", l.Name)
			l.gc()
			common.DebugLog.Printf("[Lock %v] Lock Key GC finished.\n", l.Name)
		}
	}()
}
