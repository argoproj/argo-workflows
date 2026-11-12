package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
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
	// Report merges r into the task's final result: outputs, logs or a
	// failure. Finalize writes it, so a task's final result and its
	// completion reach the API server in one write; once Finalize has
	// run, Report writes r straight away.
	Report(ctx context.Context, r wfv1.NodeResult) error
	// Send writes r to the task's result straight away, leaving it
	// incomplete. It is for a report that another process finalizes.
	Send(ctx context.Context, r wfv1.NodeResult) error
	// Progress records the task's progress. It retries only briefly: the
	// next progress report sends the latest progress again.
	Progress(ctx context.Context, progress wfv1.Progress) error
	// Fail attributes a failure to the task.
	Fail(ctx context.Context, err error)
	// Finalize writes the reported result and marks the task's outputs as
	// completely reported, in one write.
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

	// retryTimeout limits the retries of the final writes and
	// interimRetryTimeout those of the placeholder and progress writes
	// before them; backoff spaces out the retries of both.
	retryTimeout        time.Duration
	interimRetryTimeout time.Duration
	backoff             wait.Backoff

	// recordFailure selects how Fail attributes a failure. In a pod the
	// termination log carries it (process death is task death); a worker
	// running many tasks records it in the task result instead, keeping
	// the first failure.
	recordFailure bool
	// writeTerminate writes the pod's termination message.
	writeTerminate func(msg string)

	// mu guards the fields below: the deadline and progress goroutines
	// report alongside the task's own flow.
	mu      sync.Mutex
	failed  bool
	created bool
	// final is the result Report has gathered for Finalize to write, nil
	// if nothing was reported.
	final *wfv1.NodeResult
	// finalized is set once Finalize has run, after which Report writes
	// straight away.
	finalized bool
}

var _ ResultSink = &taskResultSink{}

// NewTaskResultSink returns the sink for the task cfg describes. With
// recordFailure false (a pod), Fail writes the termination message; with
// recordFailure true (a worker), Fail records the failure in the task's
// result.
func (p *Process) NewTaskResultSink(cfg TaskConfig, recordFailure bool) ResultSink {
	return &taskResultSink{
		client:         p.taskResultClient,
		tracing:        p.Tracing,
		workflow:       cfg.WorkflowName,
		workflowUID:    cfg.WorkflowUID,
		nodeID:         cfg.NodeID,
		instanceID:     p.instanceID,
		recordFailure:  recordFailure,
		writeTerminate: util.WriteTerminateMessage,

		retryTimeout:        p.taskResultRetryTimeout,
		interimRetryTimeout: p.interimResultRetryTimeout,
		backoff:             p.taskResultBackoff,
	}
}

func (s *taskResultSink) Start(ctx context.Context) error {
	return s.write(ctx, s.interimRetryTimeout, wfv1.NodeResult{})
}

func (s *taskResultSink) Report(ctx context.Context, result wfv1.NodeResult) error {
	s.mu.Lock()
	if s.finalized {
		s.mu.Unlock()
		return s.write(ctx, s.retryTimeout, result)
	}
	defer s.mu.Unlock()
	if s.final == nil {
		s.final = result.DeepCopy()
		return nil
	}
	merged, err := mergeResults(*s.final, result)
	if err != nil {
		return err
	}
	s.final = merged
	return nil
}

func (s *taskResultSink) Send(ctx context.Context, result wfv1.NodeResult) error {
	return s.write(ctx, s.retryTimeout, result)
}

func (s *taskResultSink) Progress(ctx context.Context, progress wfv1.Progress) error {
	return s.write(ctx, s.interimRetryTimeout, wfv1.NodeResult{Progress: progress})
}

func (s *taskResultSink) Fail(ctx context.Context, err error) {
	if !s.recordFailure {
		s.writeTerminate(err.Error())
		return
	}
	s.mu.Lock()
	first := !s.failed
	s.failed = true
	s.mu.Unlock()
	if !first {
		return
	}
	if reportErr := s.Report(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: err.Error()}); reportErr != nil {
		logging.RequireLoggerFromContext(ctx).WithError(reportErr).Error(ctx, "failed to record task failure")
	}
}

// Finalize writes the reported result together with the completed label,
// or only the label if nothing was reported. If the API server rejects the
// result itself, for instance as too large, it still marks the result
// complete without it, so the controller isn't left waiting for outputs
// that will never come, and returns the rejection.
func (s *taskResultSink) Finalize(ctx context.Context) error {
	s.mu.Lock()
	final := s.final
	s.final = nil
	s.finalized = true
	s.mu.Unlock()

	if final == nil {
		return s.writeCompleted(ctx, func(ctx context.Context) error {
			return s.patch(ctx, wfv1.NodeResult{}, outputsCompletedLabels())
		})
	}
	rejected := false
	err := s.writeCompleted(ctx, func(ctx context.Context) error {
		err := s.upsert(ctx, *final, outputsCompletedLabels())
		rejected = err != nil && ctx.Err() == nil && !apierr.IsForbidden(err) && !errorsutil.IsTransientErr(ctx, err)
		return err
	})
	if !rejected {
		return err
	}
	logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "task result rejected, marking it complete without it")
	fallback := wfv1.NodeResult{}
	if s.recordFailure {
		fallback.Phase = wfv1.NodeFailed
		fallback.Message = fmt.Sprintf("failed to report outputs: %v", err)
		if final.Phase == wfv1.NodeFailed {
			fallback.Message = final.Message
		}
	}
	if fallbackErr := s.writeCompleted(ctx, func(ctx context.Context) error {
		return s.upsert(ctx, fallback, outputsCompletedLabels())
	}); fallbackErr != nil {
		return fallbackErr
	}
	return err
}

// writeCompleted retries op, a write that marks the result complete.
func (s *taskResultSink) writeCompleted(ctx context.Context, op func(ctx context.Context) error) error {
	logger := logging.RequireLoggerFromContext(ctx)
	var count uint64
	return s.retry(ctx, s.retryTimeout, func(ctx context.Context) error {
		err := op(ctx)
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
		err := s.upsert(ctx, result, nil)
		if apierr.IsForbidden(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
}

// retry calls op until it succeeds, and returns nil, or fails with an
// error that isn't transient, and returns that error. op's ctx ends once
// limit has passed or ctx is done, cutting off an attempt in flight; retry
// then returns the last transient error, or the ctx error if there was
// none, wrapped to say it gave up if it was limit that ran out.
func (s *taskResultSink) retry(ctx context.Context, limit time.Duration, op func(ctx context.Context) error) error {
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
		return fmt.Errorf("gave up writing task result after %s: %w", limit, err)
	}
	return err
}

// upsert writes result to the task result, creating it if needed. labels
// are merged into the task result's labels in the same write.
func (s *taskResultSink) upsert(ctx context.Context, result wfv1.NodeResult, labels map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.created {
		err := s.create(ctx, result, labels)
		if apierr.IsAlreadyExists(err) {
			return s.patch(ctx, result, labels)
		}
		if err != nil {
			return err
		}
	} else {
		err := s.patch(ctx, result, labels)
		if err != nil {
			if apierr.IsNotFound(err) {
				return s.create(ctx, result, labels)
			}
			return err
		}
	}
	return nil
}

func (s *taskResultSink) patch(ctx context.Context, result wfv1.NodeResult, labels map[string]string) error {
	ctx, span := s.tracing.StartPatchTaskResult(ctx, marksCompleted(labels))
	defer span.End()
	data, err := json.Marshal(&wfv1.WorkflowTaskResult{
		ObjectMeta: metav1.ObjectMeta{Labels: labels},
		NodeResult: result,
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

func (s *taskResultSink) create(ctx context.Context, result wfv1.NodeResult, labels map[string]string) error {
	ctx, span := s.tracing.StartCreateTaskResult(ctx, marksCompleted(labels))
	defer span.End()
	taskResult := &wfv1.WorkflowTaskResult{
		TypeMeta: metav1.TypeMeta{
			APIVersion: workflow.APIVersion,
			Kind:       workflow.WorkflowTaskResultKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: s.nodeID,
		},
		NodeResult: result,
	}
	taskResult.SetLabels(map[string]string{
		common.LabelKeyWorkflow:               s.workflow,
		common.LabelKeyReportOutputsCompleted: "false",
	})
	maps.Copy(taskResult.Labels, labels)
	taskResult.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: workflow.APIVersion,
		Kind:       workflow.WorkflowKind,
		Name:       s.workflow,
		UID:        s.workflowUID,
	}})

	if s.instanceID != "" {
		taskResult.Labels[common.LabelKeyControllerInstanceID] = s.instanceID
	}
	_, err := s.client.Create(ctx,
		taskResult,
		metav1.CreateOptions{},
	)
	s.created = err == nil || apierr.IsAlreadyExists(err)
	return err
}

// mergeResults returns b merged into a the way a JSON merge patch of b
// would merge it into a task result holding a, so gathering reports for
// one write ends as writing each in turn would have.
func mergeResults(a, b wfv1.NodeResult) (*wfv1.NodeResult, error) {
	doc, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	patch, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	merged, err := jsonpatch.MergePatch(doc, patch)
	if err != nil {
		return nil, err
	}
	var result wfv1.NodeResult
	if err := json.Unmarshal(merged, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// outputsCompletedLabels tells the controller that every output has been
// reported to the task result.
func outputsCompletedLabels() map[string]string {
	return map[string]string{common.LabelKeyReportOutputsCompleted: "true"}
}

// marksCompleted says whether a write carrying labels marks the task
// result complete.
func marksCompleted(labels map[string]string) bool {
	return labels[common.LabelKeyReportOutputsCompleted] == "true"
}
