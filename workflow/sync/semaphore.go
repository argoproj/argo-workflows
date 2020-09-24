package sync

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	sema "golang.org/x/sync/semaphore"
)

type Semaphore struct {
	name              string
	limit             int
	pending           *priorityQueue
	semaphore         *sema.Weighted
	lockHolder        map[string]bool
	inPending         map[string]bool
	lock              *sync.Mutex
	releaseNotifyFunc ReleaseNotifyCallbackFunc
	log               *log.Entry
}

func NewSemaphore(name string, limit int, callbackFunc func(string), lockType LockType) *Semaphore {
	return &Semaphore{
		name:              name,
		limit:             limit,
		pending:           &priorityQueue{itemByKey: make(map[string]*item)},
		semaphore:         sema.NewWeighted(int64(limit)),
		lockHolder:        make(map[string]bool),
		inPending:         make(map[string]bool),
		lock:              &sync.Mutex{},
		releaseNotifyFunc: callbackFunc,
		log: log.WithFields(log.Fields{
			string(lockType): name,
		}),
	}
}

func (s *Semaphore) getName() string {
	return s.name
}

func (s *Semaphore) getLimit() int {
	return s.limit
}

func (s *Semaphore) getCurrentHolders() []string {
	var keys []string
	for k := range s.lockHolder {
		keys = append(keys, k)
	}
	return keys
}

func (s *Semaphore) resize(n int) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	cur := len(s.lockHolder)
	// downward case, acquired n locks
	if cur > n {
		cur = n
	}

	sema := sema.NewWeighted(int64(n))
	status := sema.TryAcquire(int64(cur))
	if status {
		s.log.Infof("%s semaphore resized from %d to %d", s.name, cur, n)
		s.semaphore = sema
		s.limit = n
	}
	return status
}

func (s *Semaphore) release(key string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.lockHolder[key]; ok {
		delete(s.lockHolder, key)
		// When semaphore resized downward
		// Remove the excess holders from map once the done.
		if len(s.lockHolder) >= s.limit {
			return true
		}

		s.semaphore.Release(1)

		s.log.Infof("Lock has been released by %s. Available locks: %d", key, s.limit-len(s.lockHolder))
		if s.pending.Len() > 0 {
			item := s.pending.peek()
			keyStr := fmt.Sprintf("%v", item.key)
			items := strings.Split(keyStr, "/")
			workflowKey := keyStr
			if len(items) == 3 {
				workflowKey = fmt.Sprintf("%s/%s", items[0], items[1])
			}
			s.log.Debugf("Enqueue the workflow %s", workflowKey)
			s.releaseNotifyFunc(workflowKey)
		}
	}
	return true
}

// addToQueue adds the holderkey into priority queue that maintains the priority order to acquire the lock.
func (s *Semaphore) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("Lock is already acquired by %s", holderKey)
		return
	}

	s.pending.add(holderKey, priority, creationTime)
	s.log.Debugf("Added into Queue %s", holderKey)
}

func (s *Semaphore) acquire(holderKey string) bool {
	if s.semaphore.TryAcquire(1) {
		s.lockHolder[holderKey] = true
		return true
	}
	return false
}

func (s *Semaphore) tryAcquire(holderKey string) (bool, string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("%s is already holding a lock", holderKey)
		return true, ""
	}
	var nextKey string

	waitingMsg := fmt.Sprintf("Waiting for %s lock. Lock status: %d/%d ", s.name, s.limit-len(s.lockHolder), s.limit)

	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	if s.pending.Len() > 0 {
		item := s.pending.peek()
		nextKey = fmt.Sprintf("%v", item.key)
		if holderKey != nextKey {
			return false, waitingMsg
		}
	}

	if s.acquire(holderKey) {
		s.pending.pop()
		s.log.Infof("%s acquired by %s ", s.name, nextKey)
		return true, ""
	}
	s.log.Debugf("Current semaphore Holders. %v", s.lockHolder)
	return false, waitingMsg
}
