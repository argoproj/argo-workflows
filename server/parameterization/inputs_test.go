package parameterization

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

type inputService struct {
	*workflowService
	calls int
}

func (x *inputService) GetWorkflow(ctx context.Context, r *workflowapi.WorkflowGetRequest) (*wfv1.Workflow, error) {
	return x.client.ArgoprojV1alpha1().Workflows(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
}
func (x *inputService) SetWorkflow(ctx context.Context, r *workflowapi.WorkflowSetRequest) (*wfv1.Workflow, error) {
	x.calls++
	var outputs map[string]string
	if err := json.Unmarshal([]byte(r.OutputParameters), &outputs); err != nil {
		return nil, err
	}
	if err := wfutil.SetWorkflow(ctx, x.client.ArgoprojV1alpha1().Workflows(r.Namespace), hydratorfake.Noop, r.Name, r.NodeFieldSelector, wfutil.SetOperationValues{Phase: wfv1.NodePhase(r.Phase), OutputParameters: outputs}); err != nil {
		return nil, err
	}
	return x.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: r.Namespace, Name: r.Name})
}
func TestIntermediateInputUsesFrozenContractNativeSetAndIdempotentReceipt(t *testing.T) {
	ctx, s, w, _ := previewFixture(t)
	service := &inputService{workflowService: w}
	h := Handler{Workflows: service}
	contract := FrozenInputForm{Model: p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"replicas": map[string]any{"type": "integer", "minimum": 1}}, "required": []any{"replicas"}}}, Mappings: []p.Mapping{{Path: "/replicas", Parameter: "count", Encoding: "decimal"}}}
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "waiting", Namespace: "lab", UID: "uid", ResourceVersion: "7", Annotations: map[string]string{p.Group + "/submission": "receipt"}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, Nodes: wfv1.Nodes{"node": {ID: "node", TemplateName: "input", Name: "waiting.input", Type: wfv1.NodeTypeSuspend, Phase: wfv1.NodeRunning, Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "count", ValueFrom: &wfv1.ValueFrom{Supplied: &wfv1.SuppliedValueFrom{}}}}}}}}}
	_, err := w.client.ArgoprojV1alpha1().Workflows("lab").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "receipt", "lab", map[string]any{"workflow": wf, "inputForms": map[string]FrozenInputForm{"input": contract}}), metav1.CreateOptions{})
	require.NoError(t, err)
	pending, err := h.pendingInputs(ctx, s, "waiting")
	require.NoError(t, err)
	require.Len(t, pending.Items, 1)
	request := NodeInputRequest{UID: "uid", ResourceVersion: "7", NodeID: "node", RequestID: "intent", Values: map[string]any{"replicas": 0}}
	_, err = h.submitInput(ctx, s, "waiting", request)
	require.ErrorContains(t, err, "schema validation")
	require.Zero(t, service.calls)
	request.Values = map[string]any{"replicas": 3}
	result, err := h.submitInput(ctx, s, "waiting", request)
	require.NoError(t, err)
	require.Equal(t, wfv1.NodeSucceeded, result.Status.Nodes["node"].Phase)
	require.Equal(t, "3", result.Status.Nodes["node"].Outputs.Parameters[0].Value.String())
	_, err = h.submitInput(ctx, s, "waiting", request)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls)
	pending, err = h.pendingInputs(ctx, s, "waiting")
	require.NoError(t, err)
	require.Empty(t, pending.Items)
	require.Len(t, pending.Submitted, 1)
	request.Values = map[string]any{"replicas": 4}
	_, err = h.submitInput(ctx, s, "waiting", request)
	require.Error(t, err)
	// The native mutation itself checks identity/version at its final write boundary.
	proof := p.WithInputProof(ctx, p.NativeInputProof{UID: "another", ResourceVersion: "7", NodeID: "node", Receipt: "bad"})
	err = wfutil.SetWorkflow(proof, auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows("lab"), hydratorfake.Noop, "waiting", "id=node", wfutil.SetOperationValues{Phase: wfv1.NodeSucceeded})
	require.ErrorContains(t, err, "workflow changed")
}
