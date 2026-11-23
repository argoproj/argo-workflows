package controller

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
)

// TestStepsFailedRetries ensures a steps template will recognize exhausted retries
func TestStepsFailedRetries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps-failed-retries.yaml")
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestArtifactResolutionWhenSkipped(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/artifact-resolution-when-skipped.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// The engine cascades instant completions (skipped groups) within a
	// single operate cycle, as the pre-Engine controller did.
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestStepsWithParamAndGlobalParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-with-param-and-global-param.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestExpandStepGroupWithParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-with-param.yaml")
	woc := newWoc(ctx, *wf)
	tmpl := wf.Spec.Templates[0]
	step := tmpl.Steps[0].Steps[0]
	dagTask := &wfv1.DAGTask{
		Name:      step.Name,
		Template:  step.Template,
		Arguments: step.Arguments,
		WithParam: step.WithParam,
	}

	expanded, err := dag.ExpandTask(ctx, *dagTask, make(map[string]string), woc)
	require.NoError(t, err)
	require.Len(t, expanded, 4)

	expectedExpandedTasks := []struct {
		Name      string
		Parameter string
	}{
		{
			Name:      "use-with-param(0:1234)",
			Parameter: "1234",
		},
		{
			Name:      `use-with-param(1:foo\tbar)`,
			Parameter: "foo\tbar",
		},
		{
			Name:      "use-with-param(2:true)",
			Parameter: "true",
		},
		{
			Name:      "use-with-param(3:[])",
			Parameter: "[]",
		},
	}

	for i, expected := range expectedExpandedTasks {
		assert.Equal(t, expected.Name, expanded[i].Name)
		require.Len(t, expanded[i].Arguments.Parameters, 1)
		assert.Equal(t, expected.Parameter, expanded[i].Arguments.Parameters[0].Value.String())
	}
}

func TestExpandStepGroupWithItems(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-with-items.yaml")
	woc := newWoc(ctx, *wf)
	tmpl := wf.Spec.Templates[0]
	step := tmpl.Steps[0].Steps[0]
	dagTask := &wfv1.DAGTask{
		Name:      step.Name,
		Template:  step.Template,
		Arguments: step.Arguments,
		WithItems: step.WithItems,
	}

	expanded, err := dag.ExpandTask(ctx, *dagTask, make(map[string]string), woc)
	require.NoError(t, err)
	require.Len(t, expanded, 1)

	assert.Equal(t, "Hello\"Argo", expanded[0].Arguments.Parameters[0].Value.String())
}

func TestResourceDurationMetric(t *testing.T) {
	woc := wfOperationCtx{wf: &wfv1.Workflow{}, scope: variables.NewScope()}
	var node wfv1.NodeStatus
	wfv1.MustUnmarshal("@testdata/steps/resource-duration-metric-node-status.yaml", &node)
	localScope, _ := woc.prepareMetricScope(&node)
	assert.Equal(t, "33", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "24", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["exitCode"])
}

func TestResourceDurationMetricDefaultMetricScope(t *testing.T) {
	wf := wfv1.Workflow{Status: wfv1.WorkflowStatus{StartedAt: metav1.NewTime(time.Now())}}
	woc := wfOperationCtx{
		scope: variables.NewScope(),
		wf:    &wf,
	}

	localScope, realTimeScope := woc.prepareDefaultMetricScope()

	assert.Equal(t, "0", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "0", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["duration"])
	assert.Equal(t, "Pending", localScope["status"])
	assert.Less(t, realTimeScope["workflow.duration"](), 1.0)
}

// Regression: when realtime metrics are evaluated before Status.StartedAt has
// been populated (the first operate cycle of a brand-new workflow), the
// workflow.duration closure must return 0 rather than time.Since(zero-time)
// saturating to MaxInt64 nanoseconds (~9.22e9 seconds).
func TestRealTimeWorkflowDurationBeforeStartedAt(t *testing.T) {
	wf := wfv1.Workflow{}
	woc := wfOperationCtx{
		scope: variables.NewScope(),
		wf:    &wf,
	}

	_, realTimeScope := woc.prepareDefaultMetricScope()

	assert.InDelta(t, 0.0, realTimeScope["workflow.duration"](), 0)
}

func TestOptionalArgumentAndParameter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/optional-argument-and-parameter.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestOptionalArgumentUseSubPathInLoop(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/artifact-resolution-when-optional-and-subpath.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

// Regression test: referencing {{steps.<expanded_step>.id}} where the step was expanded
// via withItems. Before the fix, buildLocalScope was not called for the StepGroup node
// when a step had withItem/withParam expansion, so steps.<step>.id was unavailable
// and caused a requeue.
func TestStepsStepGroupIDReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-step-group-id-ref.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Verify the use-id step was created (not stuck in requeue due to missing variable)
	useIDNode := woc.wf.Status.Nodes.FindByDisplayName("use-id")
	require.NotNil(t, useIDNode, "use-id node should be created when steps.fanout.id is resolvable")

	// For an expanded step, steps.<step>.id resolves to the enclosing StepGroup
	// node (as it always has: Steps had no TaskGroup node before the Engine),
	// not to the TaskGroup node the Engine now creates for the expansion.
	require.NotNil(t, useIDNode.Inputs)
	require.Len(t, useIDNode.Inputs.Parameters, 1)
	assert.Equal(t, woc.wf.NodeID("steps-stepgroup-id-ref[0]"), useIDNode.Inputs.Parameters[0].Value.String())
}

// TestStepsWhenSkipNoRequeue verifies that a step with a "when" clause that evaluates to false
// does not cause a requeue even when other fields in the step reference outputs that don't exist.
// Scenario: A is skipped (when: "false"), so A's outputs don't exist. B has
// when: "{{steps.A.status}} == Succeeded" which evaluates to false ("Skipped == Succeeded").
// B also references {{steps.A.outputs.result}} which is unresolvable since A was skipped.
// Without the fix, the full ReplaceStrict would fail on the missing output and requeue.
// With the fix, the when clause is resolved first, evaluates to false, and B is skipped early.
func TestStepsWhenSkipNoRequeue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-when-skip-no-requeue.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Workflow should succeed: A was skipped, B's when evaluated false so B was also skipped
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, nodeB)
	assert.Equal(t, wfv1.NodeSkipped, nodeB.Phase)
}

// TestStepsWhenExprWithParamFilter verifies that expression templates work correctly
// in a steps workflow with withParam expansion and a when clause that filters items
// using expression functions (concat, get, ??, toJSON, fromJSON, join).
// This mirrors a real-world pattern where a dynamic list parameter controls which
// withParam items execute.
func TestStepsWhenExprWithParamFilter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-when-expr-with-param-filter.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Workflow is Running because pods haven't completed, but we can verify:
	// 1. No error occurred during expression evaluation
	// 2. All 4 items were expanded and scheduled (none were incorrectly skipped/errored)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// "first" has type "" which matches empty string in the filter list
	node0 := woc.wf.Status.Nodes.FindByDisplayName("fst(0:name:first,type:)")
	require.NotNil(t, node0)
	assert.Equal(t, wfv1.NodePending, node0.Phase)

	// "second" has type "always" which is in the filter list
	node1 := woc.wf.Status.Nodes.FindByDisplayName("fst(1:name:second,type:always)")
	require.NotNil(t, node1)
	assert.Equal(t, wfv1.NodePending, node1.Phase)

	// "third" has type "test" which is in the filter list (test=true)
	node2 := woc.wf.Status.Nodes.FindByDisplayName("fst(2:name:third,type:test)")
	require.NotNil(t, node2)
	assert.Equal(t, wfv1.NodePending, node2.Phase)

	// "fourth" has no type, defaults to "always" via ?? operator
	node3 := woc.wf.Status.Nodes.FindByDisplayName("fst(3:name:fourth)")
	require.NotNil(t, node3)
	assert.Equal(t, wfv1.NodePending, node3.Phase)
}

// TestThreeStepCrossGroupArtifactResolution tests that step [2] can reference
// artifacts from step [0] (not a direct dependency) via the preceding groups scope fix.
// This matches the artifact-passing-subpath.yaml example workflow.
func TestThreeStepCrossGroupArtifactResolution(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Build a steps template with 3 groups where [2] references [0]'s artifact
	tmpl := &wfv1.Template{
		Steps: []wfv1.ParallelSteps{
			{Steps: []wfv1.WorkflowStep{{Name: "generate-artifact", Template: "gen"}}},
			{Steps: []wfv1.WorkflowStep{{Name: "list-artifact", Template: "list", Arguments: wfv1.Arguments{
				Artifacts: []wfv1.Artifact{{Name: "message", From: "{{steps.generate-artifact.outputs.artifacts.hello-art}}"}},
			}}}},
			{Steps: []wfv1.WorkflowStep{{Name: "consume-artifact", Template: "consume", Arguments: wfv1.Arguments{
				Artifacts: []wfv1.Artifact{{Name: "message", From: "{{steps.generate-artifact.outputs.artifacts.hello-art}}", SubPath: "hello_world.txt"}},
			}}}},
		},
	}

	wfName := "test-three-step-artifact"
	wf := &wfv1.Workflow{}
	wf.Name = wfName
	wf.Status.Nodes = make(wfv1.Nodes)

	// Create the [0].generate-artifact node (completed with artifact outputs)
	genNodeName := wfName + "[0].generate-artifact"
	genNodeID := wf.NodeID(genNodeName)
	wf.Status.Nodes[genNodeID] = wfv1.NodeStatus{
		ID:    genNodeID,
		Name:  genNodeName,
		Phase: wfv1.NodeSucceeded,
		Type:  wfv1.NodeTypePod,
		Outputs: &wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "hello-art",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{Bucket: "my-bucket"},
							Key:      "outputs/hello-art",
						},
					},
				},
			},
		},
	}

	// Create the [1].list-artifact node (completed, no outputs)
	listNodeName := wfName + "[1].list-artifact"
	listNodeID := wf.NodeID(listNodeName)
	wf.Status.Nodes[listNodeID] = wfv1.NodeStatus{
		ID:    listNodeID,
		Name:  listNodeName,
		Phase: wfv1.NodeSucceeded,
		Type:  wfv1.NodeTypePod,
	}

	// Build the step adapters (same way executeSteps does)
	var tasks []dag.Task
	var prevStepNames []string
	for i, stepGroup := range tmpl.Steps {
		var currentStepNames []string
		for _, step := range stepGroup.Steps {
			s := step // capture
			task := &StepAdapter{
				step:         &s,
				dependencies: prevStepNames,
				groupIndex:   i,
			}
			tasks = append(tasks, task)
			currentStepNames = append(currentStepNames, task.GetName())
		}
		prevStepNames = currentStepNames
	}

	// The third task is [2].consume-artifact
	consumeTask := tasks[2]
	require.Equal(t, "[2].consume-artifact", consumeTask.GetName())
	require.Equal(t, []string{"[1].list-artifact"}, consumeTask.GetDependencies())

	// Create a minimal woc and Engine
	woc := &wfOperationCtx{
		wf:    wf,
		scope: variables.NewScope(),
		log:   logging.RequireLoggerFromContext(ctx),
	}
	engine := &Engine{
		woc:      woc,
		nodeName: wfName,
		tmpl:     tmpl,
		log:      woc.log,
	}
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(wf, tasks, tmpl, "", wfName)

	// Build scope for [2].consume-artifact
	scope, err := engine.buildLocalScopeFromTask(ctx, consumeTask)
	require.NoError(t, err, "buildLocalScopeFromTask should not error")

	// Verify the scope contains the artifact from step [0]
	artKey := "steps.generate-artifact.outputs.artifacts.hello-art"
	_, val, resolveErr := scope.resolveVar("{{" + artKey + "}}")
	require.NoError(t, resolveErr, "scope should resolve %s", artKey)
	require.NotNil(t, val, "resolved artifact should not be nil")

	// Verify it's an actual artifact
	art, ok := val.(wfv1.Artifact)
	require.True(t, ok, "resolved value should be a wfv1.Artifact, got %T", val)
	assert.Equal(t, "hello-art", art.Name)
	assert.Equal(t, "outputs/hello-art", art.S3.Key)
}

// TestStepsInstantCompletionCascades proves that the engine cascades instant
// completions (when-skipped groups) within a single operate() call.
// If the old N-cycle regression claim were true, step-c would NOT exist after
// one operate() because groups [0] and [1] would each consume a separate cycle.
func TestStepsInstantCompletionCascades(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-instant-completion-cascade.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Single operate() call — should cascade through all skipped groups.
	woc.operate(ctx)

	// Groups [0] and [1] should be skipped.
	nodeA := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, nodeA, "step-a node should exist")
	assert.Equal(t, wfv1.NodeSkipped, nodeA.Phase)

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	require.NotNil(t, nodeB, "step-b node should exist")
	assert.Equal(t, wfv1.NodeSkipped, nodeB.Phase)

	// The key assertion: step-c must exist after a single operate().
	// This proves the engine cascaded through the skipped groups in one call.
	nodeC := woc.wf.Status.Nodes.FindByDisplayName("step-c")
	require.NotNil(t, nodeC, "step-c should be reached in a single operate() call, proving cascading works")
}

// TestStepsGroupsChainedNotSiblings verifies that StepGroup nodes are chained
// sequentially in the node graph rather than attached as siblings to the Steps
// root. Legacy behavior (and the expected behavior) is:
//
//	root          → [0]
//	[0]           → step-a
//	step-a (pod)  → [1]        (outbound linkage)
//	[1]           → step-b
//
// A regression during the DAG refactor attached every StepGroup directly under
// the Steps root, producing a graph where [0] and [1] were siblings — breaking
// the sequential chain required for UI rendering and outbound-node traversal.
//
// [1] must not be linked while [0] is in flight: injecting it into the
// descendant chain of a running pod would poison childrenFulfilled() and break
// retry finalization / sync-lock release for the prior group. Each StepGroup
// is created when it starts, as executeSteps did, so [1] only appears in the
// graph once [0] has fulfilled.
func TestStepsGroupsChainedNotSiblings(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps/steps-sequential-groups.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: step-a starts but is not fulfilled. [1] has not started, so it
	// does not exist yet — wiring it into the graph now would break
	// retry/sync semantics for [0].
	woc.operate(ctx)

	rootID := woc.wf.NodeID(wf.Name)
	sg0ID := woc.wf.NodeID(wf.Name + "[0]")
	sg1ID := woc.wf.NodeID(wf.Name + "[1]")

	root, err := woc.wf.Status.Nodes.Get(rootID)
	require.NoError(t, err, "root node should exist")
	assert.Contains(t, root.Children, sg0ID, "[0] must be a child of the Steps root")
	assert.NotContains(t, root.Children, sg1ID,
		"[1] must NOT be a sibling of [0] under the Steps root (regression check)")

	stepA := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepA, "step-a pod node should exist")
	assert.NotContains(t, stepA.Children, sg1ID,
		"[1] must NOT be wired under step-a until step-a fulfills (timing invariant)")
	_, err = woc.wf.Status.Nodes.Get(sg1ID)
	require.Error(t, err, "[1] must not exist before it starts")

	// Cycle 2: step-a completes. [1] starts, linked under step-a; step-b
	// gets scheduled under [1]; the full chain root → [0] → step-a → [1] →
	// step-b must be coherent.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	root, err = woc.wf.Status.Nodes.Get(rootID)
	require.NoError(t, err)
	assert.Equal(t, []string{sg0ID}, root.Children,
		"the Steps root must have exactly [0] as its only child")

	sg0, err := woc.wf.Status.Nodes.Get(sg0ID)
	require.NoError(t, err)
	stepA = woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepA)
	assert.Contains(t, sg0.Children, stepA.ID, "[0] must contain step-a as a child")
	assert.Contains(t, stepA.Children, sg1ID,
		"[1] must be wired as a child of step-a (the outbound of [0]) once [0] fulfills")

	// [1] is linked once: it appears exactly once under step-a.
	count := 0
	for _, c := range stepA.Children {
		if c == sg1ID {
			count++
		}
	}
	assert.Equal(t, 1, count, "[1] should appear exactly once in step-a.Children")

	stepB := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	require.NotNil(t, stepB, "step-b should be scheduled after step-a succeeds")
	sg1, err := woc.wf.Status.Nodes.Get(sg1ID)
	require.NoError(t, err)
	assert.True(t, slices.Contains(sg1.Children, stepB.ID),
		"[1] must contain step-b as a child")
}

const stepsPendingTimeoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-pending-timeout
  namespace: default
spec:
  entrypoint: main
  onExit: exit
  templates:
  - name: main
    steps:
    - - name: slow
        template: run
      - name: stuck
        template: pending
        continueOn:
          failed: {{continue}}
    - - name: after
        template: run
  - name: run
    container:
      image: alpine:3.23
  - name: pending
    pendingTimeout: 30s
    container:
      image: alpine:3.23
  - name: exit
    container:
      image: alpine:3.23
`

// TestStepsPendingTimeoutWaitsForSiblings verifies that a step whose
// pendingTimeout passes fails only itself, as a DAG task does: its step group
// waits for its running siblings before it ends, the exit handler starts only
// then, and continueOn.failed on the step lets the next group run.
func TestStepsPendingTimeoutWaitsForSiblings(t *testing.T) {
	for _, continueOn := range []bool{false, true} {
		t.Run("continueOn.failed="+strconv.FormatBool(continueOn), func(t *testing.T) {
			manifest := strings.Replace(stepsPendingTimeoutWf, "{{continue}}", strconv.FormatBool(continueOn), 1)
			ctx, r := r4Start(t, manifest)
			// slow runs; stuck's pod stays Pending past its pendingTimeout.
			r.r4SetPods(ctx, map[string]apiv1.PodPhase{"stuck": apiv1.PodPending})
			require.Equal(t, wfv1.NodeRunning, r.r4Phase("slow"))
			r.r4AgeNode(ctx, "stuck", time.Hour)
			r.op(ctx)

			stuck := r.woc.wf.Status.Nodes.FindByDisplayName("stuck")
			require.NotNil(t, stuck)
			assert.Equal(t, wfv1.NodeFailed, stuck.Phase)
			assert.Equal(t, wfv1.NodeRunning, r.r4Phase("slow"))
			assert.Equal(t, wfv1.NodeRunning, r.r4Phase("[0]"), "the group waits for slow")
			assert.Empty(t, r.r4Phase("steps-pending-timeout.onExit"), "the exit handler waits for the group")

			r.r4SetPods(ctx, map[string]apiv1.PodPhase{"slow": apiv1.PodSucceeded})
			if continueOn {
				assert.Equal(t, wfv1.NodeSucceeded, r.r4Phase("[0]"))
				assert.NotEmpty(t, r.r4Phase("after"), "continueOn.failed lets the next group run")
				r.drive(ctx, 10, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
				assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase)
				return
			}
			group := r.woc.wf.Status.Nodes.FindByDisplayName("[0]")
			require.NotNil(t, group)
			assert.Equal(t, wfv1.NodeFailed, group.Phase)
			assert.Equal(t, "child '"+stuck.ID+"' failed", group.Message)
			assert.Equal(t, wfv1.NodeOmitted, r.r4Phase("after"))
			assert.NotEmpty(t, r.r4Phase("steps-pending-timeout.onExit"), "the exit handler starts once the group has ended")
			r.drive(ctx, 10, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
			assert.Equal(t, wfv1.WorkflowFailed, r.woc.wf.Status.Phase)
			assert.Equal(t, "child '"+stuck.ID+"' failed", r.woc.wf.Status.Message)
		})
	}
}

// TestStepsMissingOutputWaitsAlone verifies that a step referring to an
// output its producer never reported (the producer failed, and continueOn
// let the template carry on) waits on its own: its siblings in the group
// start, while the step gets no node and the next group does not start.
// The step waits until the output is reported, as the whole group did before.
func TestStepsMissingOutputWaitsAlone(t *testing.T) {
	ctx, r := r4Start(t, "@testdata/steps/steps-missing-output-wf.yaml")
	r.r4SetPods(ctx, map[string]apiv1.PodPhase{"producer": apiv1.PodFailed})
	for range 3 {
		r.r4SetPods(ctx, nil)
	}

	assert.Equal(t, wfv1.NodeFailed, r.r4Phase("producer"))
	assert.Equal(t, wfv1.NodeSucceeded, r.r4Phase("[0]"), "continueOn.failed carries the template on")
	assert.NotEmpty(t, r.r4Phase("sibling"), "the consumer's sibling starts")
	assert.Empty(t, r.r4Phase("consumer"), "the consumer waits for the output, with no node")

	r.r4SetPods(ctx, map[string]apiv1.PodPhase{"sibling": apiv1.PodSucceeded})
	r.op(ctx)
	assert.Equal(t, wfv1.NodeSucceeded, r.r4Phase("sibling"))
	assert.Empty(t, r.r4Phase("consumer"))
	assert.Equal(t, wfv1.NodeRunning, r.r4Phase("[1]"), "the group waits for the consumer")
	assert.Empty(t, r.r4Phase("last"), "the next group does not start")
	assert.Equal(t, wfv1.WorkflowRunning, r.woc.wf.Status.Phase)
}

// TestStepsOmittedOutboundNode verifies that when a Steps template fails in
// an early group, its outbound node is the omitted step of its last group,
// which is now recorded, rather than the step that failed: a task that
// depends on the template hangs off that omitted step, and still runs.
func TestStepsOmittedOutboundNode(t *testing.T) {
	ctx, r := r4Start(t, "@testdata/steps/steps-omitted-outbound-wf.yaml")
	r.r4SetPods(ctx, map[string]apiv1.PodPhase{"first": apiv1.PodFailed})
	r.op(ctx)

	inner := r.woc.wf.Status.Nodes.FindByDisplayName("inner")
	require.NotNil(t, inner)
	assert.Equal(t, wfv1.NodeFailed, inner.Phase)
	second := r.woc.wf.Status.Nodes.FindByDisplayName("second")
	require.NotNil(t, second)
	assert.Equal(t, wfv1.NodeOmitted, second.Phase)
	assert.Equal(t, []string{second.ID}, inner.OutboundNodes, "the outbound node is the omitted step")

	dependant := r.woc.wf.Status.Nodes.FindByDisplayName("dependant")
	require.NotNil(t, dependant, "continueOn.failed lets the dependant run")
	assert.Contains(t, second.Children, dependant.ID, "the dependant hangs off the omitted step")

	r.drive(ctx, 10, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase)
}

// Leading empty step groups must stay complete while their successor runs.
func TestEmptyStepGroupCompletes(t *testing.T) {
	for _, phase := range []wfv1.NodePhase{wfv1.NodeSucceeded, wfv1.NodeFailed} {
		t.Run(string(phase), func(t *testing.T) {
			for _, test := range []struct {
				name        string
				emptyGroups int
				workflow    string
			}{
				{
					name:        "OneEmptyGroup",
					emptyGroups: 1,
					workflow:    "@testdata/steps/empty-step-group-one-empty-group.yaml",
				},
				{
					name:        "TwoEmptyGroups",
					emptyGroups: 2,
					workflow:    "@testdata/steps/empty-step-group-two-empty-groups.yaml",
				},
			} {
				t.Run(test.name, func(t *testing.T) {
					ctx := logging.TestContext(t.Context())
					wf := wfv1.MustUnmarshalWorkflow(test.workflow)
					woc := newWoc(ctx, *wf)
					woc.operate(ctx)
					require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
					childName := fmt.Sprintf("%s[%d].hello", wf.Name, test.emptyGroups)
					child, err := woc.wf.GetNodeByName(childName)
					require.NoError(t, err)
					require.Equal(t, wfv1.NodeRunning, child.Phase)

					// Reconcile again while the real step is still running. This used to reopen
					// the empty group, blocking all subsequent progress through the template.
					for range 2 {
						woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
						woc.operate(ctx)
						require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
						for i := range test.emptyGroups {
							group, err := woc.wf.GetNodeByName(fmt.Sprintf("%s[%d]", wf.Name, i))
							require.NoError(t, err)
							assert.Equal(t, wfv1.NodeSucceeded, group.Phase)
						}
					}

					woc.markNodePhase(ctx, childName, phase)
					woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
					woc.operate(ctx)
					expected := wfv1.WorkflowSucceeded
					if phase == wfv1.NodeFailed {
						expected = wfv1.WorkflowFailed
					}
					assert.Equal(t, expected, woc.wf.Status.Phase)
					assert.False(t, woc.wf.Status.FinishedAt.IsZero())
				})
			}
		})
	}
}
