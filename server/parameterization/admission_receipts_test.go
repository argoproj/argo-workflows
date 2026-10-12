package parameterization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestAdmissionRejectsUnavailableOrForgedReceiptsWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		edit       func(*unstructured.Unstructured)
		remove     bool
	}{
		{"missing receipt", "cannot read the frozen WorkflowSubmission", nil, true},
		{"malformed receipt", "invalid WorkflowSubmission", func(o *unstructured.Unstructured) { o.Object["spec"] = "invalid" }, false},
		{"changed definitions", "definition snapshot differs", func(o *unstructured.Unstructured) {
			require.NoError(t, unstructured.SetNestedField(o.Object, "another-template", "spec", "definitions", "parameterBinding", "workflowTemplate"))
		}, false},
		{"stale execution digest", "submission is stale", func(o *unstructured.Unstructured) {
			require.NoError(t, unstructured.SetNestedField(o.Object, "stale", "spec", "executionDigest"))
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, store, service, revision := previewFixture(t)
			preview, err := store.Preview(ctx, "standard", revision)
			require.NoError(t, err)
			created, err := (Handler{Workflows: service}).submit(ctx, store, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "receipt-guard"})
			require.NoError(t, err)
			wf := created.(*wfv1.Workflow)
			name := wf.Annotations[p.Group+"/submission"]
			if tc.remove {
				require.NoError(t, store.resource("workflowsubmissions").Delete(ctx, name, metav1.DeleteOptions{}))
			} else {
				receipt, err := store.get(ctx, "workflowsubmissions", name)
				require.NoError(t, err)
				tc.edit(receipt)
				_, err = store.resource("workflowsubmissions").Update(ctx, receipt, metav1.UpdateOptions{})
				require.NoError(t, err)
			}
			client := store.Client.(*dynamicfake.FakeDynamicClient)
			client.ClearActions()
			require.ErrorContains(t, store.ValidateNativeWorkflow(ctx, wf, nil), tc.want)
			for _, action := range client.Actions() {
				require.Contains(t, []string{"get", "list"}, action.GetVerb(), "admission must not create or repair execution evidence")
			}
		})
	}
}

func TestAdmissionRejectsIdentityChangesBeforeReadingReceipts(t *testing.T) {
	old := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "lab", UID: "uid", Annotations: map[string]string{p.Group + "/submission": "receipt"}}}
	for _, next := range []*wfv1.Workflow{nil, {ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "other", UID: "uid"}}, {ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "lab", UID: "replacement"}}} {
		require.Error(t, (Store{Namespace: "lab"}).ValidateNativeWorkflow(context.Background(), next, old))
	}
}
