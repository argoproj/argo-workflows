package executor

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel/trace"

	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TaskSource yields the tasks a process runs, one at a time.
type TaskSource interface {
	// Next returns the next task. Check err first; ok=false means the
	// source is exhausted.
	Next(ctx context.Context) (task Task, ok bool, err error)
}

// Role is the part of a task's flow one process runs: in a pod the flow
// is split across containers (init, main, wait or supervisor), while a
// worker runs all of it. Every Role is declared in roles.go, the registry.
type Role struct {
	Name string
	// Layout is how the flow is split across processes, which selects the
	// plan: each role belongs to exactly one layout.
	Layout Layout
	// StartSpan starts the task's top-level span.
	StartSpan func(we *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span)
	// Run is the role's part of the flow. ctx is cancellable; bgCtx is not,
	// so outputs are still captured during termination.
	Run func(ctx, bgCtx context.Context, we *WorkflowExecutor) error
	// Finally runs on bgCtx after Run, even if Run panicked. Errors are
	// recorded with AddError.
	Finally []Stage
}

// Harness runs every task its source yields through one role, each with a
// fresh executor and its own result sink. Whatever goes wrong with one task
// is reported through that task's sink and does not stop the next; a pod's
// source yields once, so there the outcome is the same as running the task
// directly.
type Harness struct {
	Process *Process
	Source  TaskSource
	// Sink builds the result sink for a task.
	Sink func(Task) ResultSink
	Role Role
}

// Run runs tasks until the source is exhausted or ctx is cancelled. It
// returns the source's error if Next fails, an error if the source yielded
// nothing, and otherwise the first task's error (nil if every task
// succeeded). The first task always runs, so a pod's task still collects
// and finalizes after SIGTERM; once ctx is cancelled no further task is
// taken from the source, and the cancellation itself is not an error.
//
// Process-wide setup and teardown (tracing shutdown, stats) belong to the
// caller.
func (h *Harness) Run(ctx context.Context) error {
	var firstErr error
	ran := false
	for !ran || ctx.Err() == nil {
		task, ok, err := h.Source.Next(ctx)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		ran = true
		if err := h.runTask(ctx, task); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !ran {
		return errors.New("no task to run")
	}
	return firstErr
}

// runTask runs one task through the role on its own context, which ends
// with the task. Its failure (a panic, else the first recorded error, else
// the error Run returned) is reported to the sink before the Finally stages
// run, so a result is never marked complete without it. Each Finally stage
// runs even if an earlier one panicked. A panic anywhere returns exit code
// 1 with the panic as its message, which is what a pod reported before
// tasks had a harness.
//
//nolint:contextcheck
func (h *Harness) runTask(ctx context.Context, task Task) (err error) {
	sink := h.Sink(task)
	we := h.Process.NewExecutor(task, h.Role.Layout, sink)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx, span := h.Role.StartSpan(we, ctx)
	defer span.End()
	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	bgCtx := trace.ContextWithSpan(logging.RequireLoggerFromContext(ctx).NewBackgroundContext(), span)

	failed := false
	fail := func(cause error) {
		if !failed {
			failed = true
			sink.Fail(bgCtx, cause)
		}
	}
	var panicCause error
	//nolint:revive // exitOnPanic is only ever called directly by defer
	exitOnPanic := func() {
		if r := recover(); r != nil {
			cause := recoveredCause(bgCtx, r)
			fail(cause)
			if panicCause == nil {
				panicCause = cause
			}
		}
	}

	err = func() error {
		defer exitOnPanic()
		return h.Role.Run(ctx, bgCtx, we)
	}()
	switch first := we.HasError(); {
	case panicCause != nil:
	case first != nil:
		fail(first)
	case err != nil:
		fail(err)
	}

	for _, s := range h.Role.Finally {
		func() {
			defer exitOnPanic()
			if stageErr := s.Run(bgCtx, we); stageErr != nil {
				we.AddError(bgCtx, stageErr)
			}
		}()
	}
	// An error from Finally is the task's failure only if nothing failed
	// before it.
	if first := we.HasError(); first != nil {
		fail(first)
	}
	if panicCause != nil {
		// In a pod, main() writes this cause to the termination log again;
		// it is the same text the sink wrote.
		return argoerrors.NewExitErrWithCause(1, panicCause)
	}
	return err
}

// recoveredCause logs a recovered panic and returns it as an error, keeping
// the value itself when it already is one.
func recoveredCause(ctx context.Context, r any) error {
	logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
		"error": r,
		"stack": string(debug.Stack()),
	}).Error(ctx, "executor panic")
	if cause, ok := r.(error); ok {
		return cause
	}
	return fmt.Errorf("%v", r)
}
