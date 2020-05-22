package controller

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	sem "golang.org/x/sync/semaphore"
)

type Semaphore struct {
	name       string
	limit      int
	pending    *priorityQueue
	semaphore  *sem.Weighted
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
		semaphore:  sem.NewWeighted(int64(limit)),
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

func (ws *Semaphore) enqueueWorkflow(key string) {
	ws.controller.wfQueue.AddAfter(key, 0)
}

func (ws *Semaphore) Release(key string) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	if _, ok := ws.lockHolder[key]; ok {
		ws.semaphore.Release(1)
		delete(ws.lockHolder, key)
		ws.log.Infof("Lock has been released by %s", key)
		if ws.pending.Len() > 0 {
			item := ws.pending.Peek().(*item)
			keyStr := fmt.Sprintf("%v", item.key)
			items := strings.Split(keyStr, "/")
			workflowKey := keyStr
			if len(items) == 3 {
				workflowKey = fmt.Sprintf("%s/%s", items[0], items[1])
			}
			ws.log.Debugf("Enqueue the Workflow %s \n", workflowKey)
			ws.enqueueWorkflow(workflowKey)
		}
	}
}

func (ws *Semaphore) AddToQueue(holderKey string, priority int32, creationTime time.Time) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	if _, ok := ws.lockHolder[holderKey]; ok {
		ws.log.Debugf("Already Lock is acquired %s \n", holderKey)
		return
	}

	if _, ok := ws.inPending[holderKey]; ok {
		ws.log.Debugf("Already is queue %s \n", holderKey)
		return
	}
	ws.pending.add(holderKey, priority, creationTime)
	ws.inPending[holderKey] = true
	ws.log.Debugf("Added into Queue %s \n", holderKey)
}

func (ws *Semaphore) TryAcquire(holderKey string) (bool, string) {
	ws.lock.Lock()
	defer ws.lock.Unlock()

	if _, ok := ws.lockHolder[holderKey]; ok {
		ws.log.Debugf("%s is holding a lock\n", holderKey)
		return true, ""
	}
	var nextKey string

	waitingMsg := fmt.Sprintf("Waiting for Lock. Lock status: %d/%d ", ws.limit-len(ws.lockHolder), ws.limit)
	if ws.pending.Len() > 0 {
		item := ws.pending.Peek().(*item)
		nextKey = fmt.Sprintf("%v", item.key)
		if holderKey != nextKey {
			return false, waitingMsg
		}
	}

	if ws.semaphore.TryAcquire(1) {
		ws.pending.Pop()
		delete(ws.inPending, holderKey)
		ws.log.Infof("Lock is acquired by %s \n", nextKey)
		ws.lockHolder[nextKey] = true
		return true, ""
	}
	return false, waitingMsg

}
