package metrics

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

type MetricWorkflowPhase string

const (
	WorkflowUnknown   MetricWorkflowPhase = MetricWorkflowPhase(wfv1.WorkflowUnknown)
	WorkflowPending   MetricWorkflowPhase = MetricWorkflowPhase(wfv1.WorkflowPending)
	WorkflowRunning   MetricWorkflowPhase = MetricWorkflowPhase(wfv1.WorkflowRunning)
	WorkflowSucceeded MetricWorkflowPhase = MetricWorkflowPhase(wfv1.WorkflowSucceeded)
	WorkflowFailed    MetricWorkflowPhase = MetricWorkflowPhase(wfv1.WorkflowFailed)
	WorkflowError     MetricWorkflowPhase = MetricWorkflowPhase(wfv1.WorkflowError)
	WorkflowNew       MetricWorkflowPhase = "New"
)

func ConvertWorkflowPhase(inPhase wfv1.WorkflowPhase) MetricWorkflowPhase {
	switch inPhase {
	case wfv1.WorkflowPending:
		return WorkflowPending
	case wfv1.WorkflowRunning:
		return WorkflowRunning
	case wfv1.WorkflowSucceeded:
		return WorkflowSucceeded
	case wfv1.WorkflowFailed:
		return WorkflowFailed
	case wfv1.WorkflowError:
		return WorkflowError
	default:
		return WorkflowUnknown
	}
}

func addWorkflowPhaseCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentTotalCount)
}

func (m *Metrics) ChangeWorkflowPhase(ctx context.Context, phase MetricWorkflowPhase, namespace string) {
	m.AddTotalCount(ctx, 1, string(phase), namespace)
}
