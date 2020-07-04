package keys

import (
	"sync"

	"k8s.io/client-go/tools/cache"
)

// this struct is designed to store a compact cache of meta namespace keys (i.e. "namespace/name") created by an
// informer than can be iterated safely and effeciently
type Keys struct {
	// allow us to list with locking for write
	lock sync.RWMutex
	keys map[string]bool
}

func New() *Keys {
	return &Keys{sync.RWMutex{}, make(map[string]bool)}
}

func (k *Keys) Add(obj interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		k.keys[key] = true
	}
}

func (k *Keys) Remove(obj interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		delete(k.keys, key)
	}
}

func (k *Keys) List() []string {
	k.lock.RLock()
	defer k.lock.RUnlock()
	var keys []string
	for key := range k.keys {
		keys = append(keys, key)
	}
	return keys
}

func Split(key string) (string, string) {
	// this can never happen
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)
	return namespace, name
}
