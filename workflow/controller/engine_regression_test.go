package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Each test in this file pins a behaviour of main (base 4389bbf96) that the
// Engine refactor changed. They are written to pass at base and fail at HEAD.

// operateUntilFulfilled drives the workflow through operate cycles, marking
// every pod with the given phase between cycles, until it is fulfilled or the
// cycle budget is spent.
func operateUntilFulfilled(t *testing.T, woc *wfOperationCtx, podPhase apiv1.PodPhase, cycles int) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	for range cycles {
		woc.operate(ctx)
		if woc.wf.Status.Fulfilled() {
			return woc
		}
		makePodsPhase(ctx, woc, podPhase)
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
	}
	return woc
}

func nodeExists(woc *wfOperationCtx, name string) bool {
	_, err := woc.wf.GetNodeByName(name)
	return err == nil
}

// 1. Dependants must not be dispatched before a completed task's exit hook
// finishes (#12192), and a task that already succeeded must never get a second
// retry attempt.
var regDependantWaitsForHook = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-hook
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: z
        template: retried
        hooks:
          exit:
            template: echo
      - name: b
        template: echo
        depends: z
  - name: retried
    retryStrategy:
      limit: 1
    container:
      image: argoproj/argosay:v2
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_DependantWaitsForExitHookOfRetriedTask(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regDependantWaitsForHook)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // creates z(0)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // z(0) succeeded -> z succeeded -> z.onExit starts

	hook, err := woc.wf.GetNodeByName("reg-hook.z.onExit")
	require.NoError(t, err, "z.onExit must have been created")
	require.False(t, hook.Fulfilled(), "z.onExit must still be pending in this cycle")
	assert.False(t, nodeExists(woc, "reg-hook.b"), "b must not be dispatched while z.onExit is pending")
	assert.False(t, nodeExists(woc, "reg-hook.z(1)"), "z already succeeded; no second attempt may be created")
}

// 2. A Steps template whose earlier group failed must end Failed even when a
// step in a later group carries continueOn.
var regStepsFailedGroupThenContinueOn = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-steps-cont
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: echo
    - - name: B
        template: echo
        continueOn:
          failed: true
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_StepsFailedGroupIsNotExcusedByLaterContinueOn(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regStepsFailedGroupThenContinueOn)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := operateUntilFulfilled(t, newWorkflowOperationCtx(ctx, wf, controller), apiv1.PodFailed, 6)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	sg0, err := woc.wf.GetNodeByName("reg-steps-cont[0]")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, sg0.Phase)
}

// 3. With failFast: false, one task's hard error must not end the DAG while a
// sibling branch is still running.
var regTaskErrorKeepsSiblingsRunning = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-ff
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      failFast: false
      tasks:
      - name: A
        template: producer
      - name: B
        template: consumer
        depends: A
        arguments:
          artifacts:
          - name: nope
            from: "{{tasks.A.outputs.artifacts.nope}}"
      - name: C
        template: echo
        depends: A
  - name: producer
    container:
      image: argoproj/argosay:v2
    outputs:
      artifacts:
      - name: nope
        path: /tmp/nope
  - name: consumer
    inputs:
      artifacts:
      - name: nope
        path: /tmp/nope
    container:
      image: argoproj/argosay:v2
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_TaskErrorDoesNotAbortDAGWithFailFastFalse(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regTaskErrorKeepsSiblingsRunning)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)                            // A pod
	makePodsPhase(ctx, woc, apiv1.PodSucceeded) // A succeeds but reports no artifact
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // B cannot resolve its artifact; C must still be dispatched

	c, err := woc.wf.GetNodeByName("reg-ff.C")
	require.NoError(t, err, "C must be dispatched")
	assert.False(t, c.Fulfilled(), "C should still be running")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "DAG must keep running while C is in flight")
	dag, err := woc.wf.GetNodeByName("reg-ff")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, dag.Phase)
}

// 4. An exit hook on an expanded task runs once per item, on the item node.
var regExitHookPerItem = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-items
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        withItems: [alpha, beta]
        hooks:
          exit:
            template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_ExitHookRunsPerExpandedItem(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regExitHookPerItem)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := operateUntilFulfilled(t, newWorkflowOperationCtx(ctx, wf, controller), apiv1.PodSucceeded, 10)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.True(t, nodeExists(woc, "reg-items.A(0:alpha).onExit"), "exit hook for item alpha")
	assert.True(t, nodeExists(woc, "reg-items.A(1:beta).onExit"), "exit hook for item beta")
	assert.False(t, nodeExists(woc, "reg-items.A.onExit"), "no group-level exit hook")
}

// 5. An inline template on an expanded step must still be executed.
var regInlineWithItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-inline
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        withItems: [alpha, beta]
        inline:
          container:
            image: argoproj/argosay:v2
            args: ["{{item}}"]
`

func TestRegression_InlineTemplateOnExpandedStep(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regInlineWithItems)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2, "one pod per item")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

// 6. spec.volumes may reference a step's output parameter (docs/variables.md).
var regVolumeFromStepOutput = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-vol
  namespace: default
spec:
  entrypoint: main
  volumes:
  - name: work
    persistentVolumeClaim:
      claimName: "{{steps.gen.outputs.parameters.pvc}}"
  templates:
  - name: main
    steps:
    - - name: gen
        template: gen
    - - name: use
        template: use
  - name: gen
    container:
      image: argoproj/argosay:v2
    outputs:
      parameters:
      - name: pvc
        valueFrom:
          path: /tmp/pvc
  - name: use
    container:
      image: argoproj/argosay:v2
      volumeMounts:
      - name: work
        mountPath: /work
`

func TestRegression_VolumesSubstitutedFromStepOutput(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regVolumeFromStepOutput)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // gen pod
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withOutputs(ctx, wfv1.Outputs{
		Parameters: []wfv1.Parameter{{Name: "pvc", Value: wfv1.AnyStringPtr("my-claim")}},
	}))
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // use pod

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	var found bool
	for _, pod := range pods.Items {
		for _, v := range pod.Spec.Volumes {
			if v.Name == "work" && v.PersistentVolumeClaim != nil {
				found = true
				assert.Equal(t, "my-claim", v.PersistentVolumeClaim.ClaimName)
			}
		}
	}
	assert.True(t, found, "the consuming pod must mount the substituted claim")
}

// 7. {{steps.name}} inside an expanded step is the expanded step name, without
// the engine's internal group prefix.
var regStepsNameExpanded = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-name
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: echo
        withItems: [alpha]
  - name: echo
    container:
      image: argoproj/argosay:v2
      args: ["{{steps.name}}"]
`

func TestRegression_StepsNameForExpandedStep(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regStepsNameExpanded)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	var args []string
	for _, c := range pods.Items[0].Spec.Containers {
		if c.Name == "main" {
			args = c.Args
		}
	}
	assert.Equal(t, []string{"A(0:alpha)"}, args)
}
