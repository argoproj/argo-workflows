package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoprojv1 "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// ResultSink is how one task's result reaches the controller. Every
// method is for the task the sink was built for.
type ResultSink interface {
	// Start records an empty placeholder result, so the controller knows
	// outputs are on their way.
	Start(ctx context.Context) error
	// Report merges r into the task's result: outputs, logs or a failure.
	Report(ctx context.Context, r wfv1.NodeResult) error
	// Progress records the task's progress. It retries only briefly: the
	// next progress report sends the latest progress again.
	Progress(ctx context.Context, progress wfv1.Progress) error
	// Fail attributes a failure to the task.
	Fail(ctx context.Context, err error)
	// Finalize marks the task's outputs as completely reported.
	Finalize(ctx context.Context) error
}

// taskResultSink reports through the task's WorkflowTaskResult, named by
// its node ID.
type taskResultSink struct {
	client      argoprojv1.WorkflowTaskResultInterface
	tracing     *tracing.Tracing
	workflow    string
	workflowUID types.UID
	nodeID      string
	instanceID  string

	resultRetry

	// writeTerminate writes the pod's termination message, which carries
	// the task's failure: the pod's process death is the task's death.
	writeTerminate func(msg string)

	// mu guards created: the deadline and progress goroutines report
	// alongside the task's own flow.
	mu      sync.Mutex
	created bool
}

var _ ResultSink = &taskResultSink{}

// NewTaskResultSink returns the sink for the pod's task cfg describes:
// Fail writes the termination message.
func (p *Process) NewTaskResultSink(cfg TaskConfig) ResultSink {
	return &taskResultSink{
		client:         p.taskResultClient,
		tracing:        p.Tracing,
		workflow:       cfg.WorkflowName,
		workflowUID:    cfg.WorkflowUID,
		nodeID:         cfg.NodeID,
		instanceID:     p.instanceID,
		writeTerminate: util.WriteTerminateMessage,

		resultRetry: p.resultRetry(),
	}
}

// resultRetry is how writes to a task's result retry: retryTimeout limits
// the retries of the final writes and interimRetryTimeout those of the
// writes before them; backoff spaces out the retries of both.
type resultRetry struct {
	retryTimeout        time.Duration
	interimRetryTimeout time.Duration
	backoff             wait.Backoff
}

func (p *Process) resultRetry() resultRetry {
	return resultRetry{
		retryTimeout:        p.taskResultRetryTimeout,
		interimRetryTimeout: p.interimResultRetryTimeout,
		backoff:             p.taskResultBackoff,
	}
}

func (s *taskResultSink) Start(ctx context.Context) error {
	return s.write(ctx, s.interimRetryTimeout, wfv1.NodeResult{})
}

func (s *taskResultSink) Report(ctx context.Context, result wfv1.NodeResult) error {
	return s.write(ctx, s.retryTimeout, result)
}

func (s *taskResultSink) Progress(ctx context.Context, progress wfv1.Progress) error {
	return s.write(ctx, s.interimRetryTimeout, wfv1.NodeResult{Progress: progress})
}

func (s *taskResultSink) Fail(_ context.Context, err error) {
	s.writeTerminate(err.Error())
}

// firstFailure records a task's first failure in its result, and ignores
// the rest, as a pool pod's worker's sinks do.
type firstFailure struct {
	// mu guards failed: the deadline and progress goroutines report
	// alongside the task's own flow.
	mu     sync.Mutex
	failed bool
}

// record reports err through report as the task's failure, with phase,
// unless a failure was recorded already.
func (f *firstFailure) record(ctx context.Context, report func(context.Context, wfv1.NodeResult) error, phase wfv1.NodePhase, err error) {
	f.mu.Lock()
	first := !f.failed
	f.failed = true
	f.mu.Unlock()
	if !first {
		return
	}
	if reportErr := report(ctx, wfv1.NodeResult{Phase: phase, Message: err.Error()}); reportErr != nil {
		logging.RequireLoggerFromContext(ctx).WithError(reportErr).Error(ctx, "failed to record task failure")
	}
}

func (s *taskResultSink) Finalize(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	var count uint64
	return s.retry(ctx, s.retryTimeout, func(ctx context.Context) error {
		err := s.patchLabels(ctx, map[string]string{
			common.LabelKeyReportOutputsCompleted: "true",
		})
		if apierr.IsForbidden(err) || apierr.IsNotFound(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
}

// write merges result into the task result, retrying transient errors for
// up to limit.
func (s *taskResultSink) write(ctx context.Context, limit time.Duration, result wfv1.NodeResult) error {
	var count uint64 // used to avoid spamming with these messages
	logger := logging.RequireLoggerFromContext(ctx)
	return s.retry(ctx, limit, func(ctx context.Context) error {
		err := s.upsert(ctx, result)
		if apierr.IsForbidden(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
}

// retry retries op, a write to a task's result (retryDoing).
func (s resultRetry) retry(ctx context.Context, limit time.Duration, op func(ctx context.Context) error) error {
	return s.retryDoing(ctx, limit, "writing task result", op)
}

// retryDoing calls op until it succeeds, and returns nil, or fails with an
// error that isn't transient, and returns that error. op's ctx ends once
// limit has passed or ctx is done, cutting off an attempt in flight; it
// then returns the last transient error, or the ctx error if there was
// none, wrapped to say it gave up doing what op does if it was limit that
// ran out.
func (s resultRetry) retryDoing(ctx context.Context, limit time.Duration, doing string, op func(ctx context.Context) error) error {
	retryCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	var lastErr error
	err := s.backoff.DelayFunc().Until(retryCtx, true, true, func(ctx context.Context) (bool, error) {
		err := op(ctx)
		switch {
		case err == nil:
			return true, nil
		case ctx.Err() != nil:
			return false, ctx.Err()
		case errorsutil.IsTransientErr(ctx, err):
			lastErr = err
			return false, nil
		default:
			return false, err
		}
	})
	if err == nil || retryCtx.Err() == nil {
		return err
	}
	if lastErr != nil {
		err = lastErr
	}
	if ctx.Err() == nil {
		return fmt.Errorf("gave up %s after %s: %w", doing, limit, err)
	}
	return err
}

func (s *taskResultSink) upsert(ctx context.Context, result wfv1.NodeResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.created {
		err := s.create(ctx, result)
		if apierr.IsAlreadyExists(err) {
			return s.patch(ctx, result)
		}
		if err != nil {
			return err
		}
	} else {
		err := s.patch(ctx, result)
		if err != nil {
			if apierr.IsNotFound(err) {
				return s.create(ctx, result)
			}
			return err
		}
	}
	return nil
}

func (s *taskResultSink) patch(ctx context.Context, result wfv1.NodeResult) error {
	ctx, span := s.tracing.StartPatchTaskResult(ctx)
	defer span.End()
	data, err := json.Marshal(&wfv1.WorkflowTaskResult{NodeResult: result})
	if err != nil {
		return err
	}
	_, err = s.client.Patch(ctx,
		s.nodeID,
		types.MergePatchType,
		data,
		metav1.PatchOptions{},
	)
	return err
}

func (s *taskResultSink) patchLabels(ctx context.Context, labels map[string]string) error {
	ctx, span := s.tracing.StartPatchTaskResultLabels(ctx)
	defer span.End()
	data, err := json.Marshal(&wfv1.WorkflowTaskResult{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
	})
	if err != nil {
		return err
	}
	_, err = s.client.Patch(ctx,
		s.nodeID,
		types.MergePatchType,
		data,
		metav1.PatchOptions{},
	)
	return err
}

func (s *taskResultSink) create(ctx context.Context, result wfv1.NodeResult) error {
	ctx, span := s.tracing.StartCreateTaskResult(ctx)
	defer span.End()
	taskResult := common.NewTaskResult("", s.nodeID, s.workflow, s.workflowUID, s.instanceID)
	taskResult.NodeResult = result
	_, err := s.client.Create(ctx,
		taskResult,
		metav1.CreateOptions{},
	)
	s.created = err == nil || apierr.IsAlreadyExists(err)
	return err
}
