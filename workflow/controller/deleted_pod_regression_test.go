package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestDeletedRunningPodDoesNotRestart verifies that incomplete task results do
// not turn a deleted pod into an implicit retry, with or without a retry policy.
func TestDeletedRunningPodDoesNotRestart(t *testing.T) {
	for _, tc := range []struct {
		name  string
		retry bool
		dag   bool
	}{
		{name: "steps-without-retry"},
		{name: "steps-with-retry", retry: true},
		{name: "dag-without-retry", dag: true},
		{name: "dag-with-retry", retry: true, dag: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: deleted-running-pod
  namespace: default
spec:
  entrypoint: main
  onExit: exit
  templates:
  - name: main
    steps:
    - - name: run
        template: work
  - name: work
    container:
      image: argoproj/argosay:v2
      command: [echo]
  - name: exit
    container:
      image: argoproj/argosay:v2
      command: [echo]
`)
			if tc.dag {
				wf.Spec.Templates[0].Steps = nil
				wf.Spec.Templates[0].DAG = &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "run", Template: "work"}}}
			}
			if tc.retry {
				wf.Spec.Templates[1].RetryStrategy = &wfv1.RetryStrategy{RetryPolicy: wfv1.RetryPolicyOnError}
			}
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := r4Operate(t, ctx, controller, wf)
			pods, err := listPods(ctx, woc)
			require.NoError(t, err)
			require.Len(t, pods.Items, 1)
			nodeID := woc.nodeID(&pods.Items[0])
			makePodsPhase(ctx, woc, apiv1.PodRunning, r4IncompleteTaskResult(ctx, t))
			woc = r4Operate(t, ctx, controller, woc.wf)
			deletePods(ctx, woc)
			woc = r4Operate(t, ctx, controller, woc.wf)
			pods, err = listPods(ctx, woc)
			require.NoError(t, err)
			require.Empty(t, pods.Items, "must wait for task results instead of recreating the attempt")
			node := woc.wf.Status.Nodes[nodeID]
			assert.Equal(t, wfv1.NodeError, node.Phase)
			assert.Equal(t, "pod deleted", node.Message)
			assert.False(t, node.Fulfilled())

			// Allow the existing missing-pod reconciliation to finish the attempt.
			woc = r4Operate(t, ctx, controller, woc.wf)
			pods, err = listPods(ctx, woc)
			require.NoError(t, err)
			require.Len(t, pods.Items, 1)
			if tc.retry {
				assert.NotEqual(t, nodeID, woc.nodeID(&pods.Items[0]), "a configured retry must use a new attempt")
				makePodsPhase(ctx, woc, apiv1.PodSucceeded)
				woc = r4Operate(t, ctx, controller, woc.wf)
			}
			exitNode, err := woc.wf.GetNodeByName(woc.wf.Name + ".onExit")
			require.NoError(t, err, "exit handler must run")
			assert.Equal(t, wfv1.NodePending, exitNode.Phase)
			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			woc = r4Operate(t, ctx, controller, woc.wf)
			assert.True(t, woc.wf.Status.Phase.Completed())
			if tc.retry {
				assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
			} else {
				assert.Equal(t, wfv1.NodeError, woc.wf.Status.Nodes[nodeID].Phase)
				expected := wfv1.WorkflowFailed
				if tc.dag {
					expected = wfv1.WorkflowError
				}
				assert.Equal(t, expected, woc.wf.Status.Phase)
			}
		})
	}
}
