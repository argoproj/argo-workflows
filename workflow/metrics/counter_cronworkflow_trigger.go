package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addCronWfTriggerCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentCronworkflowsTriggeredTotal)
}

func (m *Metrics) CronWfTrigger(ctx context.Context, name, namespace string) {
	m.AddCronworkflowsTriggeredTotal(ctx, 1, name, namespace)
}
