package utils

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

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
		if convErr := runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, &item); convErr != nil {
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
