package sync

import (
	"errors"
	"reflect"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type syncItem struct {
	semaphore *v1alpha1.SemaphoreRef
	mutex     *v1alpha1.Mutex
}

func allSyncItems(sync *v1alpha1.Synchronization) ([]*syncItem, error) {
	var syncItems []*syncItem
	for _, semaphore := range sync.Semaphores {
		syncItems = append(syncItems, &syncItem{semaphore: semaphore})
	}
	for _, mtx := range sync.Mutexes {
		syncItems = append(syncItems, &syncItem{mutex: mtx})
	}
	return syncItems, checkDuplicates(syncItems)
}

// checkDuplicates returns an error if any two items are deeply equal. Items are
// grouped by their names, so each one is only compared with the few items that
// could equal it rather than with every other item.
func checkDuplicates(items []*syncItem) error {
	seen := make(map[syncItemKey][]*syncItem, len(items))
	for _, item := range items {
		key := item.key()
		for _, other := range seen[key] {
			if reflect.DeepEqual(*item, *other) {
				return errors.New("duplicate synchronization item found")
			}
		}
		seen[key] = append(seen[key], item)
	}
	return nil
}

// syncItemKey holds the names of a syncItem. Deeply equal items have equal keys.
type syncItemKey struct {
	mutex                       bool
	namespace, name, key, dbKey string
}

func (i *syncItem) key() syncItemKey {
	var k syncItemKey
	if i.semaphore != nil {
		k.namespace = i.semaphore.Namespace
		if ref := i.semaphore.ConfigMapKeyRef; ref != nil {
			k.name, k.key = ref.Name, ref.Key
		}
		if db := i.semaphore.Database; db != nil {
			k.dbKey = db.Key
		}
	}
	if i.mutex != nil {
		k.mutex = true
		k.namespace, k.name = i.mutex.Namespace, i.mutex.Name
	}
	return k
}

func (i *syncItem) getType() v1alpha1.SynchronizationType {
	switch {
	case i.semaphore != nil:
		return v1alpha1.SynchronizationTypeSemaphore
	case i.mutex != nil:
		return v1alpha1.SynchronizationTypeMutex
	default:
		return v1alpha1.SynchronizationTypeUnknown
	}
}
