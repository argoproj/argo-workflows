package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/server/auth"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	native "github.com/argoproj/argo-workflows/v4/workflow/util"
)

func TestSelectedRetryKeepsUnrelatedFailureAndApproval(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(map[bool]string{false: "original", true: "changed"}[changed], func(t *testing.T) {
			w, n := partialFixture()
			for _, x := range []*wfv1.Workflow{w, n} {
				x.Spec.Templates[0].DAG.Tasks = append(x.Spec.Templates[0].DAG.Tasks, wfv1.DAGTask{Name: "other", Template: "keep"})
				node := wfv1.NodeStatus{ID: "o", Name: "wf.other", TemplateName: "keep", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeFailed}
				x.Status.Nodes["o"] = node
				root := x.Status.Nodes["wf"]
				root.Children = append(root.Children, "o")
				x.Status.Nodes["wf"] = root
			}
			if !changed {
				n = w.DeepCopy()
			}
			ctx := logging.TestContext(t.Context())
			before := w.DeepCopy()
			plan, err := planPartialRetry(ctx, w, n, "change")
			require.NoError(t, err)
			wire, err := json.Marshal(plan)
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(wire, &document))
			require.IsType(t, []any{}, document["changedParameters"])
			require.IsType(t, []any{}, document["parameters"])
			require.Equal(t, []string{"change"}, plan.AffectedTasks)
			require.Contains(t, plan.RetainedNodes, "o")
			require.Contains(t, plan.RetainedNodes, "a")
			after, _, err := native.FormulateRetryWorkflow(p.WithRetryProof(ctx, p.NativeRetryProof{UID: string(w.UID), ResourceVersion: w.ResourceVersion, RetainedNodes: plan.RetainedNodes}), w, true, plan.Selector, plan.Parameters)
			require.NoError(t, err)
			require.Equal(t, w.UID, after.UID)
			require.Equal(t, w.Status.Nodes["o"], after.Status.Nodes["o"])
			require.Equal(t, p.ResultFingerprint(w.Status.Nodes["a"]), p.ResultFingerprint(after.Status.Nodes["a"]))
			require.NotContains(t, after.Status.Nodes, "c")
			require.Equal(t, before, w)
			// Native callers without a checked proof retain their original behavior.
			plain, _, err := native.FormulateRetryWorkflow(ctx, w, true, plan.Selector, plan.Parameters)
			require.NoError(t, err)
			require.NotContains(t, plain.Status.Nodes, "o")
		})
	}
}

func TestSelectedRetryRejectsAmbiguityAndFailedDependencies(t *testing.T) {
	for _, paths := range [][]string{{"missing"}, {"change", "change"}, {"/change"}, {""}} {
		w, _ := partialFixture()
		_, err := planPartialRetry(logging.TestContext(t.Context()), w, w, paths...)
		require.Error(t, err)
	}
	w, _ := partialFixture()
	x := w.Status.Nodes["k"]
	x.Phase = wfv1.NodeFailed
	w.Status.Nodes["k"] = x
	_, err := planPartialRetry(logging.TestContext(t.Context()), w, w, "change")
	require.ErrorContains(t, err, "not successfully completed")
}

func TestDirectGlobalResourceReferenceHasLocalImpact(t *testing.T) {
	w, n := partialFixture()
	for _, x := range []*wfv1.Workflow{w, n} {
		x.Spec.Templates[0].DAG.Tasks[2].Arguments = wfv1.Arguments{}
		x.Spec.Templates[3].Resource = &wfv1.ResourceTemplate{Action: "get", Manifest: "{{workflow.parameters.value}}"}
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
	require.Contains(t, plan.RetainedNodes, "a")
	require.Contains(t, plan.RetainedNodes, "k")
}

func TestCheckedRetryPreservesArgumentOrderAndMetadata(t *testing.T) {
	w, next := partialFixture()
	for _, x := range []*wfv1.Workflow{w, next} {
		x.Spec.Arguments.Parameters = append([]wfv1.Parameter{{Name: "stable", Value: wfv1.AnyStringPtr("kept")}}, x.Spec.Arguments.Parameters...)
		x.Spec.Arguments.Parameters[1].Description = wfv1.AnyStringPtr("declared input")
	}
	ctx := logging.TestContext(t.Context())
	plan, err := planPartialRetry(ctx, w, next, "change")
	require.NoError(t, err)
	after, _, err := native.FormulateRetryWorkflow(p.WithRetryProof(ctx, p.NativeRetryProof{UID: string(w.UID), ResourceVersion: w.ResourceVersion, RetainedNodes: plan.RetainedNodes}), w, true, plan.Selector, plan.Parameters)
	require.NoError(t, err)
	require.Equal(t, next.Spec.Arguments, after.Spec.Arguments)
}

func TestOrderingOnlyDependencyKeepsApprovalButNotOutputConsumer(t *testing.T) {
	w, next := partialFixture()
	for _, x := range []*wfv1.Workflow{w, next} {
		x.Spec.Templates[0].DAG.Tasks = append(x.Spec.Templates[0].DAG.Tasks, wfv1.DAGTask{Name: "later-approval", Template: "approve", Dependencies: []string{"change"}})
		for i := range x.Spec.Templates {
			if x.Spec.Templates[i].Name == "approve" {
				x.Spec.Templates[i].Metadata.Annotations = map[string]string{p.Group + "/retry-dependencies": "inputs"}
			}
		}
		x.Status.Nodes["later"] = wfv1.NodeStatus{ID: "later", Name: "wf.later-approval", Type: wfv1.NodeTypeSuspend, TemplateName: "approve", BoundaryID: "wf", Phase: wfv1.NodeSucceeded}
		parent := x.Status.Nodes["c"]
		parent.Children = append(parent.Children, "later")
		x.Status.Nodes["c"] = parent
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, next, "change")
	require.NoError(t, err)
	require.Contains(t, plan.RetainedNodes, "later")
	// Original-input replay after the entire DAG succeeded must preserve the
	// same completed descendant decision, not just independent root siblings.
	successful := w.DeepCopy()
	successful.Status.Phase = wfv1.WorkflowSucceeded
	for id, node := range successful.Status.Nodes {
		node.Phase = wfv1.NodeSucceeded
		successful.Status.Nodes[id] = node
	}
	original, err := planPartialRetry(logging.TestContext(t.Context()), successful, successful.DeepCopy(), "change")
	require.NoError(t, err)
	require.Contains(t, original.RetainedNodes, "later")
	for _, x := range []*wfv1.Workflow{w, next} {
		x.Spec.Templates[0].DAG.Tasks[len(x.Spec.Templates[0].DAG.Tasks)-1].Arguments.Parameters = []wfv1.Parameter{{Name: "value", Value: wfv1.AnyStringPtr("{{tasks.change.outputs.parameters.result}}")}}
	}
	graph, err := newRetryGraph(&w.Spec, []string{"value"}, "change")
	require.NoError(t, err)
	require.Contains(t, graph.affectedLeaves(), "later-approval")
}

func TestSelectedRetryReevaluatesOmittedInvocations(t *testing.T) {
	for _, composite := range []bool{false, true} {
		t.Run(fmt.Sprint(composite), func(t *testing.T) {
			w, _ := partialFixture()
			callee := "approve"
			want := "later"
			w.Spec.Templates[1].Metadata.Annotations = map[string]string{p.Group + "/retry-dependencies": "inputs"}
			if composite {
				callee = "later-group"
				want = "later/approve"
				w.Spec.Templates = append(w.Spec.Templates, wfv1.Template{Name: callee,
					Metadata: wfv1.Metadata{Annotations: map[string]string{p.Group + "/retry-dependencies": "inputs"}},
					DAG:      &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "approve", Template: "approve"}}}})
			}
			w.Spec.Templates[0].DAG.Tasks = append(w.Spec.Templates[0].DAG.Tasks,
				wfv1.DAGTask{Name: "later", Template: callee, Dependencies: []string{"change"}})
			w.Status.Nodes["later"] = wfv1.NodeStatus{ID: "later", Name: "wf.later", BoundaryID: "wf", TemplateName: callee,
				Type: wfv1.NodeTypeSkipped, Phase: wfv1.NodeOmitted}
			c := w.Status.Nodes["c"]
			c.Children = []string{"later"}
			w.Status.Nodes["c"] = c
			ctx := logging.TestContext(t.Context())
			plan, err := planPartialRetry(ctx, w, w.DeepCopy(), "change")
			require.NoError(t, err)
			require.ElementsMatch(t, []string{"change", want}, plan.AffectedTasks)
			require.NotContains(t, plan.RetainedNodes, "later", "not executed is not an approved result")
			require.Contains(t, plan.RetainedNodes, "a", "completed approval remains valid")
			after, _, err := native.FormulateRetryWorkflow(p.WithRetryProof(ctx, p.NativeRetryProof{UID: string(w.UID), ResourceVersion: w.ResourceVersion, RetainedNodes: plan.RetainedNodes}), w, true, plan.Selector, plan.Parameters)
			require.NoError(t, err)
			require.NotContains(t, after.Status.Nodes, "later")
			require.Equal(t, p.ResultFingerprint(w.Status.Nodes["a"]), p.ResultFingerprint(after.Status.Nodes["a"]))
		})
	}
}

type selectedRetryService struct {
	workflowapi.WorkflowServiceServer
	wf           *wfv1.Workflow
	store        Store
	calls        int
	loseResponse bool
}

func (s *selectedRetryService) GetWorkflow(context.Context, *workflowapi.WorkflowGetRequest) (*wfv1.Workflow, error) {
	return s.wf.DeepCopy(), nil
}
func (s *selectedRetryService) RetryWorkflow(ctx context.Context, req *workflowapi.WorkflowRetryRequest) (*wfv1.Workflow, error) {
	s.calls++
	after, _, err := native.FormulateRetryWorkflow(ctx, s.wf, req.RestartSuccessful, req.NodeFieldSelector, req.Parameters)
	if err != nil {
		return nil, err
	}
	proof, _ := p.RetryProof(ctx)
	after.Annotations[p.Group+"/retry"] = proof.Receipt
	if proof.Revision != "" {
		after.Annotations[p.Group+"/revision"] = proof.Revision
	}
	if err = s.store.ValidateNativeWorkflow(ctx, after, s.wf); err != nil {
		return nil, err
	}
	after.ResourceVersion = "18"
	s.wf, err = auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.store.Namespace).Update(ctx, after, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if s.loseResponse {
		s.loseResponse = false
		return nil, fmt.Errorf("response lost")
	}
	return s.wf.DeepCopy(), nil
}
func TestSelectedOriginalRetryReceiptAdmissionAndLostResponse(t *testing.T) {
	ctx, s, _, revision := previewFixture(t)
	wf, _ := partialFixture()
	wf.Namespace = s.Namespace
	wf.Annotations = map[string]string{p.Group + "/revision": revision, p.Group + "/execution-digest": "frozen"}
	_, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	service := &selectedRetryService{wf: wf, store: s, loseResponse: true}
	h := Handler{Workflows: service}
	plan, _, err := h.partialRetryPreview(ctx, s, PartialRetryRequest{Workflow: wf.Name, UID: string(wf.UID), CallPaths: []string{"change"}})
	require.NoError(t, err)
	req := RetryRequest{Workflow: wf.Name, UID: string(wf.UID), ResourceVersion: wf.ResourceVersion, CallPaths: []string{"change"}, PlanDigest: plan.Digest, IdempotencyKey: "selected-original"}
	_, err = h.retry(ctx, s, req)
	require.ErrorContains(t, err, "response lost")
	result, err := h.retry(ctx, s, req)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls)
	require.Equal(t, wf.UID, result.UID)
	require.Equal(t, p.ResultFingerprint(wf.Status.Nodes["k"]), p.ResultFingerprint(result.Status.Nodes["k"]))
	require.NotContains(t, result.Status.Nodes, "c")
	forged := result.DeepCopy()
	delete(forged.Status.Nodes, "k")
	require.Error(t, s.ValidateNativeWorkflow(ctx, forged, wf), "admission must independently retain unrelated evidence")
	req.CallPaths = []string{"keep"}
	_, err = h.retry(ctx, s, req)
	require.Error(t, err, "idempotency key cannot change target")
}
