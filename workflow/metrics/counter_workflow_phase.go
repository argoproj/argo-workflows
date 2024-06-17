package metrics

import (
	"context"
)

const (
	nameWorkflowPhaseCounter = `total_count`
)

func addWorkflowPhaseCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		nameWorkflowPhaseCounter,
		"Total number of workflows that have entered each phase",
		"{workflow}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) ChangeWorkflowPhase(ctx context.Context, phase, namespace string) {
	m.addInt(ctx, nameWorkflowPhaseCounter, 1, instAttribs{
		{name: labelWorkflowPhase, value: phase},
		{name: labelWorkflowNamespace, value: namespace},
	})
}
