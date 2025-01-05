package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addCronWfTriggerCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentCronworkflowsTriggeredTotal)
}

func (m *Metrics) CronWfTrigger(ctx context.Context, name, namespace string) {
	m.AddInt(ctx, telemetry.InstrumentCronworkflowsTriggeredTotal.Name(), 1, telemetry.InstAttribs{
		{Name: telemetry.AttribCronWFName, Value: name},
		{Name: telemetry.AttribCronWFNamespace, Value: namespace},
	})
}
