package cache

import (
	"sync"

	"k8s.io/client-go/tools/cache"
)

// this struct is designed to store a compact cache of meta namespace keys (i.e. "namespace/name") created by an
// informer than can be iterated safely and efficiently
type store struct {
	lock sync.RWMutex
	keys map[string]bool
}

var _ cache.KeyListerGetter = &store{}

func newStore() *store {
	return &store{sync.RWMutex{}, make(map[string]bool)}
}

func (k *store) GetByKey(key string) (interface{}, bool, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	_, exists := k.keys[key]
	return cache.ExplicitKey(key), exists, nil
}

func (k *store) Add(obj interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return // error should never happen
	}
	k.keys[key] = true
}

func (k *store) Delete(obj interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return // error should never happen
	}
	delete(k.keys, key)
}

func (k *store) ListKeys() []string {
	k.lock.RLock()
	defer k.lock.RUnlock()
	var keys []string
	for key := range k.keys {
		keys = append(keys, key)
	}
	return keys
}
