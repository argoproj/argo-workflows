package informer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/scheme"
)

func annotatedTemplate(kind, namespace string) *unstructured.Unstructured {
	metadata := map[string]any{
		"name":          "my-template",
		"managedFields": []any{map[string]any{"manager": "kubectl-client-side-apply"}},
		"annotations": map[string]any{
			apiv1.LastAppliedConfigAnnotation: `{"spec":{}}`,
			"keep-me":                         "yes",
		},
	}
	if namespace != "" {
		metadata["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       kind,
		"metadata":   metadata,
	}}
}

// cachedTemplate starts the informer and returns the single object its cache holds.
func cachedTemplate(t *testing.T, informer cache.SharedIndexInformer) *unstructured.Unstructured {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	go informer.Run(ctx.Done())
	require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced), "informer cache did not sync")
	objects := informer.GetStore().List()
	require.Len(t, objects, 1)
	un, ok := objects[0].(*unstructured.Unstructured)
	require.True(t, ok, "expected an *unstructured.Unstructured in the cache, got %T", objects[0])
	return un
}

// The tolerant constructors are shared: argo-server builds them too, and its
// GetWorkflowTemplate serves the cached object straight to a client that may
// send it back in an update. So the shared transform must strip managedFields
// and leave the client-side-apply annotation in place. Only the controller,
// which never writes these objects back, swaps in StripTemplateCacheFields.
//
// Unlike a direct test of StripManagedFields, this fails if either constructor
// is switched to the controller's transform.
func TestTolerantInformersKeepLastAppliedConfiguration(t *testing.T) {
	t.Run("WorkflowTemplate", func(t *testing.T) {
		seeded := annotatedTemplate("WorkflowTemplate", "my-ns")
		client := dynamicfake.NewSimpleDynamicClient(scheme.Scheme, seeded)

		cached := cachedTemplate(t, NewTolerantWorkflowTemplateInformer(client, 0, "my-ns").Informer())

		assert.Contains(t, cached.GetAnnotations(), apiv1.LastAppliedConfigAnnotation,
			"argo-server serves this cache to clients, so the annotation must survive")
		assert.Equal(t, "yes", cached.GetAnnotations()["keep-me"])
		assert.Nil(t, cached.GetManagedFields(), "managedFields is loss-free to strip")
	})
	t.Run("ClusterWorkflowTemplate", func(t *testing.T) {
		seeded := annotatedTemplate("ClusterWorkflowTemplate", "")
		client := dynamicfake.NewSimpleDynamicClient(scheme.Scheme, seeded)

		cached := cachedTemplate(t, NewTolerantClusterWorkflowTemplateInformer(client, 0).Informer())

		assert.Contains(t, cached.GetAnnotations(), apiv1.LastAppliedConfigAnnotation,
			"argo-server serves this cache to clients, so the annotation must survive")
		assert.Equal(t, "yes", cached.GetAnnotations()["keep-me"])
		assert.Nil(t, cached.GetManagedFields(), "managedFields is loss-free to strip")
	})
}
