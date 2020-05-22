package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"sync"
)

// ResourceVersionLatch tracks incremental resource version by object UID.
type ResourceVersionLatch struct {
	sync.RWMutex

	store map[types.UID]string
}

// Get returns resource version by UID key.
func (s *ResourceVersionLatch) Get(uid types.UID) string {
	s.RLock()
	defer s.RUnlock()
	return s.store[uid]
}

// Set setup resource version, without monotonically increasing test.
// Should be called when object is added into cache
func (s *ResourceVersionLatch) Set(uid types.UID, rv string) {
	s.Lock()
	defer s.Unlock()
	s.store[uid] = rv
}

// Update resource version with ensured monotonicity.
func (s *ResourceVersionLatch) Update(uid types.UID, rv string) (success bool) {
	s.Lock()
	defer s.Unlock()
	if rvLocal, exist := s.store[uid]; exist && rv >= rvLocal {
		s.store[uid] = rv
		success = true
	}

	return
}

// Delete resource version by UID. Must be called when object is removed from cache.
func (s *ResourceVersionLatch) Delete(uid types.UID) (success bool) {
	s.Lock()
	defer s.Unlock()
	if _, exist := s.store[uid]; exist {
		delete(s.store, uid)
		success = true
	}

	return
}

// Pass test whether resource version can be accepted.
func (s *ResourceVersionLatch) Pass(uid types.UID, resourceVersion string) bool {
	if rv := s.Get(uid); rv > resourceVersion {
		return false
	}

	return true
}

func NewResourceVersionLatch() *ResourceVersionLatch {
	return &ResourceVersionLatch{store: make(map[types.UID]string)}
}
