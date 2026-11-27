package executor

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/pooled"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// Worker is `argoexec worker`, the worker container of a hot worker pool
// pod: it claims the pool's tasks one at a time and runs each through
// executor.PoolRole, which hands it to the pool pod's looping main
// container, until SIGTERM. Then it stops taking work, terminates its
// process, which releases the task in hand unless main finishes it before
// the release is due, and exits once the task is over.
type Worker struct {
	// Process is the worker's executor process. It must be built with
	// Stop as its ProcessConfig.StopWorker.
	Process *executor.Process
	// Consumer claims the pool's tasks.
	Consumer dispatch.Consumer
	// CheckAccess, if set, fails if the worker isn't allowed to find and
	// claim its pool's tasks (dispatch.CheckAccess).
	CheckAccess func(ctx context.Context) error
	// RunWatches, if set, starts the watches Consumer finds its tasks
	// through, until ctx ends.
	RunWatches func(ctx context.Context)
	// Base is the pool pod's shared directories: executor.PodPaths() in a
	// pool pod. The pool directory and each task's directories are beneath
	// them.
	Base executor.TaskPaths
	// PodName is the worker's pod.
	PodName string

	mu sync.Mutex
	// stopped is why a task stopped the worker, if one did.
	stopped error
	src     *pooled.Source
}

// RunWorker is `argoexec worker`'s composition root: the worker's process,
// its pool's watches and consumer, all from the worker container's
// environment, then Worker.Run.
func RunWorker(ctx context.Context, clientConfig clientcmd.ClientConfig) error {
	podUID := types.UID(os.Getenv(common.EnvVarPodUID))
	poolHash := os.Getenv(common.EnvVarPoolHash)
	if podUID == "" || poolHash == "" {
		return fmt.Errorf("a worker needs its pod's UID and pool hash, from %s and %s", common.EnvVarPodUID, common.EnvVarPoolHash)
	}
	reserve, err := ReleaseReserve()
	if err != nil {
		return err
	}
	w := &Worker{Base: executor.PodPaths()}
	process, config, client, namespace := initProcess(ctx, clientConfig, func(cfg *executor.ProcessConfig) {
		cfg.StopWorker = w.Stop
		cfg.ReleaseReserve = reserve
	})
	stopProcess := startProcess(ctx, process)
	defer stopProcess()
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return err
	}
	worker := dispatch.Worker{Namespace: namespace, PoolHash: poolHash, PodName: process.PodName, PodUID: podUID}
	watches := dispatch.NewWatches(client, metadataClient, worker)
	w.Process = process
	w.Consumer = dispatch.NewConsumer(client, worker, watches.Sources())
	w.CheckAccess = func(ctx context.Context) error { return dispatch.CheckAccess(ctx, process.ClientSet, worker) }
	w.RunWatches = watches.Run
	w.PodName = process.PodName
	return w.Run(ctx)
}

// Run runs the worker until ctx ends, which is SIGTERM, and the task in
// hand, if any, is over. First it ends what an earlier run of the worker
// container left (executor.Process.RecoverPool), so the task that run had
// is never run again. It returns an error if the worker can't go on,
// including when a task stopped it, such as when its result couldn't be
// reported: the worker container then restarts, and its next run fails the
// task's claim. Failed tasks aren't the worker's failure: each is reported
// through its claim. A worker that isn't allowed to find and claim its
// pool's tasks returns an error saying so, and crash loops until it is.
func (w *Worker) Run(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx).WithField("workerPod", w.PodName)
	ctx = logging.WithLogger(ctx, logger)
	// Tasks run on a context SIGTERM doesn't cancel: SIGTERM stops the
	// source and terminates the process, whose task in hand decides itself
	// whether to finish or be released.
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	// main may run as another user, and reads and writes what the worker
	// stages for it, as in an init-less pod.
	osspecific.AllowGrantingAccessToEveryone()
	pool := poolpod.NewPool(w.Base.PoolDir())
	if err := pool.Init(); err != nil {
		return fmt.Errorf("failed to create the pool directory: %w", err)
	}
	recoverErr := w.Process.RecoverPool(runCtx, pool, w.Consumer, common.WorkerRestartedMessage(w.PodName))
	// Checked after recovery, which settles main's task without the API, and
	// even if recovery failed, which a worker without its rights does too,
	// with a less helpful error.
	if w.CheckAccess != nil {
		if err := w.CheckAccess(runCtx); err != nil {
			return err
		}
	}
	if recoverErr != nil {
		return recoverErr
	}
	// What an earlier run failed to remove after its task. If it still
	// can't be removed, the worker exits again, rather than run tasks that
	// could see it: the pod crash loops, and the pool shows as degraded.
	if err := executor.ClearPool(w.Base); err != nil {
		return fmt.Errorf("failed to clear the pool: %w", err)
	}
	if w.RunWatches != nil {
		w.RunWatches(runCtx)
	}
	// A main that can't start, such as when its image can't be pulled,
	// leaves the pool's tasks queued, where their nodes say why, rather
	// than claimed by a worker that can never run them.
	ready := func(ctx context.Context) error {
		if up, err := pool.MainUp(); err != nil || up {
			return err
		}
		logger.Info(ctx, "Waiting for main to start before taking a task")
		return pool.WaitMainUp(ctx)
	}
	src := pooled.NewSource(w.Consumer, pooled.Options{Process: w.Process, Paths: poolTaskPaths(w.Base), Ready: ready})
	w.mu.Lock()
	w.src = src
	if w.stopped != nil {
		src.Stop()
	}
	w.mu.Unlock()
	stopOnTerm := context.AfterFunc(ctx, func() {
		logger.Info(ctx, "Terminated: taking no more tasks, and releasing the task in hand unless it ends first")
		w.Process.Terminate()
		src.Stop()
	})
	defer stopOnTerm()
	logger.Info(ctx, "Taking the pool's tasks")
	source := &recordingSource{TaskSource: src}
	h := &executor.Harness{Process: w.Process, Source: source, Sink: src.Sink(), Role: executor.PoolRole}
	err := h.Run(runCtx)
	w.mu.Lock()
	stopped := w.stopped
	w.mu.Unlock()
	switch {
	case stopped != nil:
		return stopped
	case source.err != nil:
		return source.err
	case errors.Is(err, executor.ErrNoTask):
		logger.Info(ctx, "Terminated before taking a task")
	}
	return nil
}

// Stop is how a task stops the worker: no more tasks are taken, and Run
// returns cause once the task in hand is over. The first cause stands.
func (w *Worker) Stop(cause error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped == nil {
		w.stopped = fmt.Errorf("the worker stopped: %w", cause)
	}
	if w.src != nil {
		w.src.Stop()
	}
}

// recordingSource records the error its source's Next failed with, which
// the harness returns alongside its tasks' own.
type recordingSource struct {
	executor.TaskSource
	err error
}

func (s *recordingSource) Next(ctx context.Context) (executor.Task, bool, error) {
	task, ok, err := s.TaskSource.Next(ctx)
	if err != nil {
		s.err = err
	}
	return task, ok, err
}

// poolTaskPaths gives each task the worker claims directories of its own
// beneath base's, named after its node and made unique by a random suffix:
// main runs the task `current` names once `current` changes, so every task
// needs a new name, even a node's task claimed again after it was released.
func poolTaskPaths(base executor.TaskPaths) func(executor.TaskConfig) (executor.TaskPaths, error) {
	return func(cfg executor.TaskConfig) (executor.TaskPaths, error) {
		return base.PoolTask(fmt.Sprintf("%s-%08x", cfg.NodeID, rand.Uint32()))
	}
}
