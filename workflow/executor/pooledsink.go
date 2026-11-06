package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	jsonpatch "github.com/evanphx/json-patch"
	apierr "k8s.io/apimachinery/pkg/api/errors"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	kubeerrors "github.com/argoproj/argo-workflows/v4/util/kube/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// PooledSink reports a pooled task through the result its claim created.
// The claim's result is written once, by Finalize, with everything
// reported before it; every write is fenced by the claim, so once the
// claim is lost (its result deleted or replaced) writes fail with
// dispatch.ErrClaimLost and the result is never recreated.
//
// Stopped and StopReason tell the role running the task when the task must
// stop from outside the process: dispatch.StopInterrupted when the
// controller ended the node (stop the task, then Finalize as usual; the
// controller ignores the outcome), dispatch.StopLost when the claim was
// lost (stop the task and report nothing).
type PooledSink struct {
	resultRetry
	tracing *tracing.Tracing
	claim   dispatch.Claimed
	failure firstFailure

	// mu guards result: the deadline and progress goroutines report
	// alongside the task's own flow.
	mu     sync.Mutex
	result wfv1.NodeResult
}

var _ ResultSink = &PooledSink{}

// NewPooledSink returns the sink for the task claim holds, with the
// process's retries.
func (p *Process) NewPooledSink(claim dispatch.Claimed) *PooledSink {
	return &PooledSink{resultRetry: p.resultRetry(), tracing: p.Tracing, claim: claim}
}

// Start does nothing: the claim created the result.
func (s *PooledSink) Start(context.Context) error { return nil }

// Report merges r into what Finalize writes, as a pod's sink merges it
// into its result.
func (s *PooledSink) Report(_ context.Context, r wfv1.NodeResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	merged, err := mergeResult(s.result, r)
	if err != nil {
		return err
	}
	s.result = merged
	return nil
}

func (s *PooledSink) Progress(ctx context.Context, progress wfv1.Progress) error {
	return s.retry(ctx, s.interimRetryTimeout, func(ctx context.Context) error {
		ctx, span := s.tracing.StartPatchTaskResult(ctx)
		defer span.End()
		return s.claim.Progress(ctx, progress)
	})
}

// Fail records the task's first failure, as a worker's task result sink
// does. A failure of the task's own, such as main exiting non-zero, makes it
// Failed; any other, the worker's, makes it Error, as a pod's init, wait or
// supervisor container failing does.
func (s *PooledSink) Fail(ctx context.Context, err error) {
	s.failure.record(ctx, s.Report, poolTaskFailurePhase(err), err)
}

// Finalize completes the claim's result with everything reported, in one
// write. A task that reported no failure succeeded. A result the API server
// refuses as too large or invalid can't be reported however often it is
// tried, so the claim is completed as Error instead, saying why, with none
// of the task's outputs: the node fails visibly, and nothing is truncated.
func (s *PooledSink) Finalize(ctx context.Context) error {
	s.mu.Lock()
	result := *s.result.DeepCopy()
	s.mu.Unlock()
	if result.Phase == "" {
		result.Phase = wfv1.NodeSucceeded
	}
	err := s.complete(ctx, result)
	if !unreportable(err) {
		return err
	}
	logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "The task's result can't be reported, so its claim is completed as Error without it")
	return s.complete(ctx, wfv1.NodeResult{
		Phase:    wfv1.NodeError,
		Message:  fmt.Sprintf("failed to report the task's result: %v", err),
		Progress: result.Progress,
	})
}

func (s *PooledSink) complete(ctx context.Context, result wfv1.NodeResult) error {
	return s.retry(ctx, s.retryTimeout, func(ctx context.Context) error {
		ctx, span := s.tracing.StartPatchTaskResult(ctx)
		defer span.End()
		return s.claim.Complete(ctx, result)
	})
}

// unreportable reports whether err, from completing a claim, means its
// result can never be written: the API server refused it as too large or
// invalid.
func unreportable(err error) bool {
	return err != nil && !errors.Is(err, dispatch.ErrClaimLost) &&
		(kubeerrors.IsRequestEntityTooLargeErr(err) || apierr.IsInvalid(err))
}

// Release gives the task back, deleting the claim so that another worker
// can claim it, without reporting anything. A claim already lost is
// dispatch.ErrClaimLost.
func (s *PooledSink) Release(ctx context.Context) error {
	return s.retry(ctx, s.interimRetryTimeout, func(ctx context.Context) error {
		return s.claim.Release(ctx)
	})
}

// Stopped is closed when the task must stop; StopReason then says why.
func (s *PooledSink) Stopped() <-chan struct{} { return s.claim.Stopped() }

// StopReason is the first reason the task had to stop, or empty.
func (s *PooledSink) StopReason() dispatch.StopReason { return s.claim.StopReason() }

// mergeResult applies r to into as a JSON merge patch, as the API server
// applies a pod's reports to its result.
func mergeResult(into, r wfv1.NodeResult) (wfv1.NodeResult, error) {
	original, err := json.Marshal(into)
	if err != nil {
		return into, err
	}
	patch, err := json.Marshal(r)
	if err != nil {
		return into, err
	}
	data, err := jsonpatch.MergePatch(original, patch)
	if err != nil {
		return into, fmt.Errorf("failed to merge task result: %w", err)
	}
	var merged wfv1.NodeResult
	if err := json.Unmarshal(data, &merged); err != nil {
		return into, err
	}
	return merged, nil
}

// UnlessClaimLost returns err, unless it says the task's claim was lost
// (dispatch.ErrClaimLost), when it logs that with why and returns nil: the
// task's node has moved on, so completing or releasing the claim is no
// longer its worker's to do.
func UnlessClaimLost(ctx context.Context, err error, why string) error {
	if !errors.Is(err, dispatch.ErrClaimLost) {
		return err
	}
	logging.RequireLoggerFromContext(ctx).WithError(err).Info(ctx, why)
	return nil
}
