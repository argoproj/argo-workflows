package informer

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

// ResourceVersionUnchanged reports whether an informer update event carries the same
// resource version on both sides, i.e. it is a store replay (relist or resync) rather
// than a real change. Objects without accessible metadata are treated as changed.
func ResourceVersionUnchanged(oldObj, newObj any) bool {
	oldMeta, err := meta.Accessor(oldObj)
	if err != nil {
		return false
	}
	newMeta, err := meta.Accessor(newObj)
	if err != nil {
		return false
	}
	return oldMeta.GetResourceVersion() == newMeta.GetResourceVersion()
}

// ChangeHandlers call f with every added, changed or deleted object, skipping
// the store replays ResourceVersionUnchanged reports.
func ChangeHandlers(f func(any)) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: f,
		UpdateFunc: func(oldObj, newObj any) {
			if ResourceVersionUnchanged(oldObj, newObj) {
				return
			}
			f(newObj)
		},
		DeleteFunc: f,
	}
}

// Untombstone returns the last known state of an object a delete event carries,
// unwrapping a cache.DeletedFinalStateUnknown.
func Untombstone(obj any) any {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		return tombstone.Obj
	}
	return obj
}
