package controller

// Regression tests for bugs found while unifying DAG and Steps execution in
// the Engine. Each test asserts the correct behavior and locks the fix in.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
)

// TestBug_Depends_NegationCausesPrematureOmit: a task whose depends
// expression negates a sibling ("!slow-task.Failed") must not be omitted
// before the sibling has run. An Omitted node is permanent, so omitting it on
// the first cycle would skip it even after both deps succeed. The task waits
// until every task it references has finished, then runs.
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

	engine := NewEngine(woc, mainNode.Name, tmplCtx, tmpl, mainNode.ID, false)
	engine.reconciler = &fakeReconciler{}
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, []dag.Task{
		&dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[0]},
	}, tmpl, mainNode.ID, mainNode.Name)

	node, err := engine.executeTask(ctx, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[0]})

	require.Error(t, err,
		"engine must surface a reconciler materialization failure when Reconcile returns nil but no task node exists")
	assert.Nil(t, node)
	require.ErrorIs(t, err, ErrReconcilerNoMaterialize,
		"missing materialization must be reported with its own sentinel")
	assert.NotErrorIs(t, err, ErrParallelismReached,
		"missing materialization must not be reported as ordinary parallelism throttling")
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
// SubstituteParams call (operator_template_execution.go, handleRetries).
//
// Bug: that call site once passed opts.onExitTemplate as an allowUnresolved
// flag, making template.Replace strict for normal (non-exit) retries and
// erroring on any late-resolved tag (e.g. {{pod.name}} in a non-pod
// retry-decorated template, {{tasks.X.outputs.*}} carried into the inner
// template body) inside the retry-decorated template. SubstituteParams now
// always lets unresolved tags through, as on origin/main.
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

	_, err := common.SubstituteParams(ctx, tmpl, globalParams, localParams)
	require.NoError(t, err,
		"SubstituteParams must pass through unresolved late tags — this is the contract the retry path relies on")
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

// TestBug_MarkNodePhase_UnsyncedNodeMayOnlyError pins the rule for unsynced nodes: a
// terminal node whose task result has not arrived may still go to Error,
// but every other change out of its terminal phase is refused.
func TestBug_MarkNodePhase_UnsyncedNodeMayOnlyError(t *testing.T) {
	for _, tc := range []struct {
		from, to, want wfv1.NodePhase
	}{
		{wfv1.NodeFailed, wfv1.NodeError, wfv1.NodeError},
		{wfv1.NodeSucceeded, wfv1.NodeError, wfv1.NodeError},
		{wfv1.NodeFailed, wfv1.NodeSucceeded, wfv1.NodeFailed},
		{wfv1.NodeFailed, wfv1.NodePending, wfv1.NodeFailed},
		{wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeSucceeded},
		{wfv1.NodeError, wfv1.NodeRunning, wfv1.NodeError},
	} {
		t.Run(string(tc.from)+"-"+string(tc.to), func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(`{"metadata":{"name":"t","namespace":"argo"},"spec":{"entrypoint":"e","templates":[{"name":"e","container":{"image":"alpine"}}]}}`)
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			_, node := woc.initializeNode(ctx, "t.A", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", tc.from, &wfv1.NodeFlag{}, true)
			node.TaskResultSynced = new(false)
			woc.wf.Status.Nodes.Set(ctx, node.ID, *node)

			assert.Equal(t, tc.want, woc.markNodePhase(ctx, "t.A", tc.to).Phase)
		})
	}
}

var dagRetryConsumerAbsentOptional = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-retry-absent-optional
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: stage-a
            template: echo
          - name: stage-b
            template: produce
            depends: "stage-a.Failed"
          - name: stage-c
            template: consume
            depends: "stage-a && (stage-b || stage-b.Omitted)"
            arguments:
              parameters:
                - name: msg
                  value: "{{tasks.stage-b.outputs.parameters.output-message}}"
    - name: echo
      container:
        image: argoproj/argosay:v2
    - name: produce
      outputs:
        parameters:
          - name: output-message
            valueFrom:
              path: /tmp/output.txt
      container:
        image: argoproj/argosay:v2
    - name: consume
      retryStrategy:
        limit: "2"
      inputs:
        parameters:
          - name: msg
      container:
        image: argoproj/argosay:v2
`

// TestBug_RetryPath_ToleratesLateTags is the end-to-end counterpart of
// TestBug_Retry_AllowsUnresolvedTags: a retry-decorated task whose argument is
// an unhandled absent optional must reach the engine's terminal "absent
// optional" handling. If the retry path's SubstituteParams call forwarded a
// strict allowUnresolved, it would fail first with a generic "failed to
// resolve" error instead.
func TestBug_RetryPath_ToleratesLateTags(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(dagRetryConsumerAbsentOptional)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	for i := 0; i < 3 && !woc.wf.Status.Fulfilled(); i++ {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	nodeC := woc.wf.Status.Nodes.FindByDisplayName("stage-c")
	require.NotNil(t, nodeC, "stage-c must be materialized as a terminal node")
	assert.Equal(t, wfv1.NodeError, nodeC.Phase)
	assert.Contains(t, nodeC.Message, "absent optional")
	assert.NotContains(t, nodeC.Message, "failed to resolve")
}

// stuckTaskResultWorkflow returns a two-step workflow in the state reported in
// #17380: its first step completed normally, while the second step's pod reached
// Succeeded with its WorkflowTaskResult still a report-outputs-completed=false
// placeholder — the state of a pod whose aux container reported a non-retryable
// failure to the API server and gave up. No operation can reconcile that task
// result, so podReconciliation reports "pod reconciliation didn't complete, will
// retry" forever and the run only ends when something external removes the pod.
// specFields is spliced into the workflow spec, so callers can set (or omit)
// activeDeadlineSeconds, and extraTemplates into its template list.
func stuckTaskResultWorkflow(ctx context.Context, t *testing.T, specFields, extraTemplates string) *wfOperationCtx {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: stuck-task-result
  namespace: argo
spec:
` + specFields + `
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: step-1
        template: echo
    - - name: step-2
        template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
` + extraTemplates)

	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	step1, err := woc.wf.GetNodeByName("stuck-task-result[0].step-1")
	require.NoError(t, err)

	// step-1 finishes normally: its pod succeeds and its task result arrives, so
	// the node is fulfilled and the workflow moves on to step-2.
	setPodPhasesForTest(ctx, t, controller, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(n.Name, "step-1") {
			return apiv1.PodSucceeded
		}
		return ""
	})
	woc.wf.Status.MarkTaskResultComplete(ctx, step1.ID)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	require.Equal(t, wfv1.NodeSucceeded, nodePhase(t, woc.wf, step1.Name), "step-1 completes normally before step-2 gets stuck")

	step2, err := woc.wf.GetNodeByName("stuck-task-result[1].step-2")
	require.NoError(t, err)
	// The pod's aux container holds a placeholder task result labelled
	// report-outputs-completed=false from the moment the pod starts, and it is
	// what keeps the node's task result unsynced.
	injectPlaceholderTaskResult(ctx, t, woc, step2.ID)

	// The step runs, so its node is recorded Running.
	setPodPhasesForTest(ctx, t, controller, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(n.Name, "step-2") {
			return apiv1.PodRunning
		}
		return ""
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	require.Equal(t, wfv1.NodeRunning, nodePhase(t, woc.wf, step2.Name), "the step must be running before it completes")

	// The pod then completed successfully, so the only thing missing is its task result.
	setPodPhasesForTest(ctx, t, controller, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(n.Name, "step-2") {
			return apiv1.PodSucceeded
		}
		return ""
	})
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	return woc
}

// TestBug_WorkflowDeadlineWithUnreconcilableTaskResult pins #17380: a step whose
// pod reached Succeeded but whose WorkflowTaskResult never completed kept
// podReconciliation reporting "pod reconciliation didn't complete, will retry"
// on every operation, so neither the step's nor the workflow's
// activeDeadlineSeconds ever fired and the workflow stayed Running for as long
// as the pod lived (observed for over five hours, blocking every subsequent
// CronWorkflow run under concurrencyPolicy: Forbid).
//
// The step deadline cannot bound this case: template activeDeadlineSeconds is
// enforced by the pod and by the executor's deadline monitor, neither of which
// can act on a pod whose containers have already exited. The workflow deadline
// is the only bound, so it must fire whether or not the task result reconciled.
func TestBug_WorkflowDeadlineWithUnreconcilableTaskResult(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := stuckTaskResultWorkflow(ctx, t, "  activeDeadlineSeconds: 300", "")
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "a task result that is merely slow must be retried, not failed")

	woc = rewindDeadline(ctx, t, woc)

	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "workflow must fail once its deadline has passed")
	assert.Equal(t, "Step exceeded its deadline", woc.wf.Status.Message, "the workflow takes the entry node's message, as it does for any other deadline failure")
	assert.Equal(t, "true", woc.wf.Labels[common.LabelKeyCompleted], "a failed workflow must be marked completed so it is archived and GC'd")

	step1, err := woc.wf.GetNodeByName("stuck-task-result[0].step-1")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, step1.Phase, "the step that completed normally is untouched")

	step2, err := woc.wf.GetNodeByName("stuck-task-result[1].step-2")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, step2.Phase)
	assert.Equal(t, "Step exceeded its deadline", step2.Message)
	require.NotNil(t, step2.TaskResultSynced)
	assert.True(t, *step2.TaskResultSynced,
		"the unreconcilable task result must be force-completed, or the node is never fulfilled and the workflow cannot complete")
}

// TestBug_DeadlineStopsNestedDaemonWithUnreconcilableTaskResult covers the
// daemon half of the same deadline path: a daemoned node counts as fulfilled
// while it runs, so failing a workflow at its deadline leaves it daemoned unless
// it is stopped, and markWorkflowPhase then refuses to complete a workflow that
// still has daemon nodes. Its root-level fallback cannot reach a daemon nested
// under a Steps or DAG boundary, so the daemon has to be stopped from that
// boundary.
func TestBug_DeadlineStopsNestedDaemonWithUnreconcilableTaskResult(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: stuck-with-daemon
  namespace: argo
spec:
  activeDeadlineSeconds: 300
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: stuck-step
        template: echo
      - name: daemon-step
        template: daemon
  - name: echo
    container:
      image: argoproj/argosay:v2
  - name: daemon
    daemon: true
    container:
      image: argoproj/argosay:v2
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Both steps run in the same group, so neither waits for the other: the
	// daemon becomes ready and is recorded daemoned, while the other step's pod
	// completes without ever reporting its task result.
	stuckStep, err := woc.wf.GetNodeByName("stuck-with-daemon[0].stuck-step")
	require.NoError(t, err)
	injectPlaceholderTaskResult(ctx, t, woc, stuckStep.ID)
	daemonStep, err := woc.wf.GetNodeByName("stuck-with-daemon[0].daemon-step")
	require.NoError(t, err)
	injectPlaceholderTaskResult(ctx, t, woc, daemonStep.ID)
	setPodPhasesForTest(ctx, t, controller, woc, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodRunning }, withReadyContainers())
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	require.Equal(t, wfv1.NodeRunning, nodePhase(t, woc.wf, daemonStep.Name))
	daemonNode, err := woc.wf.GetNodeByName(daemonStep.Name)
	require.NoError(t, err)
	require.True(t, daemonNode.IsDaemoned(), "the daemon step must be recorded daemoned before the deadline")

	// Only the stuck step's pod completes; the daemon keeps running.
	setPodPhasesForTest(ctx, t, controller, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(n.Name, "stuck-step") {
			return apiv1.PodSucceeded
		}
		return ""
	})
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "workflow must not fail before its deadline")
	}

	woc = rewindDeadline(ctx, t, woc)

	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase,
		"a nested daemon must not keep the workflow Running past its deadline")
	daemonNode, err = woc.wf.GetNodeByName(daemonStep.Name)
	require.NoError(t, err)
	assert.Nil(t, daemonNode.Daemoned, "the daemon must be stopped")
	assert.Equal(t, wfv1.NodeSucceeded, daemonNode.Phase, "a daemon the controller stops completes Succeeded, as killDaemonedChildren records it")
}

// exitHandlerTemplate is spliced into stuckTaskResultWorkflow's template list by
// the exit-handler test.
const exitHandlerTemplate = `  - name: exit-handler
    container:
      image: argoproj/argosay:v2
`

// TestBug_DeadlineRunsExitHandlerWithUnreconcilableTaskResult covers the
// completion half of the same deadline path: the workflow must go out through
// the normal tail, so that spec.onExit still runs and the PVCs are still
// deleted. Completing the workflow inside the deadline branch would skip both,
// because every later operation returns at the completed-workflow guard.
func TestBug_DeadlineRunsExitHandlerWithUnreconcilableTaskResult(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := stuckTaskResultWorkflow(ctx, t, `  activeDeadlineSeconds: 300
  onExit: exit-handler`, exitHandlerTemplate)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// The first operation past the deadline fails the step and starts the exit
	// handler; the workflow itself is not completed yet.
	woc = rewindDeadline(ctx, t, woc)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "the workflow stays Running until its exit handler has run")
	onExit := woc.wf.Status.Nodes.FindByDisplayName("stuck-task-result.onExit")
	require.NotNil(t, onExit, "the exit handler must have been started by the deadline path")

	// The exit handler's pod completes normally.
	setPodPhasesForTest(ctx, t, woc.controller, woc, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
	woc.wf.Status.MarkTaskResultComplete(ctx, onExit.ID)
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
		woc.operate(ctx)
	}

	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "workflow must still fail at its deadline")
	assert.Equal(t, "true", woc.wf.Labels[common.LabelKeyCompleted])
	onExit = woc.wf.Status.Nodes.FindByDisplayName("stuck-task-result.onExit")
	require.NotNil(t, onExit)
	assert.Equal(t, wfv1.NodeSucceeded, onExit.Phase, "spec.onExit must run even though the step's task result never completed")
}

// TestBug_StuckTaskResultWithoutDeadlineKeepsRetrying bounds the fix above: with
// no workflow deadline there is nothing to enforce, so an unreconcilable task
// result must keep being retried rather than failing the workflow.
func TestBug_StuckTaskResultWithoutDeadlineKeepsRetrying(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := stuckTaskResultWorkflow(ctx, t, "", "")

	woc.wf.Status.StartedAt = metav1.Time{Time: woc.wf.Status.StartedAt.Add(-time.Hour)}
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
		woc.operate(ctx)
	}

	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "without a deadline there is nothing to enforce, so the workflow is still retried")
	step2, err := woc.wf.GetNodeByName("stuck-task-result[1].step-2")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, step2.Phase, "the step that completed is still waiting for its task result")
}

// TestBug_DeadlineWithUnreconcilableTaskResultWhileSuspended covers the same
// deadline path for a workflow the user has suspended after its step got stuck:
// the suspension check in operate() must not defer a workflow whose nodes were
// just failed at its deadline, or the deadline fails the nodes and then requeues
// forever instead of ending the run.
func TestBug_DeadlineWithUnreconcilableTaskResultWhileSuspended(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := stuckTaskResultWorkflow(ctx, t, "  activeDeadlineSeconds: 300", "")
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// The user suspends the workflow, as `argo suspend` does.
	suspend := true
	woc.wf.Spec.Suspend = &suspend
	for range 2 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
		woc.operate(ctx)
		require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "a suspended workflow waits to be resumed while its deadline has not passed")
	}

	woc = rewindDeadline(ctx, t, woc)

	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "an expired workflow must be failed even while it is suspended")
	assert.Equal(t, "true", woc.wf.Labels[common.LabelKeyCompleted])
}

// TestBug_SuspendedDeadlineFinalizesAcrossReconciliations covers the same
// deadline path for a suspended workflow with spec.onExit: the exit handler is
// started by the first operation past the deadline and completes several
// operations later, so the suspension bypass has to hold across
// reconciliations rather than only in the operation that failed the nodes.
func TestBug_SuspendedDeadlineFinalizesAcrossReconciliations(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := stuckTaskResultWorkflow(ctx, t, `  activeDeadlineSeconds: 300
  onExit: exit-handler`, exitHandlerTemplate)

	suspend := true
	woc.wf.Spec.Suspend = &suspend

	// The first operation past the deadline fails the step and starts the exit
	// handler, which is then running while the workflow is still suspended.
	woc = rewindDeadline(ctx, t, woc)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "the workflow is not completed until its exit handler has run")
	require.Equal(t, "true", woc.wf.Annotations[common.AnnotationKeyDeadlineNodesFailed],
		"failNodesAfterDeadline must record the marker that the operations below rely on to bypass suspension")
	onExit := woc.wf.Status.Nodes.FindByDisplayName("stuck-task-result.onExit")
	require.NotNil(t, onExit, "the exit handler must have been started")

	// Operations in which neither the exit handler nor the stuck task result
	// changed must not lose the bypass: the workflow is still suspended, so
	// without it the exit handler would never be allowed to finish.
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
		woc.operate(ctx)
		require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	}

	// The exit handler's pod then completes normally.
	setPodPhasesForTest(ctx, t, woc.controller, woc, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
	woc.wf.Status.MarkTaskResultComplete(ctx, onExit.ID)
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
		woc.operate(ctx)
	}

	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "the workflow must complete once its exit handler has finished")
	assert.Equal(t, "true", woc.wf.Labels[common.LabelKeyCompleted])
	onExit = woc.wf.Status.Nodes.FindByDisplayName("stuck-task-result.onExit")
	require.NotNil(t, onExit)
	assert.Equal(t, wfv1.NodeSucceeded, onExit.Phase)
}

// TestBug_SuspendedWorkflowPastDeadlineStillWaits bounds the fix above: a
// suspended workflow whose pods reconciled normally must still wait to be
// resumed, even once its deadline has passed. Only a workflow whose nodes were
// failed at its deadline is allowed past the suspension check.
func TestBug_SuspendedWorkflowPastDeadlineStillWaits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: susp-waits
  namespace: argo
spec:
  activeDeadlineSeconds: 300
  entrypoint: main
  templates:
  - name: main
    container:
      image: argoproj/argosay:v2
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	// The step completes normally and its task result arrives, so pod
	// reconciliation completes and the deadline branch never fires.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	suspend := true
	woc.wf.Spec.Suspend = &suspend
	// While the deadline has not passed, suspension is honoured as usual.
	woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
	woc.operate(ctx)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "a suspended workflow waits to be resumed while its deadline has not passed")

	woc = rewindDeadline(ctx, t, woc)

	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "a suspended workflow whose pods reconciled is still waiting to be resumed")
	assert.NotEqual(t, "true", woc.wf.Labels[common.LabelKeyCompleted], "suspension must still stop the workflow from completing")
}

// TestBug_ExpiredSuspendedExitHandlerWithoutDeadlineFailureStillWaits bounds the
// suspension bypass from the other side: an unfinished exit handler after the
// deadline is not enough on its own. A workflow that reached its exit handler,
// was then suspended, and only later expired has never had its nodes failed by
// failNodesAfterDeadline, so the marker is absent and it must still wait to be
// resumed rather than finalizing.
//
// The exit handler has two steps so the suspension shows up as observable
// progress: step-a has already finished, so without the marker gate the
// bypassed operation would run the template engine past the suspension check
// and start step-b while the workflow is suspended. With the gate it never
// starts until the workflow is resumed.
func TestBug_ExpiredSuspendedExitHandlerWithoutDeadlineFailureStillWaits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: susp-exit-waits
  namespace: argo
spec:
  activeDeadlineSeconds: 300
  entrypoint: main
  onExit: exit-handler
  templates:
  - name: main
    container:
      image: argoproj/argosay:v2
  - name: exit-handler
    steps:
    - - name: step-a
        template: echo
    - - name: step-b
        template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	// The step completes normally and its task result arrives, so pod
	// reconciliation completes and the deadline branch never fires.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// The exit handler starts while the workflow is not yet suspended, and its
	// first step finishes too.
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	onExit := woc.wf.Status.Nodes.FindByDisplayName("susp-exit-waits.onExit")
	require.NotNil(t, onExit, "the exit handler must have started before the workflow was suspended")
	_, err := woc.wf.GetNodeByName("susp-exit-waits.onExit[0].step-a")
	require.NoError(t, err, "the exit handler's first step must have started")
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// The workflow is then suspended and only afterwards expires. Its pods all
	// reconciled, so failNodesAfterDeadline never ran and the marker is absent.
	suspend := true
	woc.wf.Spec.Suspend = &suspend
	woc = rewindDeadline(ctx, t, woc)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase,
		"suspension still applies: this workflow's nodes were never failed at its deadline")
	assert.NotEqual(t, "true", woc.wf.Annotations[common.AnnotationKeyDeadlineNodesFailed],
		"no deadline failure occurred, so the marker that gates the bypass must be absent")
	stepA, err := woc.wf.GetNodeByName("susp-exit-waits.onExit[0].step-a")
	require.NoError(t, err)
	assert.True(t, stepA.Fulfilled(), "the finished first step leaves only step-b to run, which is what the bypass would start")

	// Without the marker gate this operation would have run past the
	// suspension check and started the exit handler's second step.
	_, err = woc.wf.GetNodeByName("susp-exit-waits.onExit[1].step-b")
	require.Error(t, err, "a suspended workflow whose nodes were not failed at its deadline must not advance its exit handler")

	// It keeps waiting across further reconciliations.
	for range 2 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase,
		"a suspended workflow whose nodes were not failed at its deadline still waits to be resumed")
	assert.NotEqual(t, "true", woc.wf.Labels[common.LabelKeyCompleted], "suspension must still stop the workflow from completing")
	_, err = woc.wf.GetNodeByName("susp-exit-waits.onExit[1].step-b")
	require.Error(t, err, "the exit handler's second step must not start before the workflow is resumed")
}

// setPodPhasesForTest moves each workflow pod to the phase decide returns for its
// node, like a pod controller would, and waits for the pod informer to catch up
// so the next operate sees it. An empty phase leaves the pod alone. It never
// touches task results: unlike setPodPhases it lets a test leave a pod that
// succeeded with its task result permanently incomplete (#17380).
func setPodPhasesForTest(ctx context.Context, t *testing.T, controller *WorkflowController, woc *wfOperationCtx, decide func(node *wfv1.NodeStatus) apiv1.PodPhase, opts ...func(pod *apiv1.Pod)) {
	t.Helper()
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.NotEmpty(t, pods.Items)
	for i := range pods.Items {
		pod := pods.Items[i]
		node, ok := woc.wf.Status.Nodes[woc.nodeID(&pod)]
		if !ok {
			continue
		}
		phase := decide(&node)
		if phase == "" || pod.Status.Phase == phase {
			continue
		}
		pod.Status.Phase = phase
		for _, opt := range opts {
			opt(&pod)
		}
		updated, err := controller.kubeclientset.CoreV1().Pods(pod.Namespace).Update(ctx, &pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		waitForInformer(ctx, controller.PodController.TestingPodInformer(), updated, func(obj any) bool {
			return obj.(*apiv1.Pod).Status.Phase == phase
		})
	}
}

// withReadyContainers synthesizes what a kubelet reports for a running pod, so
// that assessNodeStatus records a daemon template's node as daemoned: a node
// becomes daemoned only once every container of its pod is ready. Only the main
// containers are reported — a running aux (wait/supervisor) container keeps a
// completed pod's node out of a terminal phase, which is a different scenario
// from the one these tests build.
func withReadyContainers() func(pod *apiv1.Pod) {
	return func(pod *apiv1.Pod) {
		pod.Status.ContainerStatuses = nil
		for _, ctr := range pod.Spec.Containers {
			switch ctr.Name {
			case common.WaitContainerName, common.SupervisorContainerName, common.InitContainerName:
				continue
			}
			pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, apiv1.ContainerStatus{
				Name:  ctr.Name,
				Ready: true,
				State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
			})
		}
	}
}

// rewindDeadline moves the workflow's start far enough into the past that its
// activeDeadlineSeconds has elapsed, and re-operates.
func rewindDeadline(ctx context.Context, t *testing.T, woc *wfOperationCtx) *wfOperationCtx {
	t.Helper()
	woc.wf.Status.StartedAt = metav1.Time{Time: woc.wf.Status.StartedAt.Add(-time.Hour)}
	fresh := newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
	fresh.operate(ctx)
	return fresh
}

// nodePhase returns the recorded phase of the named node.
func nodePhase(t *testing.T, wf *wfv1.Workflow, name string) wfv1.NodePhase {
	t.Helper()
	node, err := wf.GetNodeByName(name)
	require.NoError(t, err)
	return node.Phase
}
