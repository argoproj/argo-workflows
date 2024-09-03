package metrics

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

const (
	nameWorkflowPhaseCounter = `total_count`
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
	return m.createInstrument(int64Counter,
		nameWorkflowPhaseCounter,
		"Total number of workflows that have entered each phase",
		"{workflow}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) ChangeWorkflowPhase(ctx context.Context, phase MetricWorkflowPhase, namespace string) {
	m.addInt(ctx, nameWorkflowPhaseCounter, 1, instAttribs{
		{name: labelWorkflowPhase, value: string(phase)},
		{name: labelWorkflowNamespace, value: namespace},
	})
}
