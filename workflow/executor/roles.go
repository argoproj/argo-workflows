package executor

import (
	"context"
	"runtime/debug"

	"go.opentelemetry.io/otel/trace"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// This file is the registry of every role an argoexec process can play:
// the part of a task's flow it runs. A Role literal anywhere else in the
// package fails TestRoleLiteralsLiveInRegistry.

var (
	// InitRole is the legacy init container: the plan's Prepare phase.
	InitRole = Role{
		Name:   "init",
		Layout: LayoutLegacy,
		StartSpan: func(we *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return we.Tracing.StartRunInitContainer(ctx, we.WorkflowName(), we.Namespace)
		},
		Run: func(ctx, _ context.Context, we *WorkflowExecutor) error {
			if err := we.Prepare(ctx); err != nil {
				we.AddError(ctx, err)
				return err
			}
			return nil
		},
	}

	// WaitRole is the legacy wait container: everything after main starts.
	// Pre-main ran in the init container; had it failed, the pod would
	// never have got here.
	WaitRole = Role{
		Name:   "wait",
		Layout: LayoutLegacy,
		StartSpan: func(we *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return we.Tracing.StartRunWaitContainer(ctx, we.WorkflowName(), we.Namespace)
		},
		Run: func(ctx, bgCtx context.Context, we *WorkflowExecutor) error {
			return we.PostMain(ctx, bgCtx, false)
		},
		Finally: []Stage{killArtifactSidecars, finalizeOutput},
	}

	// SupervisorRole is the init-less auxiliary container: it prepares
	// main, signals main through the status marker, then collects outputs.
	SupervisorRole = Role{
		Name:   "supervisor",
		Layout: LayoutInitless,
		StartSpan: func(we *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return we.Tracing.StartRunSupervisorContainer(ctx, we.WorkflowName(), we.Namespace)
		},
		Run: func(ctx, bgCtx context.Context, we *WorkflowExecutor) error {
			marker := StatusMarkerPath(we.paths)
			// Heartbeat the status marker so main's emissary can tell a live (but
			// slow) supervisor from a dead one, rather than blocking to the pod
			// deadline. The initial write also overwrites any stale marker from a
			// prior attempt. Stop the heartbeat before the terminal write so no
			// heartbeat races it on the same path.
			stopHeartbeat, hbErr := startStatusHeartbeat(ctx, marker)
			if hbErr != nil {
				// We can't write the marker at all (e.g. broken mount), so main
				// can't be signalled — it will presume us dead via the
				// never-appeared timeout. Record the error and collect logs.
				we.AddError(ctx, hbErr)
				return we.PostMain(ctx, bgCtx, true)
			}

			// Zero umask so files/dirs created by Prepare are accessible to main
			// when it runs as a different uid. The legacy init container does the
			// same. A stale marker from a prior attempt needs no explicit cleanup:
			// the heartbeat's initial RUNNING write has already overwritten it.
			osspecific.AllowGrantingAccessToEveryone()
			preMainErr := we.Prepare(ctx)
			stopHeartbeat()

			preMainFailed := false
			if preMainErr != nil {
				// Surface the error to the emissary in main via the status marker,
				// then fall through to PostMain. PostMain waits for main to exit —
				// emissary in main reads the failure status, exits 65, and writes
				// its exitcode file — and then captures main's stdout/stderr (which
				// includes the supervisor failure message echoed by emissary) into
				// the task result. We must NOT write a success status here.
				writeFailureStatus(ctx, marker, preMainErr)
				we.AddError(ctx, preMainErr)
				preMainFailed = true
			} else if err := writeSuccessStatus(marker); err != nil {
				// The success status write failed (e.g. disk full). Without a
				// terminal status, main's emissary would keep waiting until the
				// heartbeat-staleness timeout. Write a failure status so main
				// exits promptly, then fall through to PostMain (logs only).
				writeFailureStatus(ctx, marker, err)
				we.AddError(ctx, err)
				preMainFailed = true
			} else {
				// Pre-main is done. Input-artifact downloads can buffer a lot
				// of heap, and all of it is now garbage. From here the
				// supervisor only babysits main (PostMain blocks until main
				// exits), so a blocking collection costs no latency. Use
				// FreeOSMemory rather than runtime.GC so the freed pages go
				// back to the kernel — main shares the pod's memory budget and
				// is about to do the real work.
				debug.FreeOSMemory()
			}
			return we.PostMain(ctx, bgCtx, preMainFailed)
		},
		Finally: []Stage{killArtifactSidecars, finalizeOutput},
	}

	// WorkerRole runs a task's whole flow in this process: check the
	// template, prepare the task's own tree, run main through the process's
	// MainDriver, collect, then remove the task's files. Exercised only by
	// tests until a worker command exists.
	WorkerRole = Role{
		Name:   "worker",
		Layout: LayoutWorker,
		StartSpan: func(we *WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return we.Tracing.StartRunWorkerTask(ctx, we.WorkflowName(), we.Namespace)
		},
		Run: func(ctx, bgCtx context.Context, we *WorkflowExecutor) error {
			// Main runs in the Run phase, so if Prepare failed nothing ran
			// and there is nothing to collect.
			if err := we.Prepare(ctx); err != nil {
				we.AddError(ctx, err)
				return err
			}
			return we.PostMain(ctx, bgCtx, false)
		},
		Finally: []Stage{finalizeOutput, removeTaskRoot},
	}
)
