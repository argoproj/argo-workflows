package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func addWorkflowActionCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowactionsProcessedTotal)
}

func (m *Metrics) WorkflowActionProcessed(ctx context.Context, action, outcome string) {
	m.AddWorkflowactionsProcessedTotal(ctx, 1, action, outcome)
}
