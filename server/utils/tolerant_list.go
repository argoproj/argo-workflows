package utils

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// DecodeUnstructured converts an *unstructured.Unstructured into *out via a
// JSON marshal/unmarshal roundtrip — the same decode the typed client performs on
// the wire bytes this path replaces.
//
// Going through json.Unmarshal is what invokes the custom json.Unmarshaler
// implementations several workflow types rely on (ParallelSteps is an anonymous
// list; Item/AnyString/Plugin/Amount/Object accept multiple shapes); a naive
// reflective field copy would silently zero or drop them. (Current apimachinery's
// runtime.DefaultUnstructuredConverter.FromUnstructured does honor custom
// unmarshalers too, so it would also work — the JSON roundtrip is just the most
// direct equivalent of the typed decode it stands in for.)
//
// ponytail: the roundtrip is 3 serialization passes per item (wire→unstructured by
// the dynamic client, then marshal→unmarshal here) and runs per item on every list
// and per event on the long-lived workflow reflector, where objects can be large.
// Swap to FromUnstructured (one reflective pass, no re-marshal) if this shows up in
// a CPU profile; TestTolerantList_PreservesCustomUnmarshalers guards the unmarshaler
// behavior that switch must not regress.
func DecodeUnstructured[T any](un *unstructured.Unstructured, out *T) error {
	data, err := un.MarshalJSON()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	// The typed clientset this path replaces returns items with empty TypeMeta —
	// the scheme's codec strips Kind/APIVersion on decode since the Go type implies
	// the GVK. The JSON roundtrip here leaves them populated, which would change
	// every list/watch response (golden tests, list-then-resubmit flows). Clear them
	// to preserve the original contract. All CRD types embed metav1.TypeMeta, so the
	// returned pointer satisfies schema.ObjectKind.
	if objKind, ok := any(out).(schema.ObjectKind); ok {
		objKind.SetGroupVersionKind(schema.GroupVersionKind{})
	}
	return nil
}

// TolerantList lists `gvr` via the dynamic client and converts each item into a
// fresh value of T. Items that fail per-item decoding (e.g. a field whose JSON
// shape does not match the typed Go struct) are logged and skipped so the list
// call still returns the well-formed items.
//
// This exists because clusters running the minimal CRDs ship without
// admission-time schema validation: a malformed object can be written to etcd
// and then every typed List() over the namespace 500s, because the typed
// client's single json.Unmarshal of *List is all-or-nothing. Per-item decoding
// is tolerant of one bad row.
//
// Pass an empty `namespace` to list cluster-scoped resources.
func TolerantList[T any](
	ctx context.Context,
	dyn dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	opts metav1.ListOptions,
) ([]T, metav1.ListMeta, error) {
	resource := dyn.Resource(gvr)
	var ul *unstructured.UnstructuredList
	var err error
	if namespace == "" {
		ul, err = resource.List(ctx, opts)
	} else {
		ul, err = resource.Namespace(namespace).List(ctx, opts)
	}
	if err != nil {
		return nil, metav1.ListMeta{}, err
	}

	logger := logging.RequireLoggerFromContext(ctx)
	items := make([]T, 0, len(ul.Items))
	for i := range ul.Items {
		raw := &ul.Items[i]
		var item T
		if convErr := DecodeUnstructured(raw, &item); convErr != nil {
			logger.
				WithField("namespace", raw.GetNamespace()).
				WithField("name", raw.GetName()).
				WithField("resource", gvr.Resource).
				WithField("error", convErr.Error()).
				Warn(ctx, "skipping malformed resource in list response")
			continue
		}
		items = append(items, item)
	}
	meta := metav1.ListMeta{
		ResourceVersion:    ul.GetResourceVersion(),
		Continue:           ul.GetContinue(),
		RemainingItemCount: ul.GetRemainingItemCount(),
	}
	return items, meta, nil
}
