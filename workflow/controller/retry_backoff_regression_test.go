package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var withItemsRetryBackoffWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: withitems-retry-backoff
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [x]
  - name: fail
    inputs:
      parameters:
      - name: msg
    retryStrategy:
      limit: "2"
      retryPolicy: Always
      backoff:
        duration: "10"
        factor: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, exit 1]
`

// TestWithItemsRetryBackoffKeepsDAGRunning asserts that a withItems task whose
// retry node is waiting out its backoff window keeps the enclosing DAG (and the
// workflow) Running. Between attempts there is no in-flight child, but the retry
// limit is not exhausted, so the last failed attempt must not be treated as the
// task's final phase.
//
// It also pins the documented bare-integer backoff form: `duration: "10"` means
// ten seconds (see RetryStrategy.Backoff docs and examples/retry-backoff.yaml),
// not an unparseable value that collapses the wait to zero.
//
// Regression coverage for the DAG/Steps engine unification (#16290).
func TestWithItemsRetryBackoffKeepsDAGRunning(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(withItemsRetryBackoffWorkflow)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1, "first attempt should be dispatched")

	// Fail the first attempt just now, so the backoff window is still open.
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	attempt := woc.wf.Status.Nodes.FindByDisplayName("A(0:x)(0)")
	require.NotNil(t, attempt)
	attempt.Phase = wfv1.NodeFailed
	attempt.StartedAt = metav1.Time{Time: time.Now().Add(-2 * time.Second)}
	attempt.FinishedAt = metav1.Time{Time: time.Now()}
	woc.wf.Status.Nodes[attempt.ID] = *attempt

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A(0:x)")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeRunning, retryNode.Phase, "retry node should stay Running while backing off")
	assert.Equal(t, "Backoff for 10 seconds", retryNode.Message)

	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1, "no new attempt should be dispatched during the backoff window")

	dagNode := woc.wf.Status.Nodes.FindByDisplayName(wf.Name)
	require.NotNil(t, dagNode)
	assert.Equal(t, wfv1.NodeRunning, dagNode.Phase, "DAG must not fail while a retry is still backing off")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "workflow must not fail while a retry is still backing off")
}
