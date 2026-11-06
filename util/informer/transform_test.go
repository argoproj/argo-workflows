package informer

import (
	"errors"
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

func TestStripLastAppliedConfiguration(t *testing.T) {
	t.Run("typed object", func(t *testing.T) {
		pod := &apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-pod",
				Annotations: map[string]string{
					apiv1.LastAppliedConfigAnnotation: `{"spec":{"containers":[]}}`,
					"keep-me":                         "yes",
				},
			},
		}
		out, err := StripLastAppliedConfiguration(pod)
		require.NoError(t, err)
		assert.Same(t, pod, out)
		assert.NotContains(t, pod.Annotations, apiv1.LastAppliedConfigAnnotation)
		assert.Equal(t, "yes", pod.Annotations["keep-me"], "other annotations must survive")
		assert.Equal(t, "my-pod", pod.Name)
	})
	t.Run("unstructured object", func(t *testing.T) {
		un := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "WorkflowTemplate",
			"metadata": map[string]any{
				"name": "my-template",
				"annotations": map[string]any{
					apiv1.LastAppliedConfigAnnotation: `{"spec":{}}`,
					"keep-me":                         "yes",
				},
			},
		}}
		out, err := StripLastAppliedConfiguration(un)
		require.NoError(t, err)
		assert.Same(t, un, out)
		assert.NotContains(t, un.GetAnnotations(), apiv1.LastAppliedConfigAnnotation)
		assert.Equal(t, "yes", un.GetAnnotations()["keep-me"])
		assert.Equal(t, "my-template", un.GetName())
	})
	t.Run("an object carrying other annotations keeps all of them", func(t *testing.T) {
		pod := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name:        "my-pod",
			Annotations: map[string]string{"keep-me": "yes", "and-me": "also"},
		}}
		want := pod.DeepCopy()
		out, err := StripLastAppliedConfiguration(pod)
		require.NoError(t, err)
		assert.Same(t, pod, out)
		assert.Equal(t, want, pod, "an object that never carried the annotation must come back unchanged")
	})
	t.Run("object with no annotations at all", func(t *testing.T) {
		un := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "WorkflowTemplate",
			"metadata":   map[string]any{"name": "my-template"},
		}}
		want := un.DeepCopy()
		out, err := StripLastAppliedConfiguration(un)
		require.NoError(t, err)
		assert.Same(t, un, out)
		assert.Equal(t, want, un)
	})
	t.Run("non-object input passes through", func(t *testing.T) {
		tombstone := cache.DeletedFinalStateUnknown{Key: "my-ns/my-template"}
		out, err := StripLastAppliedConfiguration(tombstone)
		require.NoError(t, err)
		assert.Equal(t, tombstone, out)
	})
}

// templateWithBothFields is the shape these transforms exist for: a template
// applied with client-side `kubectl apply`, so it carries both managedFields
// and the annotation holding a copy of the whole spec.
func templateWithBothFields() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "WorkflowTemplate",
		"metadata": map[string]any{
			"name":          "my-template",
			"managedFields": []any{map[string]any{"manager": "kubectl-client-side-apply"}},
			"annotations": map[string]any{
				apiv1.LastAppliedConfigAnnotation: `{"spec":{}}`,
				"keep-me":                         "yes",
			},
		},
	}}
}

// Both transforms drop managedFields; they differ only over the apply
// annotation, and that difference is the point. The shared tolerant
// constructors use StripManagedFields and argo-server builds those same
// informers to serve templates to clients, so stripping the annotation there
// would let a get/modify/update round trip drop it from the live object. Only
// the controller, which never writes these objects back, may use
// StripTemplateCacheFields.
func TestTransformsDifferOverTheApplyAnnotation(t *testing.T) {
	for _, tt := range []struct {
		name               string
		transform          cache.TransformFunc
		keepsTheAnnotation bool
		why                string
	}{
		{"StripManagedFields", StripManagedFields, true, "argo-server serves this cache to clients"},
		{"StripTemplateCacheFields", StripTemplateCacheFields, false, "the controller only ever reads its cache"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			un := templateWithBothFields()
			out, err := tt.transform(un)
			require.NoError(t, err)
			assert.Same(t, un, out)
			assert.Nil(t, un.GetManagedFields(), "managedFields is loss-free to strip either way")
			if tt.keepsTheAnnotation {
				assert.Contains(t, un.GetAnnotations(), apiv1.LastAppliedConfigAnnotation, tt.why)
			} else {
				assert.NotContains(t, un.GetAnnotations(), apiv1.LastAppliedConfigAnnotation, tt.why)
			}
			assert.Equal(t, "yes", un.GetAnnotations()["keep-me"], "unrelated annotations always survive")
			assert.Equal(t, "my-template", un.GetName())
		})
	}
}

// chain must stop at the first error and hand back what that transform
// returned, rather than running the rest over a half-transformed object.
func TestChainStopsAtFirstError(t *testing.T) {
	boom := errors.New("boom")
	var ranAfter bool
	failing := func(i any) (any, error) { return "halfway", boom }
	after := func(i any) (any, error) { ranAfter = true; return i, nil }

	got, err := chain(StripManagedFields, failing, after)("start")
	require.ErrorIs(t, err, boom)
	assert.Equal(t, "halfway", got, "the failing transform's value is returned, not the input")
	assert.False(t, ranAfter, "transforms after a failure must not run")
}

// chain threads each transform's output into the next.
func TestChainThreadsOutput(t *testing.T) {
	appendA := func(i any) (any, error) { return i.(string) + "a", nil }
	appendB := func(i any) (any, error) { return i.(string) + "b", nil }
	got, err := chain(appendA, appendB)("x")
	require.NoError(t, err)
	assert.Equal(t, "xab", got)
}
