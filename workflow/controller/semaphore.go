package controller

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	sema "golang.org/x/sync/semaphore"
)

type Semaphore struct {
	name       string
	limit      int
	pending    *priorityQueue
	semaphore  *sema.Weighted
	lockHolder map[string]bool
	inPending  map[string]bool
	lock       *sync.Mutex
	controller *WorkflowController
	log        *log.Entry
}

func NewSemaphore(name string, limit int, wfc *WorkflowController) *Semaphore {
	holder := make(map[string]bool)
	return &Semaphore{
		name:       name,
		limit:      limit,
		pending:    &priorityQueue{itemByKey: make(map[interface{}]*item)},
		semaphore:  sema.NewWeighted(int64(limit)),
		lockHolder: holder,
		inPending:  make(map[string]bool),
		lock:       &sync.Mutex{},
		controller: wfc,
		log: log.WithFields(log.Fields{
			"semaphore": name,
			"limit":     limit,
		}),
	}
}

func (s *Semaphore) enqueueWorkflow(key string) {
	s.controller.wfQueue.AddAfter(key, 0)
}

func (s *Semaphore) Release(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.lockHolder[key]; ok {
		s.semaphore.Release(1)
		delete(s.lockHolder, key)
		s.log.Infof("Lock has been released by %s. Available locks: %d", key, s.limit-len(s.lockHolder))
		if s.pending.Len() > 0 {
			item := s.pending.Peek().(*item)
			keyStr := fmt.Sprintf("%v", item.key)
			items := strings.Split(keyStr, "/")
			workflowKey := keyStr
			if len(items) == 3 {
				workflowKey = fmt.Sprintf("%s/%s", items[0], items[1])
			}
			s.log.Debugf("Enqueue the Workflow %s \n", workflowKey)
			s.enqueueWorkflow(workflowKey)
		}
	}
}

func (s *Semaphore) AddToQueue(holderKey string, priority int32, creationTime time.Time) {
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
func (s *Semaphore) Acquire(holderKey string) bool {
	if s.semaphore.TryAcquire(1) {
		s.lockHolder[holderKey] = true
		return true
	}
	return false
}

func (s *Semaphore) TryAcquire(holderKey string) (bool, string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("%s is already holding a lock\n", holderKey)
		return true, ""
	}
	var nextKey string

	waitingMsg := fmt.Sprintf("Waiting for Lock. Lock status: %d/%d ", s.limit-len(s.lockHolder), s.limit)
	if s.pending.Len() > 0 {
		item := s.pending.Peek().(*item)
		nextKey = fmt.Sprintf("%v", item.key)
		if holderKey != nextKey {
			return false, waitingMsg
		}
	}

	if s.Acquire(holderKey) {
		s.pending.Pop()
		delete(s.inPending, holderKey)
		s.log.Infof("%s acquired by %s \n", s.name, nextKey)
		return true, ""
	}
	return false, waitingMsg

}
