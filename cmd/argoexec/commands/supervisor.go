package commands

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// Status marker protocol between the supervisor (writer) and the emissary in
// main (reader), exchanged via common.StatusMarkerPath in the shared emptyDir.
// The first line of the marker is one of these tokens; for statusFailed the
// remaining lines carry the failure message.
const (
	statusRunning = "RUNNING" // pre-main in progress; rewritten as a heartbeat
	statusReady   = "READY"   // pre-main succeeded; main may proceed
	statusFailed  = "FAILED"  // pre-main failed; message follows on later lines

	// supervisorHeartbeatInterval is how often the supervisor rewrites the
	// status marker while pre-main runs. Each rewrite advances the marker's
	// mtime, which is what proves liveness to main.
	supervisorHeartbeatInterval = 5 * time.Second
	// supervisorHeartbeatTimeout is how long main tolerates a status marker that
	// has neither appeared nor advanced before presuming the supervisor dead.
	// Generously larger than the interval so a GC pause or CPU starvation in the
	// supervisor doesn't trigger a false-positive death.
	supervisorHeartbeatTimeout = 30 * time.Second
	// supervisorStatusPollInterval is how often main re-checks the marker's
	// freshness. inotify delivers the terminal READY/FAILED write promptly; this
	// poll exists only to notice the *absence* of heartbeats, which inotify can't.
	supervisorStatusPollInterval = 2 * time.Second
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
			// Heartbeat the status marker so main's emissary can tell a live (but
			// slow) supervisor from a dead one, rather than blocking to the pod
			// deadline. The initial write also overwrites any stale marker from a
			// prior attempt. Stop the heartbeat before the terminal write so no
			// heartbeat races it on the same path.
			stopHeartbeat, hbErr := startStatusHeartbeat(ctx)
			if hbErr != nil {
				// We can't write the marker at all (e.g. broken mount), so main
				// can't be signalled — it will presume us dead via the
				// never-appeared timeout. Record the error and collect logs.
				we.AddError(ctx, hbErr)
				return we.PostMain(ctx, bgCtx, true)
			}

			preMainErr := supervisorPreMain(ctx, we)
			stopHeartbeat()

			preMainFailed := false
			if preMainErr != nil {
				// Surface the error to the emissary in main via the status marker,
				// then fall through to PostMain. PostMain waits for main to exit —
				// emissary in main reads the failure status, exits 65, and writes
				// its exitcode file — and then captures main's stdout/stderr (which
				// includes the supervisor failure message echoed by emissary) into
				// the task result. We must NOT write a success status here.
				writeFailureStatus(ctx, preMainErr)
				we.AddError(ctx, preMainErr)
				preMainFailed = true
			} else if err := writeSuccessStatus(); err != nil {
				// The success status write failed (e.g. disk full). Without a
				// terminal status, main's emissary would keep waiting until the
				// heartbeat-staleness timeout. Write a failure status so main
				// exits promptly, then fall through to PostMain (logs only).
				writeFailureStatus(ctx, err)
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

	// A stale marker from a prior attempt needs no explicit cleanup: the
	// heartbeat's initial RUNNING write (startStatusHeartbeat, before we get
	// here) has already overwritten it with a fresh mtime.
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

// startStatusHeartbeat writes an initial RUNNING status, then rewrites it every
// supervisorHeartbeatInterval on a background goroutine until the returned stop
// function is called. Each rewrite advances the marker's mtime, which main's
// emissary uses to distinguish a live (but slow) supervisor from a dead one.
//
// The initial write is synchronous so a broken shared mount fails fast (its
// error is returned). stop() cancels the goroutine and blocks until it has
// exited, guaranteeing no heartbeat write can race the terminal status write
// that follows it.
func startStatusHeartbeat(ctx context.Context) (stop func(), err error) {
	if err := writeRunningStatus(); err != nil {
		return nil, fmt.Errorf("failed to write initial status marker: %w", err)
	}
	hbCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(supervisorHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				if err := writeRunningStatus(); err != nil {
					logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "failed to write status heartbeat")
				}
			}
		}
	}()
	return func() { cancel(); <-done }, nil
}

// writeRunningStatus rewrites the status marker as RUNNING (the heartbeat).
func writeRunningStatus() error {
	return writeRunningStatusAt(common.StatusMarkerPath)
}

func writeRunningStatusAt(path string) error {
	return writeStatusMarkerAt(path, []byte(statusRunning+"\n"))
}

// writeSuccessStatus signals to the emissary in main that pre-main setup is
// complete by writing the terminal READY token.
func writeSuccessStatus() error {
	return writeSuccessStatusAt(common.StatusMarkerPath)
}

func writeSuccessStatusAt(path string) error {
	return writeStatusMarkerAt(path, []byte(statusReady+"\n"))
}

// writeFailureStatus records a pre-main failure: the terminal FAILED token
// followed by the cause. Best-effort — if the write fails, main keeps waiting
// and eventually hits the heartbeat-staleness timeout.
func writeFailureStatus(ctx context.Context, cause error) {
	writeFailureStatusAt(ctx, common.StatusMarkerPath, cause)
}

func writeFailureStatusAt(ctx context.Context, path string, cause error) {
	body := statusFailed
	if msg := cause.Error(); msg != "" {
		body += "\n" + msg
	}
	if err := writeStatusMarkerAt(path, []byte(body)); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "failed to write failure status marker")
	}
}

// writeStatusMarkerAt writes body to path via write-then-rename so a reader
// watching path (via inotify) only ever observes the fully-written file — it
// never sees a torn terminal message. It is the path-parameterized form used by
// tests; production calls go through writeRunningStatus / writeSuccessStatus /
// writeFailureStatus with the constant. We deliberately do not fsync — the
// marker lives in an emptyDir and the pod is gone if the node crashes before
// the write hits disk.
func writeStatusMarkerAt(path string, body []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return fmt.Errorf("failed to write marker tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename marker %s: %w", path, err)
	}
	return nil
}
