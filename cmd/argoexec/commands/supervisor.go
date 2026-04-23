package commands

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

func NewSupervisorCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "supervisor",
		Short: "init-less auxiliary: prepare main, signal readiness, then collect outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := supervisorContainer(cmd.Context()); err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		},
	}
	return &command
}

func supervisorContainer(ctx context.Context) error {
	return runAuxiliaryContainer(ctx,
		func(we *wfexecutor.WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return we.Tracing.StartRunSupervisorContainer(ctx, we.WorkflowName(), we.Namespace)
		},
		func(ctx, bgCtx context.Context, we *wfexecutor.WorkflowExecutor) error {
			preMainFailed := false
			if preMainErr := supervisorPreMain(ctx, we); preMainErr != nil {
				// Surface the error to the emissary in main via the failed marker,
				// then fall through to PostMain. PostMain waits for main to exit —
				// emissary in main observes the failed marker, exits 65, and writes
				// its exitcode file — and then captures main's stdout/stderr (which
				// includes the supervisor failure message echoed by emissary) into
				// the task result. We must NOT write the ready marker here.
				writeFailedMarker(ctx, preMainErr)
				we.AddError(ctx, preMainErr)
				preMainFailed = true
			} else if err := writeReadyMarker(); err != nil {
				// The ready marker write failed (e.g. disk full). Without a
				// failed marker, main's emissary would block on
				// waitForSupervisorReady indefinitely until the pod-level
				// deadline. Write the failed marker so main exits promptly,
				// then fall through to PostMain (logs only).
				writeFailedMarker(ctx, err)
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
	)
}

// preMainStages is the subset of WorkflowExecutor that supervisorPreMain
// drives. Carved out so the orchestration (umask, marker cleanup, sequence,
// parallel artifact loading, errgroup cancellation) can be tested without a
// full WorkflowExecutor.
type preMainStages interface {
	WriteTemplate() error
	StageFiles(ctx context.Context) error
	LoadArtifactsWithoutPlugins(ctx context.Context) error
	LoadArtifactsFromPlugin(ctx context.Context, pluginName wfv1.ArtifactPluginName) error
}

// supervisorPreMain runs the pre-main phase: template write, script staging,
// and input artifact download (non-plugin and per-plugin in parallel).
func supervisorPreMain(ctx context.Context, wfExecutor *wfexecutor.WorkflowExecutor) error {
	// Zero umask so files/dirs created here are accessible to main when it
	// runs as a different uid. The legacy init container does the same.
	osspecific.AllowGrantingAccessToEveryone()

	// Make restarts idempotent: a previous crashed attempt may have left a
	// failed marker in the shared emptyDir, which would otherwise race the
	// ready marker on the emissary's watcher and fail the pod even after
	// pre-main succeeds. Ignore not-exist; surface other errors so a broken
	// mount fails fast rather than silently producing the wrong outcome.
	if err := os.Remove(common.FailedMarkerPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean stale failed marker: %w", err)
	}

	return runSupervisorPreMain(ctx, wfExecutor, inputArtifactPluginNames())
}

// runSupervisorPreMain is the testable core of supervisorPreMain. Takes
// the plugin name list explicitly so tests don't have to mutate env vars.
func runSupervisorPreMain(ctx context.Context, stages preMainStages, pluginNames []wfv1.ArtifactPluginName) error {
	if err := stages.WriteTemplate(); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}
	if err := stages.StageFiles(ctx); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := stages.LoadArtifactsWithoutPlugins(gctx); err != nil {
			return fmt.Errorf("failed to load non-plugin input artifacts: %w", err)
		}
		return nil
	})
	for _, name := range pluginNames {
		g.Go(func() error {
			if err := stages.LoadArtifactsFromPlugin(gctx, name); err != nil {
				return fmt.Errorf("failed to load input artifacts from plugin %q: %w", name, err)
			}
			return nil
		})
	}
	return g.Wait()
}

// inputArtifactPluginNames reads the controller-supplied list of input
// artifact plugin names from the supervisor's environment.
func inputArtifactPluginNames() []wfv1.ArtifactPluginName {
	raw := common.SplitPluginNames(os.Getenv(common.EnvVarInputArtifactPluginNames))
	names := make([]wfv1.ArtifactPluginName, 0, len(raw))
	for _, p := range raw {
		names = append(names, wfv1.ArtifactPluginName(p))
	}
	return names
}

// writeReadyMarker signals to the emissary in main that pre-main setup is
// complete. We write to a tmp path then rename: rename is atomic relative to
// readers (the watcher inotify-rx event fires only after rename), so main
// never observes a partially-written file. We deliberately do not fsync —
// these markers live in an emptyDir and the pod is gone if the node crashes
// before the write hits disk.
func writeReadyMarker() error {
	return writeReadyMarkerAt(common.ReadyMarkerPath)
}

// writeReadyMarkerAt is the path-parameterized form used by tests; production
// calls writeReadyMarker with the constant.
func writeReadyMarkerAt(path string) error {
	return atomicWriteMarker(path, nil)
}

// writeFailedMarker is best-effort: if we can't write it, the emissary will
// continue waiting (inotify-based, with polling only as a fallback) for the
// ready marker and eventually hit the pod-level deadline.
// Same tmp-then-rename pattern as writeReadyMarker — see notes there.
func writeFailedMarker(ctx context.Context, cause error) {
	writeFailedMarkerAt(ctx, common.FailedMarkerPath, cause)
}

func writeFailedMarkerAt(ctx context.Context, path string, cause error) {
	if err := atomicWriteMarker(path, []byte(cause.Error())); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "failed to write failed marker")
	}
}

// atomicWriteMarker writes body to path via write-then-rename so a reader
// watching path (via inotify) only ever observes the fully-written file. See
// writeReadyMarker for why there is no fsync.
func atomicWriteMarker(path string, body []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return fmt.Errorf("failed to write marker tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename marker %s: %w", path, err)
	}
	return nil
}
