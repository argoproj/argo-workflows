package executor

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// countingStage returns a stage that counts its calls and runs fn if set.
func countingStage(name string, calls *atomic.Int32, fn func(ctx context.Context) error) Stage {
	return Stage{Name: name, Run: func(ctx context.Context, _ *WorkflowExecutor) error {
		calls.Add(1)
		if fn != nil {
			return fn(ctx)
		}
		return nil
	}}
}

func stageNames(stages []Stage) []string {
	names := make([]string, len(stages))
	for i, s := range stages {
		names[i] = s.Name
	}
	return names
}

func TestPrepare_RunsStagesInOrder(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var order []string
	record := func(name string) Stage {
		return Stage{Name: name, Run: func(context.Context, *WorkflowExecutor) error {
			order = append(order, name)
			return nil
		}}
	}
	we := &WorkflowExecutor{plan: &Plan{Prepare: []Stage{record("a"), record("b"), record("c")}}}

	require.NoError(t, we.Prepare(ctx))
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestPrepare_FirstErrorShortCircuits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var aCalls, bCalls, cCalls atomic.Int32
	we := &WorkflowExecutor{plan: &Plan{Prepare: []Stage{
		countingStage("a", &aCalls, nil),
		countingStage("b", &bCalls, func(context.Context) error { return errors.New("b boom") }),
		countingStage("c", &cCalls, nil),
	}}}

	err := we.Prepare(ctx)
	require.ErrorContains(t, err, "b: b boom", "the error names the failing stage")
	assert.EqualValues(t, 1, aCalls.Load())
	assert.EqualValues(t, 1, bCalls.Load())
	assert.EqualValues(t, 0, cCalls.Load(), "stages after the failure must not run")
}

func TestParallel_OneFailureCancelsSiblings(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// "fail" returns an error immediately. The others block on ctx.Done()
	// and must observe cancellation and return.
	var calls atomic.Int32
	blockUntilCancelled := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	we := &WorkflowExecutor{plan: &Plan{Prepare: []Stage{Parallel(
		countingStage("ok0", &calls, blockUntilCancelled),
		countingStage("fail", &calls, func(context.Context) error { return errors.New("plugin boom") }),
		countingStage("ok1", &calls, blockUntilCancelled),
		countingStage("ok2", &calls, blockUntilCancelled),
	)}}}

	done := make(chan error, 1)
	go func() { done <- we.Prepare(ctx) }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "plugin boom",
			"the failing child's error should surface; sibling cancellations should not mask it")
	case <-time.After(5 * time.Second):
		t.Fatal("Prepare did not return; siblings did not observe context cancellation")
	}
	assert.EqualValues(t, 4, calls.Load(), "all children must have been entered")
}

func TestParallel_NameListsChildren(t *testing.T) {
	var n atomic.Int32
	s := Parallel(countingStage("x", &n, nil), countingStage("y", &n, nil))
	assert.Equal(t, "parallel(x,y)", s.Name)
}

func TestPostMain_SkipsMainOutputStagesWhenPreMainFailed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var runCalls, needsMainCalls, alwaysCalls atomic.Int32
	needsMain := countingStage("needs-main", &needsMainCalls, nil)
	needsMain.NeedsMainOutputs = true
	we := &WorkflowExecutor{plan: &Plan{
		Run:     []Stage{countingStage("run", &runCalls, nil)},
		Collect: []Stage{needsMain, countingStage("always", &alwaysCalls, nil)},
	}}

	require.NoError(t, we.PostMain(ctx, ctx, true))
	assert.EqualValues(t, 1, runCalls.Load())
	assert.EqualValues(t, 0, needsMainCalls.Load())
	assert.EqualValues(t, 1, alwaysCalls.Load())
}

func TestPostMain_RecordsErrorsAndContinues(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var afterCalls atomic.Int32
	we := &WorkflowExecutor{plan: &Plan{
		Run:     []Stage{countingStage("wait", new(atomic.Int32), func(context.Context) error { return errors.New("wait boom") })},
		Collect: []Stage{countingStage("after", &afterCalls, func(context.Context) error { return errors.New("after boom") })},
	}}

	err := we.PostMain(ctx, ctx, false)
	require.ErrorContains(t, err, "wait boom", "the first error is the node's message")
	assert.EqualValues(t, 1, afterCalls.Load(), "a Run error must not stop Collect")
	assert.Len(t, we.errors, 2)
}

func TestPlanFor_InitlessLayout(t *testing.T) {
	plan := planFor(&wfv1.Template{Container: &apiv1.Container{}}, true, []wfv1.ArtifactPluginName{"s3", "gcs"})

	assert.Equal(t, []string{
		"write-template",
		"stage-files",
		"parallel(load-artifacts,load-artifacts-from-plugin(s3),load-artifacts-from-plugin(gcs))",
	}, stageNames(plan.Prepare))
	assert.Equal(t, []string{"initialize-output", "wait"}, stageNames(plan.Run))
	assert.Equal(t, []string{"capture-script-result", "save-parameters", "save-artifacts", "save-logs", "report-outputs"}, stageNames(plan.Collect))
}

func TestPlanFor_InitlessNoPlugins(t *testing.T) {
	plan := planFor(&wfv1.Template{Container: &apiv1.Container{}}, true, nil)
	assert.Equal(t, "parallel(load-artifacts)", plan.Prepare[2].Name)
}

// The legacy layout installs the binary and never loads plugin artifacts in
// this container: each plugin has its own artifact-plugin-init container.
func TestPlanFor_LegacyLayout(t *testing.T) {
	plan := planFor(&wfv1.Template{Container: &apiv1.Container{}}, false, []wfv1.ArtifactPluginName{"s3"})

	assert.Equal(t, []string{"install-argoexec", "stage-files", "load-artifacts"}, stageNames(plan.Prepare))
	assert.Equal(t, []string{"initialize-output", "wait"}, stageNames(plan.Run))
	assert.Equal(t, []string{"capture-script-result", "save-parameters", "save-artifacts", "save-logs", "report-outputs"}, stageNames(plan.Collect))
}

func TestPlanFor_ResourceTemplateCollectsLogsOnly(t *testing.T) {
	for _, initless := range []bool{false, true} {
		plan := planFor(&wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "get"}}, initless, nil)
		assert.Equal(t, []string{"report-logs"}, stageNames(plan.Collect), "initless=%v", initless)
	}
}

// TestPlansAreWellFormed is the plan validation: every selectable plan has
// uniquely named, runnable stages, and only Collect stages may depend on
// main's outputs (the skip flag is meaningless in the other phases).
func TestPlansAreWellFormed(t *testing.T) {
	templates := map[string]*wfv1.Template{
		"container": {Container: &apiv1.Container{}},
		"script":    {Script: &wfv1.ScriptTemplate{}},
		"resource":  {Resource: &wfv1.ResourceTemplate{}},
	}
	for name, tmpl := range templates {
		for _, initless := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/initless=%v", name, initless), func(t *testing.T) {
				plan := planFor(tmpl, initless, []wfv1.ArtifactPluginName{"s3"})
				seen := map[string]bool{}
				for phase, stages := range map[string][]Stage{"Prepare": plan.Prepare, "Run": plan.Run, "Collect": plan.Collect} {
					require.NotEmpty(t, stages, "phase %s has no stages", phase)
					for _, s := range stages {
						assert.NotEmpty(t, s.Name, "phase %s has an unnamed stage", phase)
						assert.NotNil(t, s.Run, "stage %s has no Run", s.Name)
						assert.False(t, seen[s.Name], "stage %s appears twice", s.Name)
						seen[s.Name] = true
						if s.NeedsMainOutputs {
							assert.Equal(t, "Collect", phase, "stage %s needs main outputs but is in %s", s.Name, phase)
						}
					}
				}
			})
		}
	}
}
