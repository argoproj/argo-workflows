package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var stepsSequentialGroupsWithFailure = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-sequential-groups
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: step-a
        template: echo
    - - name: step-b
        template: echo
    - - name: step-c
        template: echo

  - name: echo
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello"]
`

// TestStepsNoRunningNodesAfterEarlyFailure asserts that a completed workflow contains no
// node still in a running phase. When an early step group fails, the step groups after it
// must either not be created at all or be recorded as fulfilled; leaving them Running
// strands the node tree in a state that contradicts the workflow's final phase and that no
// later reconciliation will ever clean up. See #16290.
func TestStepsNoRunningNodesAfterEarlyFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	wf := wfv1.MustUnmarshalWorkflow(stepsSequentialGroupsWithFailure)
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	makePodsPhase(ctx, woc, apiv1.PodFailed)

	for i := 0; i < 5 && !woc.wf.Status.Phase.Completed(); i++ {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	require.NotEmpty(t, woc.wf.Status.Nodes)
	for _, node := range woc.wf.Status.Nodes {
		assert.Truef(t, node.Fulfilled(), "%s node %q is %s in a completed workflow", node.Type, node.Name, node.Phase)
	}
}
