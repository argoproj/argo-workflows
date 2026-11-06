package pooled

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"path"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// The delays a Source waits before trying again after a miss (a claim
// lost to another worker, a task gone, or a transient failure): random,
// up to minMissDelay doubled with each miss in a row, but no more than
// maxMissDelay. Until a worker misses, it claims at once.
const (
	defaultMinMissDelay = 100 * time.Millisecond
	defaultMaxMissDelay = 5 * time.Second
)

// Options configure a Source.
type Options struct {
	// Process is the worker's executor process: each task's sink is built
	// from it, and a claim the source completes or releases itself retries
	// with its task-result retries.
	Process *executor.Process
	// Paths says where a task's files live.
	Paths func(executor.TaskConfig) (executor.TaskPaths, error)
	// MinMissDelay and MaxMissDelay bound the random delay after a miss;
	// zero means the defaults.
	MinMissDelay time.Duration
	MaxMissDelay time.Duration
	// Ready, when set, is waited for before each claim, until it returns
	// nil: a pool pod's worker claims a task only while its main container
	// is up to run it.
	Ready func(context.Context) error
}

// Source yields the tasks a worker claims from its pool, one at a time,
// until it is stopped. Use Sink as the harness's sink builder, so each
// task reports through its claim.
type Source struct {
	consumer dispatch.Consumer
	process  *executor.Process
	paths    func(executor.TaskConfig) (executor.TaskPaths, error)
	ready    func(context.Context) error
	// backoff is the delays from the first miss in a row: minMissDelay,
	// doubled with each miss after it, up to maxMissDelay.
	backoff wait.Backoff

	// stopped ends on Stop; it is a context so Next's waits end with it.
	//nolint:containedctx
	stopped context.Context
	stop    context.CancelFunc

	mu sync.Mutex
	// missing is backoff as it stands after the misses in a row.
	missing wait.Backoff
	// limit bounds the random delay before the next claim: zero until the
	// worker misses.
	limit time.Duration
	// claims are the claims of the tasks yielded whose sink isn't built yet.
	claims map[string]dispatch.Claimed
}

var _ executor.TaskSource = &Source{}

// NewSource returns a source claiming through consumer, which needs its
// watch-backed sources (dispatch.NewWatches).
func NewSource(consumer dispatch.Consumer, opts Options) *Source {
	minDelay := opts.MinMissDelay
	if minDelay <= 0 {
		minDelay = defaultMinMissDelay
	}
	maxDelay := max(opts.MaxMissDelay, minDelay)
	if opts.MaxMissDelay <= 0 {
		maxDelay = max(defaultMaxMissDelay, minDelay)
	}
	stopped, stop := context.WithCancel(context.Background())
	backoff := wait.Backoff{Duration: minDelay, Factor: 2, Cap: maxDelay, Steps: math.MaxInt}
	return &Source{
		consumer: consumer,
		process:  opts.Process,
		paths:    opts.Paths,
		ready:    opts.Ready,
		backoff:  backoff,
		missing:  backoff,
		stopped:  stopped,
		stop:     stop,
		claims:   map[string]dispatch.Claimed{},
	}
}

// Stop makes the source stop taking work: from now on Next returns
// ok=false, including a Next waiting for a task. A task already yielded is
// the caller's to finish or release. Stop is how a worker stops taking work
// on SIGTERM while running its harness on a context SIGTERM doesn't cancel,
// so the task in hand isn't cancelled with it.
func (s *Source) Stop() {
	s.stop()
}

func (s *Source) isStopped() bool {
	return s.stopped.Err() != nil
}

// Next claims the oldest task it can and returns it, waiting until there is
// one. Claims lost to other workers, and tasks gone by the time they are
// claimed (the claim is then withdrawn), are skipped. A task whose TaskSet
// can't be run, such as one without spec.dispatch, is completed as Error.
// It returns ok=false once the source is stopped, and ctx's error if ctx
// ends first; a claim made as it is stopped is released. A claim it can
// neither complete nor release is left incomplete under the worker, so
// Next returns an error: only the worker's restart, which fails the claims
// it holds, settles it.
func (s *Source) Next(ctx context.Context) (executor.Task, bool, error) {
	// Waits end on Stop; claims don't, so a claim is never cut off half made.
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stopAfter := context.AfterFunc(s.stopped, cancel) //nolint:contextcheck // the source's own lifetime, which ends the waits on Stop
	defer stopAfter()
	logger := logging.RequireLoggerFromContext(ctx)
	for {
		if s.isStopped() {
			return executor.Task{}, false, nil
		}
		if err := ctx.Err(); err != nil {
			return executor.Task{}, false, err
		}
		candidates, err := s.consumer.Candidates(waitCtx)
		if err != nil {
			if waitCtx.Err() != nil {
				continue
			}
			if !errorsutil.IsTransientErr(ctx, err) {
				return executor.Task{}, false, err
			}
			s.miss()
			s.pause(waitCtx)
			continue
		}
		for _, candidate := range candidates {
			if !s.pause(waitCtx) || s.isStopped() {
				break
			}
			if s.ready != nil {
				if err := s.ready(waitCtx); err != nil {
					if waitCtx.Err() != nil {
						break
					}
					return executor.Task{}, false, err
				}
			}
			claimed, err := s.consumer.Claim(ctx, candidate)
			if errors.Is(err, dispatch.ErrClaimLost) || errors.Is(err, dispatch.ErrTaskGone) {
				logger.WithFields(logging.Fields{"namespace": candidate.Namespace, "taskSet": candidate.TaskSetName()}).
					WithError(err).Debug(ctx, "Missed a pooled task")
				s.miss()
				continue
			}
			if err != nil {
				if !errorsutil.IsTransientErr(ctx, err) {
					return executor.Task{}, false, err
				}
				// Find the candidates afresh.
				s.miss()
				break
			}
			s.hit()
			if s.isStopped() {
				return executor.Task{}, false, s.release(ctx, claimed)
			}
			task, err := s.task(claimed)
			if err != nil {
				if failErr := s.fail(ctx, claimed, err); failErr != nil {
					return executor.Task{}, false, failErr
				}
				continue
			}
			s.mu.Lock()
			s.claims[task.Config.NodeID] = claimed
			s.mu.Unlock()
			logger.WithFields(logging.Fields{"namespace": candidate.Namespace, "nodeID": task.Config.NodeID, "templateName": task.Config.Template.Name}).
				Info(ctx, "Claimed a pooled task")
			return task, true, nil
		}
	}
}

// Sink returns the harness's sink builder: each task the source yielded
// reports through its claim, with the process's retries. The sink is a
// *executor.PooledSink, whose Stopped says when the task must stop.
func (s *Source) Sink() func(executor.Task) executor.ResultSink {
	return func(task executor.Task) executor.ResultSink {
		s.mu.Lock()
		claimed, ok := s.claims[task.Config.NodeID]
		delete(s.claims, task.Config.NodeID)
		s.mu.Unlock()
		if !ok {
			panic(fmt.Sprintf("no claim on task %s: it didn't come from this source", task.Config.NodeID))
		}
		return s.process.NewPooledSink(claimed)
	}
}

func (s *Source) task(claimed dispatch.Claimed) (executor.Task, error) {
	cfg, err := TaskConfig(claimed.TaskSet(), claimed.Claim())
	if err != nil {
		return executor.Task{}, err
	}
	paths, err := s.paths(cfg)
	if err != nil {
		return executor.Task{}, err
	}
	// In the task's own shared directory, as a pod's is in its only one:
	// tasks run one after another must not share it.
	if cfg.ProgressFile != "" {
		cfg.ProgressFile = filepath.Join(paths.VarRunArgo, path.Base(common.ArgoProgressPath))
	}
	return executor.Task{Config: cfg, Paths: paths}, nil
}

// fail completes a claimed task that can't be run as Error, so its node
// fails rather than every worker claiming it in turn. It returns an error
// if the claim is left incomplete.
func (s *Source) fail(ctx context.Context, claimed dispatch.Claimed, cause error) error {
	logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"namespace": claimed.Claim().Namespace, "nodeID": claimed.Claim().NodeID}).
		WithError(cause).Error(ctx, "Cannot run a pooled task")
	sink := s.process.NewPooledSink(claimed)
	sink.Fail(ctx, cause)
	if err := executor.UnlessClaimLost(ctx, sink.Finalize(ctx), "The claim of a task that can't be run was lost"); err != nil {
		return fmt.Errorf("failed to fail pooled task %s, which cannot be run: %w", claimed.Claim().NodeID, err)
	}
	return nil
}

// release gives back a task claimed as the source was stopped. It returns
// an error if the claim is left incomplete.
func (s *Source) release(ctx context.Context, claimed dispatch.Claimed) error {
	ctx = context.WithoutCancel(ctx)
	if err := executor.UnlessClaimLost(ctx, s.process.NewPooledSink(claimed).Release(ctx), "The claim of a task claimed while stopping was lost"); err != nil {
		return fmt.Errorf("failed to release pooled task %s, claimed while stopping: %w", claimed.Claim().NodeID, err)
	}
	return nil
}

func (s *Source) miss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limit = s.missing.Step()
}

func (s *Source) hit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.missing, s.limit = s.backoff, 0
}

// pause waits a random delay if the worker has been missing, so workers
// racing for the same tasks spread out. It returns false if ctx ended
// first.
func (s *Source) pause(ctx context.Context) bool {
	s.mu.Lock()
	limit := s.limit
	s.mu.Unlock()
	if limit == 0 {
		return ctx.Err() == nil
	}
	// Full jitter, over the whole of the limit.
	t := time.NewTimer(rand.N(limit) + 1)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
