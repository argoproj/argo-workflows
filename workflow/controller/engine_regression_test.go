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

// 1b. The same hazard with no hook at all: a dependant that sorts before its
// retried dependency must not be dispatched in the pass that finalizes the
// retry node, or the retry node sees an unfulfilled descendant and starts a
// second attempt for a task that already succeeded.
var regNoSecondAttempt = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-retry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: z
        template: retried
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

func TestRegression_NoSecondAttemptAfterSuccessWithDependant(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regNoSecondAttempt)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // creates z(0)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // z(0) succeeded -> z succeeded -> b dispatched

	z, err := woc.wf.GetNodeByName("reg-retry.z")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, z.Phase)
	assert.True(t, nodeExists(woc, "reg-retry.b"), "b runs once z has succeeded")
	assert.False(t, nodeExists(woc, "reg-retry.z(1)"), "z already succeeded; no second attempt may be created")
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

// 9. A task whose only dependency is omitted in the same pass must still be
// linked under that dependency's (Omitted) node, not left with no parent.
var regOrphanedDependant = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-orphan
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        template: echo
        depends: A
      - name: C
        template: echo
        depends: B.Omitted
        when: "false"
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_DependantOfOmittedTaskIsLinked(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regOrphanedDependant)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // A pod
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // A failed -> B omitted -> C skipped

	b, err := woc.wf.GetNodeByName("reg-orphan.B")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeOmitted, b.Phase)
	c, err := woc.wf.GetNodeByName("reg-orphan.C")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSkipped, c.Phase)
	assert.Contains(t, b.Children, c.ID, "C must hang off its dependency B")
}

// 10. A StepGroup stays Running until every step in it has finished, even
// after one step has failed; it is only then marked Failed.
var regStepGroupWaitsForSiblings = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-sg
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: echo
      - name: B
        template: echo
    - - name: C
        template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_StepGroupWaitsForRunningSiblings(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regStepGroupWaitsForSiblings)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // A and B pods
	setPodPhases(ctx, woc, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if node.Name == "reg-sg[0].A" {
			return apiv1.PodFailed
		}
		return apiv1.PodRunning // B keeps running
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	sg0, err := woc.wf.GetNodeByName("reg-sg[0]")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, sg0.Phase, "group must wait for B before it is Failed")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	b, err := woc.wf.GetNodeByName("reg-sg[0].B")
	require.NoError(t, err)
	if sg1, getErr := woc.wf.GetNodeByName("reg-sg[1]"); getErr == nil {
		assert.NotContains(t, b.Children, sg1.ID, "the next group must not be linked under a still-running step")
	}

	setPodPhases(ctx, woc, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if node.Name == "reg-sg[0].B" {
			return apiv1.PodSucceeded // B finishes
		}
		return ""
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	sg0, err = woc.wf.GetNodeByName("reg-sg[0]")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, sg0.Phase)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// 11. A throttle (parallelism reached) while starting an exit hook is not an
// error of the task or the boundary; the hook is simply tried again later.
var regHookThrottled = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reg-throttle
  namespace: default
spec:
  entrypoint: main
  parallelism: 2
  templates:
  - name: main
    dag:
      tasks:
      - name: a-long
        template: echo
      - name: b-nested
        template: nested
        hooks:
          exit:
            template: echo
      - name: c-x
        template: echo
  - name: nested
    dag:
      tasks:
      - name: skip
        template: echo
        when: "false"
  - name: echo
    container:
      image: argoproj/argosay:v2
`

func TestRegression_ThrottledExitHookIsNotAnError(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(regHookThrottled)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // a-long and c-x take both slots; b-nested completes at once and its hook is throttled

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	nested, err := woc.wf.GetNodeByName("reg-throttle.b-nested")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, nested.Phase)
	root, err := woc.wf.GetNodeByName("reg-throttle")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, root.Phase, "a throttled hook must not error the boundary")

	woc = operateUntilFulfilled(t, woc, apiv1.PodSucceeded, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	hook, err := woc.wf.GetNodeByName("reg-throttle.b-nested.onExit")
	require.NoError(t, err, "the hook runs once a slot is free")
	assert.Equal(t, wfv1.NodeSucceeded, hook.Phase)
}
