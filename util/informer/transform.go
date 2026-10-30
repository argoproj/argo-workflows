// Package informer provides helpers shared by the informers used in the
// workflow-controller and argo-server.
package informer

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
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

// StripLastAppliedConfiguration is a cache.TransformFunc that removes the
// kubectl.kubernetes.io/last-applied-configuration annotation before objects
// enter an informer cache. Client-side `kubectl apply` (and GitOps tooling
// built on it) stores a copy of the whole spec there, which on a real cluster
// measured 3.6 KB of an 8.3 KB WorkflowTemplate.
//
// Unlike managedFields this is ordinary object data, so it is only safe on a
// cache whose objects are never written back whole: dropping it from an object
// that is later sent in a full Update would remove it from the live object and
// break the next client-side-apply three-way merge.
//
// That makes it controller-only. argo-server builds the same tolerant template
// informers, and its GetWorkflowTemplate serves the cached object straight to
// the client, which may send it back to UpdateWorkflowTemplate. So the shared
// constructors keep StripManagedFields, and the controller alone swaps in
// StripTemplateCacheFields on its own informers.
func StripLastAppliedConfiguration(i any) (any, error) {
	obj, err := meta.Accessor(i)
	// err != nil: tombstones (DeletedFinalStateUnknown) etc — pass through.
	// The membership test is a cost saving rather than behaviour: without it
	// every object entering the cache has its annotations copied and written
	// back for nothing. Removing it changes no result, so no test pins it.
	if err == nil {
		annotations := obj.GetAnnotations()
		if _, ok := annotations[corev1.LastAppliedConfigAnnotation]; ok {
			delete(annotations, corev1.LastAppliedConfigAnnotation)
			obj.SetAnnotations(annotations)
		}
	}
	return i, nil
}

// StripTemplateCacheFields strips both managedFields and the client-side-apply
// annotation. It is for the controller's WorkflowTemplate and
// ClusterWorkflowTemplate informers, whose objects the controller only ever
// reads -- not for argo-server's, which serves them to clients.
func StripTemplateCacheFields(i any) (any, error) {
	return chain(StripManagedFields, StripLastAppliedConfiguration)(i)
}

// chain runs transforms in order, passing each one's result to the next and
// stopping at the first error. SetTransform takes a single function, so
// composing is the only way to apply more than one to an informer.
func chain(fns ...cache.TransformFunc) cache.TransformFunc {
	return func(i any) (any, error) {
		var err error
		for _, fn := range fns {
			if i, err = fn(i); err != nil {
				return i, err
			}
		}
		return i, nil
	}
}
