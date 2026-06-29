package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestTolerantList(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	good := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "good", Namespace: "ns1", ResourceVersion: "1"},
	}
	good.SetGroupVersionKind(wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowTemplateKind))

	// Malformed: spec.podMetadata.labels.foo is a number, but the typed Go struct
	// declares Labels as map[string]string. Mirrors the real-world failure from
	// minimal-CRD clusters that lack admission-time schema validation.
	broken := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": wfv1.SchemeGroupVersion.String(),
		"kind":       workflow.WorkflowTemplateKind,
		"metadata": map[string]any{
			"name":            "broken",
			"namespace":       "ns1",
			"resourceVersion": "2",
		},
		"spec": map[string]any{
			"podMetadata": map[string]any{
				"labels": map[string]any{"foo": int64(1)},
			},
		},
	}}

	scheme := runtime.NewScheme()
	require.NoError(t, wfv1.AddToScheme(scheme))
	dyn := dynamicfake.NewSimpleDynamicClient(scheme, good, broken)

	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTemplatePlural)
	items, _, err := TolerantList[wfv1.WorkflowTemplate](ctx, dyn, gvr, "ns1", metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items, 1, "malformed item should be skipped, leaving only the well-formed one")
	assert.Equal(t, "good", items[0].Name)
}

func TestTolerantList_PropagatesListError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	scheme := runtime.NewScheme()
	require.NoError(t, wfv1.AddToScheme(scheme))
	dyn := dynamicfake.NewSimpleDynamicClient(scheme)
	dyn.PrependReactor("list", "workflowtemplates", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})

	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTemplatePlural)
	_, _, err := TolerantList[wfv1.WorkflowTemplate](ctx, dyn, gvr, "ns1", metav1.ListOptions{})
	require.Error(t, err)
}

// TestTolerantList_PropagatesListMeta asserts the returned ListMeta carries the
// upstream pagination fields. Callers wrap meta into their *List response to feed
// the Continue token; a dropped token would silently break pagination.
func TestTolerantList_PropagatesListMeta(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	scheme := runtime.NewScheme()
	require.NoError(t, wfv1.AddToScheme(scheme))
	dyn := dynamicfake.NewSimpleDynamicClient(scheme)
	dyn.PrependReactor("list", "workflowtemplates", func(k8stesting.Action) (bool, runtime.Object, error) {
		list := &unstructured.UnstructuredList{}
		list.SetResourceVersion("123")
		list.SetContinue("next-token")
		return true, list, nil
	})

	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTemplatePlural)
	_, meta, err := TolerantList[wfv1.WorkflowTemplate](ctx, dyn, gvr, "ns1", metav1.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, "123", meta.ResourceVersion)
	assert.Equal(t, "next-token", meta.Continue)
}

func TestTolerantList_AllValid(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	scheme := runtime.NewScheme()
	require.NoError(t, wfv1.AddToScheme(scheme))

	a := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "ns1"}}
	a.SetGroupVersionKind(wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowTemplateKind))
	b := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "ns1"}}
	b.SetGroupVersionKind(wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowTemplateKind))

	dyn := dynamicfake.NewSimpleDynamicClient(scheme, a, b)
	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTemplatePlural)
	items, _, err := TolerantList[wfv1.WorkflowTemplate](ctx, dyn, gvr, "ns1", metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

// TestTolerantList_ClearsTypeMeta guards that decoded items carry empty TypeMeta,
// matching the typed clientset this path replaces (its codec strips Kind/APIVersion
// on decode). The JSON roundtrip leaves them populated, which would change every
// list response and break golden tests / list-then-resubmit flows.
func TestTolerantList_ClearsTypeMeta(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	scheme := runtime.NewScheme()
	require.NoError(t, wfv1.AddToScheme(scheme))

	wf := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "ns1"}}
	wf.SetGroupVersionKind(wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowTemplateKind))

	dyn := dynamicfake.NewSimpleDynamicClient(scheme, wf)
	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTemplatePlural)
	items, _, err := TolerantList[wfv1.WorkflowTemplate](ctx, dyn, gvr, "ns1", metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Empty(t, items[0].Kind)
	assert.Empty(t, items[0].APIVersion)
}

// TestTolerantList_PreservesCustomUnmarshalers guards against a regression where
// reflection-based conversion silently dropped every Workflow with `steps`
// because ParallelSteps relies on a custom json.Unmarshaler (it serializes as
// an anonymous list, not a struct). The fix routes per-item decoding through
// json.Unmarshal, which invokes UnmarshalJSON correctly.
func TestTolerantList_PreservesCustomUnmarshalers(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	scheme := runtime.NewScheme()
	require.NoError(t, wfv1.AddToScheme(scheme))

	wf := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": wfv1.SchemeGroupVersion.String(),
		"kind":       workflow.WorkflowKind,
		"metadata": map[string]any{
			"name":            "with-steps",
			"namespace":       "ns1",
			"resourceVersion": "1",
		},
		"spec": map[string]any{
			"entrypoint": "main",
			"templates": []any{
				map[string]any{
					"name": "main",
					// ParallelSteps is an anonymous outer list of inner lists.
					"steps": []any{
						[]any{map[string]any{"name": "a", "template": "echo"}},
					},
				},
			},
		},
	}}

	dyn := dynamicfake.NewSimpleDynamicClient(scheme, wf)
	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowPlural)
	items, _, err := TolerantList[wfv1.Workflow](ctx, dyn, gvr, "ns1", metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items, 1, "workflow with ParallelSteps must be decoded, not dropped")
	require.Len(t, items[0].Spec.Templates, 1)
	require.Len(t, items[0].Spec.Templates[0].Steps, 1)
	require.Len(t, items[0].Spec.Templates[0].Steps[0].Steps, 1)
	assert.Equal(t, "a", items[0].Spec.Templates[0].Steps[0].Steps[0].Name)
}
