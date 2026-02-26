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

func checkDuplicates(items []*syncItem) error {
	for i, item := range items {
		for j := i + 1; j < len(items); j++ {
			if reflect.DeepEqual(*item, *items[j]) {
				return errors.New("duplicate synchronization item found")
			}
		}
	}
	return nil
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
