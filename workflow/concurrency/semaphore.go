package concurrency

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

func NewSemaphore(name string, limit int, callbackFunc func(string)) *Semaphore {
	holder := make(map[string]bool)
	return &Semaphore{
		name:              name,
		limit:             limit,
		pending:           &priorityQueue{itemByKey: make(map[interface{}]*item)},
		semaphore:         sema.NewWeighted(int64(limit)),
		lockHolder:        holder,
		inPending:         make(map[string]bool),
		lock:              &sync.Mutex{},
		releaseNotifyFunc: callbackFunc,
		log: log.WithFields(log.Fields{
			"semaphore": name,
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
	if cur >= n {
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

func (s *Semaphore) release(key string) LockStatus {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.lockHolder[key]; ok {
		// When Semaphore resized downward
		// Remove the excess holders from map once the done.
		if len(s.lockHolder) > s.limit {
			delete(s.lockHolder, key)
			return Released
		}
		log.Println(key)
		s.semaphore.Release(1)
		delete(s.lockHolder, key)

		s.log.Infof("Lock has been released by %s. Available locks: %d", key, s.limit-len(s.lockHolder))
		if s.pending.Len() > 0 {
			item := s.pending.peek()
			keyStr := fmt.Sprintf("%v", item.key)
			items := strings.Split(keyStr, "/")
			workflowKey := keyStr
			if len(items) == 3 {
				workflowKey = fmt.Sprintf("%s/%s", items[0], items[1])
			}
			s.log.Debugf("Enqueue the Workflow %s \n", workflowKey)
			s.releaseNotifyFunc(workflowKey)
		}
	}
	return Released
}

func (s *Semaphore) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("Already Lock is acquired %s \n", holderKey)
		return
	}

	if _, ok := s.inPending[holderKey]; ok {
		s.log.Debugf("Already is queue %s \n", holderKey)
		return
	}
	s.pending.add(holderKey, priority, creationTime)
	s.inPending[holderKey] = true
	s.log.Debugf("Added into Queue %s \n", holderKey)
}

func (s *Semaphore) acquire(holderKey string) LockStatus {
	if s.semaphore.TryAcquire(1) {
		s.lockHolder[holderKey] = true
		return Acquired
	}
	return Waiting
}

func (s *Semaphore) tryAcquire(holderKey string) (LockStatus, string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("%s is already holding a lock\n", holderKey)
		return AlreadyAcquired, ""
	}
	var nextKey string

	waitingMsg := fmt.Sprintf("Waiting for Lock. Lock status: %d/%d ", s.limit-len(s.lockHolder), s.limit)
	if s.pending.Len() > 0 {
		item := s.pending.peek()
		nextKey = fmt.Sprintf("%v", item.key)
		if holderKey != nextKey {
			return Waiting, waitingMsg
		}
	}

	if status := s.acquire(holderKey); status == Acquired {
		s.pending.pop()
		delete(s.inPending, holderKey)
		s.log.Infof("%s acquired by %s \n", s.name, nextKey)
		return status, ""
	}
	s.log.Debugf("Current Semaphore Holders. %v", s.lockHolder)
	return Waiting, waitingMsg

}
