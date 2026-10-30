package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// sliceSource yields its tasks in order, then reports exhausted.
type sliceSource struct {
	tasks []Task
	err   error
}

func (s *sliceSource) Next(context.Context) (Task, bool, error) {
	if s.err != nil {
		return Task{}, false, s.err
	}
	if len(s.tasks) == 0 {
		return Task{}, false, nil
	}
	t := s.tasks[0]
	s.tasks = s.tasks[1:]
	return t, true, nil
}

// recordingSink records failures and ignores everything else.
type recordingSink struct{ failures []string }

func (s *recordingSink) Start(context.Context) error                   { return nil }
func (s *recordingSink) Report(context.Context, wfv1.NodeResult) error { return nil }
func (s *recordingSink) Progress(context.Context, wfv1.Progress) error { return nil }
func (s *recordingSink) Fail(_ context.Context, err error) {
	s.failures = append(s.failures, err.Error())
}
func (s *recordingSink) Finalize(context.Context) error { return nil }

func tasks(ids ...string) []Task {
	out := make([]Task, len(ids))
	for i, id := range ids {
		out[i] = Task{Config: TaskConfig{NodeID: id}, Paths: PodPaths()}
	}
	return out
}

// testRole is a role whose Run is supplied by the test.
func testRole(run func(ctx, bgCtx context.Context, we *WorkflowExecutor) error, finally ...Stage) Role {
	return Role{
		Name: "test",
		StartSpan: func(_ *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return ctx, trace.SpanFromContext(ctx)
		},
		Run:     run,
		Finally: finally,
	}
}

// newTestHarness returns a harness over source whose sinks are recorded
// in the returned map, by node ID.
func newTestHarness(t *testing.T, source TaskSource, role Role) (context.Context, *Harness, map[string]*recordingSink) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	tr, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	sinks := map[string]*recordingSink{}
	return ctx, &Harness{
		Process: &Process{
			Tracing:    tr,
			newRuntime: func(TaskPaths) ContainerRuntimeExecutor { return &mocks.ContainerRuntimeExecutor{} },
		},
		Source: source,
		Sink: func(task Task) ResultSink {
			s := &recordingSink{}
			sinks[task.Config.NodeID] = s
			return s
		},
		Role: role,
	}, sinks
}

func TestHarness_RunsEachTaskWithFreshExecutor(t *testing.T) {
	var seen []*WorkflowExecutor
	ctx, h, sinks := newTestHarness(t, &sliceSource{tasks: tasks("a", "b")}, testRole(func(ctx, _ context.Context, we *WorkflowExecutor) error {
		seen = append(seen, we)
		if len(seen) == 1 {
			we.AddError(ctx, errors.New("only in a"))
		}
		return nil
	}))
	require.NoError(t, h.Run(ctx), "the role returned nil; the recorded error still reaches the sink")
	require.Len(t, seen, 2)
	assert.NotSame(t, seen[0], seen[1])
	require.NoError(t, seen[1].HasError())
	assert.Equal(t, []string{"only in a"}, sinks["a"].failures)
	assert.Empty(t, sinks["b"].failures)
}

func TestHarness_PanicReportsAndExitsOne(t *testing.T) {
	finallyRan := false
	ctx, h, sinks := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(
		func(context.Context, context.Context, *WorkflowExecutor) error { panic("kaboom") },
		Stage{Name: "finally", Run: func(context.Context, *WorkflowExecutor) error { finallyRan = true; return nil }},
	))
	err := h.Run(ctx)
	assert.True(t, finallyRan)
	assert.Equal(t, []string{"kaboom"}, sinks["a"].failures)
	// A direct assertion, as main() makes: wrapping would lose exit code 1.
	exited, ok := err.(argoerrors.Exited)
	require.True(t, ok, "%T is not an unwrapped Exited", err)
	assert.Equal(t, 1, exited.ExitCode())
	assert.EqualError(t, err, "kaboom")
}

func TestHarness_FirstRecordedErrorReachesSink(t *testing.T) {
	ctx, h, sinks := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(func(ctx, _ context.Context, we *WorkflowExecutor) error {
		we.AddError(ctx, errors.New("first"))
		we.AddError(ctx, errors.New("second"))
		return we.HasError()
	}))
	require.EqualError(t, h.Run(ctx), "first")
	assert.Equal(t, []string{"first"}, sinks["a"].failures)
}

func TestHarness_ContinuesAfterFailedTask(t *testing.T) {
	var ran []string
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: tasks("a", "b", "c")}, testRole(func(_, _ context.Context, we *WorkflowExecutor) error {
		ran = append(ran, we.nodeID)
		if we.nodeID == "b" {
			return errors.New("b failed")
		}
		return nil
	}))
	require.EqualError(t, h.Run(ctx), "b failed")
	assert.Equal(t, []string{"a", "b", "c"}, ran)
}

func TestHarness_NextErrorReturned(t *testing.T) {
	ctx, h, _ := newTestHarness(t, &sliceSource{err: errors.New("no source")}, testRole(func(context.Context, context.Context, *WorkflowExecutor) error { return nil }))
	assert.EqualError(t, h.Run(ctx), "no source")
}

func TestHarness_ZeroTasksIsError(t *testing.T) {
	ctx, h, _ := newTestHarness(t, &sliceSource{}, testRole(func(context.Context, context.Context, *WorkflowExecutor) error { return nil }))
	assert.ErrorIs(t, h.Run(ctx), ErrNoTask)
}

func TestHarness_StopsBetweenTasksOnCancel(t *testing.T) {
	var ran []string
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: tasks("a", "b")}, Role{})
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	h.Role = testRole(func(_, _ context.Context, we *WorkflowExecutor) error {
		ran = append(ran, we.nodeID)
		cancel()
		return nil
	})
	require.NoError(t, h.Run(ctx))
	assert.Equal(t, []string{"a"}, ran)
}

func TestHarness_CancelAfterLastTaskKeepsTaskResult(t *testing.T) {
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: tasks("a")}, Role{})
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	h.Role = testRole(func(context.Context, context.Context, *WorkflowExecutor) error {
		cancel()
		return errors.New("task")
	})
	assert.EqualError(t, h.Run(ctx), "task")
}

// A panic with an error value keeps that error as the cause.
func TestHarness_PanicKeepsErrorValue(t *testing.T) {
	sentinel := errors.New("sentinel")
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(
		func(context.Context, context.Context, *WorkflowExecutor) error { panic(sentinel) },
	))
	assert.ErrorIs(t, h.Run(ctx), sentinel)
}

// A pod's only task must run even if SIGTERM arrived during setup, so its
// outputs are still collected and finalized.
func TestHarness_RunsFirstTaskEvenIfCancelled(t *testing.T) {
	ran := false
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(func(context.Context, context.Context, *WorkflowExecutor) error {
		ran = true
		return nil
	}))
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	require.NoError(t, h.Run(ctx))
	assert.True(t, ran)
}

// A panic in one Finally stage must not skip the later ones.
func TestHarness_FinallyPanicStillRunsLaterStages(t *testing.T) {
	finalized := false
	ctx, h, sinks := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(
		func(context.Context, context.Context, *WorkflowExecutor) error { return nil },
		Stage{Name: "boom", Run: func(context.Context, *WorkflowExecutor) error { panic("finally kaboom") }},
		Stage{Name: "finalize", Run: func(context.Context, *WorkflowExecutor) error { finalized = true; return nil }},
	))
	err := h.Run(ctx)
	assert.True(t, finalized)
	assert.Equal(t, []string{"finally kaboom"}, sinks["a"].failures)
	var exited argoerrors.Exited
	require.ErrorAs(t, err, &exited)
	assert.Equal(t, 1, exited.ExitCode())
}

// The failure must be reported before Finally marks the outputs complete.
func TestHarness_FailureReportedBeforeFinally(t *testing.T) {
	var failuresAtFinally []string
	var sinks map[string]*recordingSink
	ctx, h, got := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(
		func(ctx, _ context.Context, we *WorkflowExecutor) error {
			we.AddError(ctx, errors.New("failed"))
			return we.HasError()
		},
		Stage{Name: "finalize", Run: func(context.Context, *WorkflowExecutor) error {
			failuresAtFinally = append([]string(nil), sinks["a"].failures...)
			return nil
		}},
	))
	sinks = got
	_ = h.Run(ctx)
	assert.Equal(t, []string{"failed"}, failuresAtFinally)
}

// An error a role returns without recording it still reaches the sink.
func TestHarness_ReturnedErrorReachesSink(t *testing.T) {
	ctx, h, sinks := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(func(context.Context, context.Context, *WorkflowExecutor) error {
		return errors.New("unrecorded")
	}))
	require.EqualError(t, h.Run(ctx), "unrecorded")
	assert.Equal(t, []string{"unrecorded"}, sinks["a"].failures)
}

// Goroutines a task starts on its context stop when the task ends.
func TestHarness_TaskContextEndsWithTask(t *testing.T) {
	var taskCtx context.Context
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: tasks("a")}, testRole(func(ctx, _ context.Context, _ *WorkflowExecutor) error {
		taskCtx = ctx
		return nil
	}))
	require.NoError(t, h.Run(ctx))
	assert.Error(t, taskCtx.Err())
}

// The cleanup each role runs is part of its contract with the controller:
// marking outputs complete comes last, after anything that can still fail
// the task (killing sidecars, removing a worker task's files), and init
// (which runs before main) never marks them complete.
func TestRoles_Finally(t *testing.T) {
	names := func(stages []Stage) []string {
		out := []string{}
		for _, s := range stages {
			out = append(out, s.Name)
		}
		return out
	}
	assert.Empty(t, names(InitRole.Finally))
	assert.Equal(t, []string{"kill-artifact-sidecars", "finalize-output"}, names(WaitRole.Finally))
	assert.Equal(t, []string{"kill-artifact-sidecars", "finalize-output"}, names(SupervisorRole.Finally))
	assert.Equal(t, []string{"remove-task-root", "finalize-output"}, names(WorkerRole.Finally))
}

// Each task's spans continue the trace context it came with.
func TestHarness_TaskContinuesItsTraceContext(t *testing.T) {
	const parent = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
	traced := tasks("traced", "untraced")
	traced[0].Config.TraceContext = map[string]string{"traceparent": parent}
	var got []trace.SpanContext
	ctx, h, _ := newTestHarness(t, &sliceSource{tasks: traced}, testRole(func(ctx, _ context.Context, _ *WorkflowExecutor) error {
		got = append(got, trace.SpanContextFromContext(ctx))
		return nil
	}))
	require.NoError(t, h.Run(ctx))
	require.Len(t, got, 2)
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", got[0].TraceID().String())
	assert.True(t, got[0].IsRemote())
	assert.False(t, got[1].IsValid(), "a task without a trace context doesn't continue another's")
}

// A task's top-level span continues the trace context it carries, as a
// pod's continue the controller's.
func TestHarness_TaskSpanContinuesItsTraceContext(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tr, exporter, err := tracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)
	task := Task{Paths: PodPaths(), Config: TaskConfig{
		NodeID:       "a",
		TraceContext: map[string]string{"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
	}}
	role := testRole(func(context.Context, context.Context, *WorkflowExecutor) error { return nil })
	role.StartSpan = func(we *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
		return we.Tracing.StartRunWaitContainer(ctx, we.WorkflowName(), we.Namespace)
	}
	h := &Harness{
		Process: &Process{
			Tracing:    tr,
			newRuntime: func(TaskPaths) ContainerRuntimeExecutor { return &mocks.ContainerRuntimeExecutor{} },
		},
		Source: &sliceSource{tasks: []Task{task}},
		Sink:   func(Task) ResultSink { return &recordingSink{} },
		Role:   role,
	}

	require.NoError(t, h.Run(ctx))

	span, err := exporter.GetSpanByName("runWaitContainer")
	require.NoError(t, err)
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", span.SpanContext().TraceID().String())
	assert.Equal(t, "b7ad6b7169203331", span.Parent().SpanID().String())
	assert.True(t, span.Parent().IsRemote())
}
