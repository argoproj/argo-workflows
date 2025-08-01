package sync

import (
	"context"
	"errors"
	"reflect"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/deprecation"
)

type syncItem struct {
	semaphore *v1alpha1.SemaphoreRef
	mutex     *v1alpha1.Mutex
}

func allSyncItems(ctx context.Context, sync *v1alpha1.Synchronization) ([]*syncItem, error) {
	var syncItems []*syncItem
	if sync.Semaphore != nil {
		syncItems = append(syncItems, &syncItem{semaphore: sync.Semaphore})
		deprecation.Record(ctx, deprecation.Semaphore)
	}
	if sync.Mutex != nil {
		syncItems = append(syncItems, &syncItem{mutex: sync.Mutex})
		deprecation.Record(ctx, deprecation.Mutex)
	}
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
				return errors.New("Duplicate synchronization item found")
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
