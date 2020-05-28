package controller

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

type latch struct {
	sync.RWMutex
	store map[types.UID]string
}

func (s *latch) Set(uid types.UID, resourceVersion string) {
	s.Lock()
	defer s.Unlock()
	s.store[uid] = resourceVersion
}

func (s *latch) Delete(uid types.UID) {
	s.Lock()
	defer s.Unlock()
	delete(s.store, uid)
}

// Pass test whether resource version can be accepted.
func (s *latch) Pass(uid types.UID, resourceVersion string) bool {
	s.RLock()
	defer s.RUnlock()
	expected, ok := s.store[uid]
	return !ok || resourceVersion == expected
}

func NewLatch() *latch {
	return &latch{store: make(map[types.UID]string)}
}
