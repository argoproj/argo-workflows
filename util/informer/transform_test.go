package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

func TestStripManagedFields(t *testing.T) {
	t.Run("typed object", func(t *testing.T) {
		pod := &apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:          "my-pod",
				ManagedFields: []metav1.ManagedFieldsEntry{{Manager: "kubelet"}},
			},
		}
		out, err := StripManagedFields(pod)
		require.NoError(t, err)
		assert.Same(t, pod, out)
		assert.Nil(t, pod.ManagedFields)
		assert.Equal(t, "my-pod", pod.Name)
	})
	t.Run("unstructured object", func(t *testing.T) {
		un := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]any{
				"name":          "my-wf",
				"managedFields": []any{map[string]any{"manager": "workflow-controller"}},
			},
		}}
		out, err := StripManagedFields(un)
		require.NoError(t, err)
		assert.Same(t, un, out)
		assert.Nil(t, un.GetManagedFields())
		assert.NotContains(t, un.Object["metadata"], "managedFields")
		assert.Equal(t, "my-wf", un.GetName())
	})
	t.Run("nil managedFields is not mutated", func(t *testing.T) {
		un := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata":   map[string]any{"name": "my-wf"},
		}}
		want := un.DeepCopy()
		out, err := StripManagedFields(un)
		require.NoError(t, err)
		assert.Same(t, un, out)
		assert.Equal(t, want, un)
	})
	t.Run("non-object input passes through", func(t *testing.T) {
		tombstone := cache.DeletedFinalStateUnknown{Key: "my-ns/my-pod"}
		out, err := StripManagedFields(tombstone)
		require.NoError(t, err)
		assert.Equal(t, tombstone, out)
	})
}
