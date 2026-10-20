//go:build !windows

package executor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	utilretry "github.com/argoproj/argo-workflows/v4/util/retry"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

// panicky runs main with the container driver, except for one node, where
// it panics.
type panicky struct{ node string }

func (d panicky) Run(ctx context.Context, task maindriver.Task, stager maindriver.OutputStager) (int, error) {
	if task.NodeID == d.node {
		panic("driver blew up")
	}
	return maindriver.Container{}.Run(ctx, task, stager)
}

// Several tasks run one after another in one process through one harness:
// each gets its own executor, paths and result, a failure (exit code,
// deadline, panic) is attributed to its own task, and the next task still
// runs.
//
// Known gap (by design for now): user paths are not cleaned between tasks,
// so a stale file at a declared output path from an earlier task could be
// read as a later task's optional output.
func TestWorkerHarness_RunsTasksInOneProcess(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	results := argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults("ns")
	process, err := executor.NewProcess(ctx, nil, results, nil,
		func(paths executor.TaskPaths) executor.ContainerRuntimeExecutor {
			return emissary.New(paths.VarRunArgo)
		},
		executor.ProcessConfig{
			Namespace:              "ns",
			TerminationGracePeriod: 100 * time.Millisecond,
			RetryBackoff:           utilretry.ExecutorRetry(ctx),
			Environ:                os.Environ(), //nolint:forbidigo // commands in the test need the test process's PATH
			MainDriver:             panicky{node: "n3"},
		})
	require.NoError(t, err)

	root := t.TempDir()
	user := t.TempDir()
	param := filepath.Join(user, "p")
	input := filepath.Join(user, "in")
	sh := func(script string) *apiv1.Container { return &apiv1.Container{Command: []string{"sh", "-c", script}} }
	task := func(node string, tmpl wfv1.Template, includeScriptOutput bool) executor.Task {
		paths, err := executor.WorkerPaths(root, node)
		require.NoError(t, err)
		return executor.Task{
			Config: executor.TaskConfig{WorkflowName: "wf", NodeID: node, Template: tmpl, IncludeScriptOutput: includeScriptOutput},
			Paths:  paths,
		}
	}
	tasks := []executor.Task{
		task("n1", wfv1.Template{
			Script:  &wfv1.ScriptTemplate{Container: apiv1.Container{Command: []string{"sh"}}, Source: "printf r1; printf p1 > " + param},
			Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: param}}}},
		}, true),
		task("n2", wfv1.Template{Container: sh("exit 7")}, true),
		task("n3", wfv1.Template{Container: sh("true")}, false),
		task("n4", wfv1.Template{Container: sh("exec sleep 30")}, false),
		task("n5", wfv1.Template{
			Container: &apiv1.Container{Command: []string{"cat", input}},
			Inputs:    wfv1.Inputs{Artifacts: wfv1.Artifacts{{Name: "in", Path: input, ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "hello"}}}}},
		}, true),
	}
	// n4's deadline starts when it is taken from the source, so it expires
	// while n4 runs, not before.
	source := &sliceSource{tasks: tasks, deadlines: map[string]time.Duration{"n4": time.Second}}

	start := time.Now()
	runErr := (&executor.Harness{
		Process: process,
		Source:  source,
		Sink:    func(task executor.Task) executor.ResultSink { return process.NewTaskResultSink(task.Config, true) },
		Role:    executor.WorkerRole,
	}).Run(ctx)
	assert.Less(t, time.Since(start), 20*time.Second)
	require.EqualError(t, runErr, "driver blew up", "the first task error is the panic; exit codes and deadlines are failures, not errors")

	get := func(node string) *wfv1.WorkflowTaskResult {
		r, err := results.Get(ctx, node, metav1.GetOptions{})
		require.NoError(t, err, node)
		assert.Equal(t, "true", r.Labels["workflows.argoproj.io/report-outputs-completed"], node)
		return r
	}
	r1 := get("n1")
	assert.Empty(t, r1.Phase)
	require.NotNil(t, r1.Outputs)
	require.NotNil(t, r1.Outputs.Result)
	assert.Equal(t, "r1", *r1.Outputs.Result)
	require.Len(t, r1.Outputs.Parameters, 1)
	assert.Equal(t, "p1", r1.Outputs.Parameters[0].Value.String())

	r2 := get("n2")
	assert.Equal(t, wfv1.NodeFailed, r2.Phase)
	assert.Contains(t, r2.Message, "exit status 7")
	require.NotNil(t, r2.Outputs)
	require.NotNil(t, r2.Outputs.Result, "n2 captures its own (empty) stdout")
	assert.Empty(t, *r2.Outputs.Result, "n2 must not see n1's stdout")

	r3 := get("n3")
	assert.Equal(t, wfv1.NodeFailed, r3.Phase)
	assert.Equal(t, "driver blew up", r3.Message)

	r4 := get("n4")
	assert.Equal(t, wfv1.NodeFailed, r4.Phase)
	assert.Equal(t, "Step exceeded its deadline", r4.Message)

	r5 := get("n5")
	assert.Empty(t, r5.Phase, "the task after a panic still runs")
	require.NotNil(t, r5.Outputs)
	require.NotNil(t, r5.Outputs.Result)
	assert.Equal(t, "hello", *r5.Outputs.Result)

	for _, task := range tasks {
		assert.NoDirExists(t, task.Paths.Root)
	}
}

// sliceSource yields its tasks in order, then reports exhausted. A task
// listed in deadlines gets a deadline that long after it is yielded.
type sliceSource struct {
	tasks     []executor.Task
	deadlines map[string]time.Duration
}

func (s *sliceSource) Next(context.Context) (executor.Task, bool, error) {
	if len(s.tasks) == 0 {
		return executor.Task{}, false, nil
	}
	t := s.tasks[0]
	s.tasks = s.tasks[1:]
	if d, ok := s.deadlines[t.Config.NodeID]; ok {
		t.Config.Deadline = time.Now().Add(d)
	}
	return t, true, nil
}
