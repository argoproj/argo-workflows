package controller

import (
	"testing"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func actionTestSchema(t *testing.T) string {
	encoded, err := p.FreezeActionContract(map[string]any{"type": "object", "properties": map[string]any{"count": map[string]any{"type": "integer", "minimum": 1}}, "required": []any{"count"}})
	require.NoError(t, err)
	return encoded
}
func TestParameterActionRejectsInputBeforePodCreation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: {name: action-contract, namespace: default}
spec:
  entrypoint: action
  serviceAccountName: default
  arguments:
    parameters: [{name: count, value: "0"}]
  templates:
  - name: action
    inputs:
      parameters: [{name: count}]
    container: {image: alpine:3.22, command: [echo, invalid-must-not-execute]}
`)
	wf.Spec.Templates[0].Metadata.Annotations = map[string]string{p.InputContractAnnotation: actionTestSchema(t)}
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := controller.kubeclientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, pods.Items)
	require.Contains(t, woc.wf.Status.Message, "action contract rejected")
}
func TestParameterActionRejectsPodOutputsBeforeSuccess(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	val := wfv1.AnyString("0")
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "action", Namespace: "default"}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "call", Metadata: wfv1.Metadata{Annotations: map[string]string{p.OutputContractAnnotation: actionTestSchema(t)}}}}}}
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	node := &wfv1.NodeStatus{ID: "call", Name: "action.call", TemplateName: "call", Type: wfv1.NodeTypePod, Phase: wfv1.NodeRunning, Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "count", Value: &val}}}}
	woc.wf.Status.Nodes = wfv1.Nodes{node.ID: *node}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "action-call", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}
	got := woc.assessNodeStatus(ctx, pod, node)
	require.NotNil(t, got)
	require.Equal(t, wfv1.NodeError, got.Phase)
	require.Nil(t, got.Outputs)
	require.Contains(t, got.Message, "action contract rejected")
}
func TestParameterActionRejectsHTTPOutputsBeforeSuccess(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	contract, err := p.FreezeActionContract(map[string]any{"type": "object", "properties": map[string]any{"result": map[string]any{"type": "boolean"}}, "required": []any{"result"}})
	require.NoError(t, err)
	value := "invalid"
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "http-action", Namespace: "default"}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "call", HTTP: &wfv1.HTTP{URL: "http://synthetic.local"}, Metadata: wfv1.Metadata{Annotations: map[string]string{p.OutputContractAnnotation: contract}}}}}, Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"call": wfv1.NodeStatus{ID: "call", Name: "http-action.call", TemplateName: "call", Type: wfv1.NodeTypeHTTP, Phase: wfv1.NodeRunning}}}}
	ts := &wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Name: wf.Name, Namespace: wf.Namespace}, Status: wfv1.WorkflowTaskSetStatus{Nodes: map[string]wfv1.NodeResult{"call": {Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{Result: &value}}}}}
	cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	require.NoError(t, woc.reconcileTaskSet(ctx))
	got := woc.wf.Status.Nodes["call"]
	require.Equal(t, wfv1.NodeError, got.Phase)
	require.Nil(t, got.Outputs)
}

func TestParameterActionTaskResultReplayKeepsRejectedOutputsHidden(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	val := wfv1.AnyString("0")
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "action-replay", Namespace: "default"}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "call", Metadata: wfv1.Metadata{Annotations: map[string]string{p.OutputContractAnnotation: actionTestSchema(t)}}}}}, Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"call": {ID: "call", Name: "action-replay.call", TemplateName: "call", Type: wfv1.NodeTypePod, Phase: wfv1.NodeError, Message: "action contract rejected field /count"}}}}
	result := &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{Name: "call", Namespace: "default", Labels: map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyReportOutputsCompleted: "true"}}, NodeResult: wfv1.NodeResult{Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "count", Value: &val}}}}}
	cancel, controller := newController(ctx, wf, result, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.taskResultReconciliation(ctx)
	woc.taskResultReconciliation(ctx)
	got := woc.wf.Status.Nodes["call"]
	require.Equal(t, wfv1.NodeError, got.Phase)
	require.Nil(t, got.Outputs)
}

func TestParameterActionCapturesScriptResultWithoutConsumer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: {name: script-contract, namespace: default}
spec:
  entrypoint: call
  serviceAccountName: default
  templates:
  - name: call
    script: {image: alpine:3.22, command: [sh], source: "echo 1"}
`)
	contract, err := p.FreezeActionContract(map[string]any{"type": "object", "properties": map[string]any{"result": map[string]any{"type": "integer"}}, "required": []any{"result"}})
	require.NoError(t, err)
	wf.Spec.Templates[0].Metadata.Annotations = map[string]string{p.OutputContractAnnotation: contract}
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := controller.kubeclientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	for _, c := range pods.Items[0].Spec.Containers {
		if c.Name == common.WaitContainerName {
			require.Contains(t, c.Env, corev1.EnvVar{Name: common.EnvVarIncludeScriptOutput, Value: "true"})
			return
		}
	}
	t.Fatal("native output collector missing")
}

func TestExecutionSubstitutionPreservesSubmittedSpec(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: {name: expression-contract, namespace: default}
spec:
  entrypoint: approve
  serviceAccountName: default
  volumes:
  - name: recovery
    configMap: {name: "{{workflow.name}}-recovery", optional: true}
  templates:
  - name: approve
    suspend: {}
`)
	before := wf.Spec.DeepCopy()
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	require.Equal(t, *before, woc.wf.Spec)
	require.Equal(t, *before, wf.Spec)
	require.Equal(t, "expression-contract-recovery", woc.execWf.Spec.Volumes[0].ConfigMap.Name)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	require.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[wf.Name].Phase)
	require.Equal(t, wfv1.NodeRunning, woc.execWf.Status.Nodes[wf.Name].Phase)
	// A second reconciliation must resolve the same submitted expressions.
	next := newWorkflowOperationCtx(ctx, woc.wf, controller)
	next.operate(ctx)
	require.Equal(t, *before, next.wf.Spec)
	require.Equal(t, "expression-contract-recovery", next.execWf.Spec.Volumes[0].ConfigMap.Name)
}
