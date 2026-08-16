// Package informer provides helpers shared by the informers used in the
// workflow-controller and argo-server.
package informer

import (
	"k8s.io/apimachinery/pkg/api/meta"
)

// StripManagedFields is a cache.TransformFunc that removes
// metadata.managedFields before objects enter an informer cache.
// Nothing reads managedFields from the cache, and the API server ignores
// a nil managedFields on non-apply updates, so this is loss-free. It can
// be 20% or more of the object size.
func StripManagedFields(i any) (any, error) {
	obj, err := meta.Accessor(i)
	// err != nil: tombstones (DeletedFinalStateUnknown) etc — pass through.
	// The nil-check avoids mutating objects that already lack managedFields
	// (kubernetes/kubernetes#124337).
	if err == nil && obj.GetManagedFields() != nil {
		obj.SetManagedFields(nil)
	}
	return i, nil
}
