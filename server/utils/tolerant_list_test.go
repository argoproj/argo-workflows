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
