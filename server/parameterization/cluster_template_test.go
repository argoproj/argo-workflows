package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wffake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
)

func TestClusterTemplateBindingScopeAndFrozenSubmission(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	client := w.client.ArgoprojV1alpha1()
	local, err := client.WorkflowTemplates("lab").Get(ctx, "native", metav1.GetOptions{})
	require.NoError(t, err)
	cluster := &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "native", UID: "cluster-source", ResourceVersion: "3"}, Spec: *local.Spec.DeepCopy()}
	cluster.Spec.Templates[0].Container.Args = []string{"cluster {{workflow.parameters.count}}"}
	_, err = client.ClusterWorkflowTemplates().Create(ctx, cluster, metav1.CreateOptions{})
	require.NoError(t, err)
	localPreview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, []string{"{{workflow.parameters.count}}"}, localPreview.Workflow.Spec.Templates[0].Container.Args)
	require.Equal(t, map[string]string{"workflows.argoproj.io/workflow-template": "native"}, localPreview.Workflow.Labels)
	b, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(b.Object, true, "spec", "clusterScope"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, "lab", preview.Workflow.Namespace)
	require.Equal(t, map[string]string{"workflows.argoproj.io/cluster-workflow-template": "native"}, preview.Workflow.Labels)
	require.Equal(t, "ClusterWorkflowTemplate/cluster-source:3", preview.DefinitionVersions["workflowTemplate"])
	require.Equal(t, []string{"cluster {{workflow.parameters.count}}"}, preview.Workflow.Spec.Templates[0].Container.Args)
	require.NotEqual(t, localPreview.ExecutionDigest, preview.ExecutionDigest)
	h := Handler{Workflows: w}
	req := SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "cluster-bound"}
	cluster.Spec.Templates[0].Container.Args = []string{"changed"}
	_, err = client.ClusterWorkflowTemplates().Update(ctx, cluster, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = h.submit(ctx, s, req)
	require.ErrorContains(t, err, "stale")
	preview, err = s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	req.ExecutionDigest = preview.ExecutionDigest
	_, err = h.submit(ctx, s, req)
	require.NoError(t, err)
	require.NoError(t, client.ClusterWorkflowTemplates().Delete(ctx, "native", metav1.DeleteOptions{}))
	_, err = h.submit(ctx, s, req)
	require.NoError(t, err, "lost response recovers the frozen execution without reading latest source")
	require.Equal(t, 1, w.createCalls)
	_, err = s.Preview(ctx, "standard", revision)
	require.True(t, apierrors.IsNotFound(err), "no fallback to same-name namespaced source")
}

func TestClusterTemplateBindingUsesCallerRBACAndMatchingActionScope(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	b, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(b.Object, true, "spec", "clusterScope"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	fake := w.client.(*wffake.Clientset)
	fake.PrependReactor("get", "clusterworkflowtemplates", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "argoproj.io", Resource: "clusterworkflowtemplates"}, "native", nil)
	})
	_, err = s.Preview(ctx, "standard", revision)
	require.True(t, apierrors.IsForbidden(err))
	fake.ReactionChain = fake.ReactionChain[1:]
	local, err := fake.ArgoprojV1alpha1().WorkflowTemplates("lab").Get(ctx, "native", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = fake.ArgoprojV1alpha1().ClusterWorkflowTemplates().Create(ctx, &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "native"}, Spec: local.Spec}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowactionbindings").Create(ctx, object("WorkflowActionBinding", "calls", "lab", ActionBinding{WorkflowTemplate: "native"}), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(b.Object, "calls", "spec", "actionBinding"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.Preview(ctx, "standard", revision)
	require.ErrorContains(t, err, "different WorkflowTemplate")
	call, err := s.get(ctx, "workflowactionbindings", "calls")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(call.Object, true, "spec", "clusterScope"))
	_, err = s.resource("workflowactionbindings").Update(ctx, call, metav1.UpdateOptions{})
	require.NoError(t, err)
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, local.Spec.Templates, preview.Workflow.Spec.Templates)
	require.Equal(t, "2", preview.Workflow.Spec.Arguments.Parameters[0].Value.String())
}
