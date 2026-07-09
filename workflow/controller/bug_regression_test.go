package controller

// Regression tests documenting known bugs from the code review of
// branch dag-refactor-engine-v4.  Each test asserts the CORRECT (post-fix)
// behavior; all the documented bugs are now fixed, so these tests lock the
// fixes in place.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
)

// TestBug_Depends_NegationCausesPrematureOmit documents Critical #2 as it
// actually manifests at runtime.
//
// workflow/common/dag/argo.go:253-274 uses an "all fields true" value as
// the best-case scope for pending deps, in order to decide whether the
// depends expression is structurally unsatisfiable.  For any negated
// reference (e.g. "!X.Failed"), all-true is the WORST case, not the best:
// it sets X.Failed to true, which makes !X.Failed false.
//
// Consequence: on the first operate cycle, when no task has started yet,
// a task whose depends expression negates a sibling is marked Omitted
// before the sibling has a chance to run.  createOmittedNodes persists
// that verdict into wf.Status.Nodes, so subsequent cycles never
// re-evaluate the task — even after the deps complete successfully.
//
// The correct behavior: wait while any realistic future outcome of the
// pending deps could still make the expression true.
func TestBug_Depends_NegationCausesPrematureOmit(t *testing.T) {
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: negation-premature-omit
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fast-task
        template: echo
      - name: slow-task
        template: echo
      - name: dependent
        depends: "fast-task.Succeeded && !slow-task.Failed"
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: fast-task and slow-task scheduled.  Dependent must not be
	// prematurely omitted — its expression can still become true.
	woc.operate(ctx)
	require.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("fast-task"))
	require.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("slow-task"))
	if dep := woc.wf.Status.Nodes.FindByDisplayName("dependent"); dep != nil {
		assert.NotEqual(t, wfv1.NodeOmitted, dep.Phase,
			"dependent must not be prematurely omitted in cycle 1")
	}

	// Cycle 2: both pods succeed.  Dependent's expression evaluates to
	// `true && !false = true`, so dependent should be scheduled.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	dep := woc.wf.Status.Nodes.FindByDisplayName("dependent")
	require.NotNil(t, dep, "dependent should exist after cycle 2")
	assert.NotEqual(t, wfv1.NodeOmitted, dep.Phase,
		"dependent must run once fast-task succeeded and slow-task did not fail")

	// Cycle 3: dependent pod succeeds.  Workflow completes Succeeded.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	dep = woc.wf.Status.Nodes.FindByDisplayName("dependent")
	require.NotNil(t, dep)
	assert.Equal(t, wfv1.NodeSucceeded, dep.Phase,
		"dependent must end Succeeded")
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"workflow should end Succeeded, not stuck with an omitted dependent")
}

// TestBug_DAGTargetDoesNotScheduleUnrelatedRoots documents a target-filtering
// regression in the shared engine. A DAG target should schedule the target task
// and its ancestors. It must not schedule unrelated roots outside the target's
// ancestry.
func TestBug_DAGTargetDoesNotScheduleUnrelatedRoots(t *testing.T) {
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: target-filtering
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      target: deploy
      tasks:
      - name: build
        template: echo
      - name: deploy
        dependencies: [build]
        template: echo
      - name: unrelated
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	build := woc.wf.Status.Nodes.FindByDisplayName("build")
	require.NotNil(t, build, "target ancestor should be scheduled")
	assert.Equal(t, wfv1.NodePending, build.Phase)

	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("deploy"),
		"target task should wait until its ancestor is fulfilled")
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("unrelated"),
		"unrelated root outside dag.target ancestry must not be scheduled")
}

// TestBug_ReconcileErrorMasking documents Critical #3.
//
// Two collaborating sites silently reclassify genuine reconciler failures
// as ErrParallelismReached:
//
//   - workflow/controller/reconciler_k8s.go:55-58
//     Reconcile() returns nil on ErrParallelismReached / ErrResourceRateLimitReached.
//     The converge loop treats the task as dispatched even though no pod
//     was created — fine for deliberate throttling, wrong for anything else.
//
//   - workflow/controller/operator.go:2193-2199
//     reconcileTemplate() converts any "node not found after reconciliation"
//     into ErrParallelismReached at Debug level.  A silently-missing node
//     from a non-throttle path produces identical log output to legitimate
//     throttling.
//
// A proper fix would:
//
//   - Introduce a distinct sentinel such as ErrReconcilerNoMaterialize for
//     "reconciler returned nil but the expected node was not created".
//   - Raise the log level to Warn when that sentinel is used.
//   - Leave ErrParallelismReached exclusively for the deliberate throttling
//     path.
func TestBug_ReconcileErrorMasking(t *testing.T) {
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: missing-materialized-node
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	tmpl := woc.execWf.GetTemplateByName("main")
	require.NotNil(t, tmpl)
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	mainNode := &wfv1.NodeStatus{
		ID:           woc.wf.NodeID(wf.Name),
		Name:         wf.Name,
		DisplayName:  wf.Name,
		TemplateName: tmpl.Name,
		Type:         wfv1.NodeTypeDAG,
		Phase:        wfv1.NodeRunning,
	}
	woc.wf.Status.Nodes = wfv1.Nodes{mainNode.ID: *mainNode}

	engine := NewEngine(woc, mainNode.Name, tmplCtx, tmpl, mainNode, mainNode.ID, false)
	engine.reconciler = &fakeReconciler{}
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, []dag.Task{
		&dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[0]},
	}, tmpl, mainNode.ID, mainNode.Name)

	node, err := engine.executeTask(ctx, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[0]}, true)

	require.Error(t, err,
		"engine must surface a reconciler materialization failure when Reconcile returns nil but no task node exists")
	assert.Nil(t, node)
	assert.NotErrorIs(t, err, ErrParallelismReached,
		"missing materialization must not be reported as ordinary parallelism throttling")
}

// TestBug_HookFailureKillsDAG documents Critical #4.
//
// engine.go:65-69 bubbles any err from the first processHooks pass into
// markBoundaryError and returns.  A single task's hook that fails template
// resolution (missing template, unresolvable argument, etc.) aborts hook
// processing for every other task in the same operate cycle and marks the
// boundary Error.
//
// The legacy controller scoped hook errors per-task: a failing hook on task
// A did not prevent task B's hook from firing in the same cycle.
//
// This test uses three independent tasks a, b, c.  Task `a`'s exit hook
// references a non-existent template and therefore fails template resolution.
// Task `b` has a valid exit hook pointing at the "echo" template.  With the
// fix (ProcessAllTaskHooks isolates per-task hook errors), b's exit hook node
// must be created despite a's failure.  With the bug, ProcessAllTaskHooks
// bailed on a's error before reaching b — b's exit hook node was never
// created.  See also TestBug_HookFailureDoesNotKillSiblings.
func TestBug_HookFailureKillsDAG(t *testing.T) {
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-kills-dag
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: echo
        hooks:
          exit:
            template: does-not-exist
      - name: b
        template: echo
        hooks:
          exit:
            template: echo
      - name: c
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: all three tasks scheduled (no deps between them).
	woc.operate(ctx)

	require.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("a"), "a should be scheduled on cycle 1")
	require.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("b"), "b should be scheduled on cycle 1")
	require.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("c"), "c should be scheduled on cycle 1")

	// Cycle 2: pods succeed.  a's broken hook fires first (declaration
	// order); with the bug, the error short-circuits ProcessAllTaskHooks
	// before b's valid hook is reached.  A fresh woc is required: operate()
	// must not be called twice on the same wfOperationCtx.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Under fix: b's exit hook node exists and is Pending or Succeeded.
	// Under bug:  b's exit hook node is never created.  Exit-hook nodes carry
	// the DAG boundary's ID, not the task's, so look the node up by its
	// GenerateOnExitNodeName-derived name.
	bHook, err := woc.wf.GetNodeByName("hook-kills-dag.b.onExit")
	require.NoError(t, err,
		"b's exit hook node must be created — a's hook failure must not abort b's hook processing")
	require.NotNil(t, bHook)
}

// TestBug_AssessNodeStatus_OutputsNotReady_NonContainerSet verifies that a
// regular Container template that declares outputs.parameters is NOT marked
// Succeeded while its WorkflowTaskResult is still pending — i.e. the node's
// Outputs has been partially populated by the controller (e.g. ExitCode from
// the pod status), but the executor-sourced Parameters have not yet arrived.
//
// Regression #14568: the new code in assessNodeStatus only fired the
// outputs-not-ready guard for ContainerSet templates, so Container/Script/
// Resource templates with declared outputs were flushed straight to
// Succeeded before taskResult sync, breaking downstream
// {{tasks.X.outputs.parameters.*}} resolution.
func TestBug_AssessNodeStatus_OutputsNotReady_NonContainerSet(t *testing.T) {
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: outputs-param-
  name: outputs-param-test
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            template: produce
    - name: produce
      container:
        image: alpine:3.23
        command: [sh, -c, "echo hello > /tmp/out.txt"]
      outputs:
        parameters:
          - name: msg
            valueFrom:
              path: /tmp/out.txt
`

	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	dagNodeID := wf.Name
	woc.wf.Status.Nodes = make(wfv1.Nodes)
	woc.wf.Status.Nodes[dagNodeID] = wfv1.NodeStatus{
		ID:            dagNodeID,
		Name:          wf.Name,
		TemplateName:  "main",
		Phase:         wfv1.NodeRunning,
		Type:          wfv1.NodeTypeDAG,
		TemplateScope: "local/main",
	}

	nodeName := wf.Name + ".a"
	nodeID := "node-a-id"
	// Simulate the realistic mid-sync state: the controller has already
	// captured the pod's ExitCode into node.Outputs, but the executor's
	// WorkflowTaskResult (which carries Parameters) has not yet been merged.
	exitCode := "0"
	woc.wf.Status.Nodes[nodeID] = wfv1.NodeStatus{
		ID:           nodeID,
		Name:         nodeName,
		TemplateName: "produce",
		Phase:        wfv1.NodeRunning,
		Type:         wfv1.NodeTypePod,
		BoundaryID:   dagNodeID,
		Outputs: &wfv1.Outputs{
			ExitCode: &exitCode,
		},
	}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"workflows.argoproj.io/workflow": wf.Name,
			},
			Namespace: "default",
		},
		Status: apiv1.PodStatus{
			Phase: apiv1.PodSucceeded,
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name: "main",
					State: apiv1.ContainerState{
						Terminated: &apiv1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
		},
	}

	// Outputs.Parameters not yet synced — assessNodeStatus must keep node Running.
	node := woc.wf.Status.Nodes[nodeID]
	updated := woc.assessNodeStatus(ctx, pod, &node)
	require.NotNil(t, updated)
	assert.Equal(t, wfv1.NodeRunning, updated.Phase,
		"Container template with declared output parameter must not flip to Succeeded before outputs.parameters is synced (regression #14568)")

	// Once outputs.parameters is populated, the node should be marked Succeeded.
	nodeWithOutputs := node.DeepCopy()
	nodeWithOutputs.Outputs.Parameters = []wfv1.Parameter{{Name: "msg"}}
	woc.wf.Status.Nodes[nodeID] = *nodeWithOutputs
	node = woc.wf.Status.Nodes[nodeID]

	updated = woc.assessNodeStatus(ctx, pod, &node)
	require.NotNil(t, updated)
	assert.Equal(t, wfv1.NodeSucceeded, updated.Phase,
		"Should be Succeeded once outputs.parameters are populated")
}

// TestBug_Retry_AllowsUnresolvedTags pins the contract of the retry-path
// SubstituteParams call (operator_template_execution.go ~line 259).
//
// Bug: that call site passed opts.onExitTemplate as the allowUnresolved
// flag. opts.onExitTemplate is a bool meaning "this call is for an onExit
// handler" — semantically unrelated to "allow unresolved tags". For normal
// (non-exit) retries it evaluates to false, making template.Replace strict
// and erroring on any late-resolved tag (e.g. {{pod.name}} in a non-pod
// retry-decorated template, {{tasks.X.outputs.*}} carried into the inner
// template body) inside the retry-decorated template.
//
// Origin/main hardcoded allowUnresolved=true at this call site. The fix
// on v4 hardcodes true too.
//
// This is a contract-pinning test: it verifies (a) SubstituteParams with
// allowUnresolved=true passes through late tags, mirroring the post-fix
// retry-path behavior, and (b) SubstituteParams with allowUnresolved=false
// errors on the same input — i.e. the bug's failure mode. The call site
// MUST pass true. If a future refactor reintroduces the boolean confusion,
// this test still documents the expected semantics.
func TestBug_Retry_AllowsUnresolvedTags(t *testing.T) {
	tmpl := &wfv1.Template{
		Name: "main",
		Container: &apiv1.Container{
			Image:   "alpine:3.23",
			Command: []string{"sh", "-c", "echo {{tasks.upstream.outputs.parameters.late}}"},
		},
	}

	// Mimic the retry-path localParams as constructed in
	// operator_template_execution.go around lines 221-249.
	localParams := common.Parameters{
		"retries":               "0",
		"retries.last.exitCode": "",
		"retries.last.status":   "",
		"retries.last.duration": "0",
		"retries.last.message":  "",
		"pod.name":              "retry-unresolved-main-1",
	}
	globalParams := common.Parameters{
		"workflow.name":      "retry-unresolved",
		"workflow.namespace": "argo",
	}

	ctx := logging.TestContext(t.Context())

	// allowUnresolved=true (origin/main, post-fix v4): must succeed.
	_, errAllow := common.SubstituteParams(ctx, tmpl, globalParams, localParams, true)
	require.NoError(t, errAllow,
		"SubstituteParams(allowUnresolved=true) must pass through unresolved late tags — this is the contract the retry path relies on")

	// allowUnresolved=false (the value the buggy v4 call site forwards when
	// opts.onExitTemplate=false): must error. This documents the failure
	// mode the retry path inadvertently triggered.
	_, errDeny := common.SubstituteParams(ctx, tmpl, globalParams, localParams, false)
	require.Error(t, errDeny,
		"SubstituteParams(allowUnresolved=false) must error on unresolved late tags — demonstrates the failure mode the retry path triggered when it forwarded opts.onExitTemplate (false) as allowUnresolved")
	assert.Contains(t, errDeny.Error(), "failed to resolve",
		"the strict-path error must report the unresolved tag")
}

// TestBug_MarkNodePhase_RefusesPostTerminalTransition verifies that markNodePhase
// does NOT flip a terminal node to a different phase (e.g. late TaskResult or
// duplicate hook delivery trying to demote Succeeded -> Failed).
//
// markNodePhase previously logged-and-allowed invalid SM transitions. Downstream
// consumers (exit handlers, metrics, taskset reconciliation) assume a node
// observed Succeeded stays Succeeded; the transition must be refused.
func TestBug_MarkNodePhase_RefusesPostTerminalTransition(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: t
  namespace: argo
spec:
  entrypoint: e
  templates:
  - name: e
    container:
      image: alpine
`)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Initialize a node directly as Succeeded.
	woc.initializeNode(ctx, "t.A", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeSucceeded, &wfv1.NodeFlag{}, true)

	// Attempt to flip the terminal node to Failed (e.g. late TaskResult or
	// duplicate hook delivery). markNodePhase must refuse.
	woc.markNodePhase(ctx, "t.A", wfv1.NodeFailed, "late TaskResult")

	node, err := woc.wf.GetNodeByName("t.A")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase,
		"terminal node phase must not flip; got %s", node.Phase)
}
