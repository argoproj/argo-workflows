package executor

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	retryutil "k8s.io/client-go/util/retry"

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
	// Report merges r into the task's result: outputs, logs or progress.
	Report(ctx context.Context, r wfv1.NodeResult) error
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

	// recordFailure selects how Fail attributes a failure. In a pod the
	// termination log carries it (process death is task death); a worker
	// running many tasks records it in the task result instead, keeping
	// the first failure.
	recordFailure bool
	// writeTerminate writes the pod's termination message.
	writeTerminate func(msg string)

	// mu guards failed and created: the deadline and progress goroutines
	// report alongside the task's own flow.
	mu      sync.Mutex
	failed  bool
	created bool
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
	}
}

func (s *taskResultSink) Start(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	var count uint64
	return retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
		err := s.upsert(ctx, wfv1.NodeResult{})
		if apierr.IsForbidden(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
}

func (s *taskResultSink) Report(ctx context.Context, result wfv1.NodeResult) error {
	var count uint64 // used to avoid spamming with these messages
	logger := logging.RequireLoggerFromContext(ctx)
	return retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2.0,
		Jitter:   0.2,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
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

func (s *taskResultSink) Finalize(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	var count uint64
	return retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
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
