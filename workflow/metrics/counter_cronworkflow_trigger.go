package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

const (
	nameCronTriggered = `cronworkflows_triggered_total`
)

func addCronWfTriggerCounter(_ context.Context, m *Metrics) error {
	return m.CreateInstrument(telemetry.Int64Counter,
		nameCronTriggered,
		"Total number of cron workflows triggered",
		"{cronworkflow}",
		telemetry.WithAsBuiltIn(),
	)
}

func (m *Metrics) CronWfTrigger(ctx context.Context, name, namespace string) {
	m.AddInt(ctx, nameCronTriggered, 1, telemetry.InstAttribs{
		{Name: telemetry.AttribCronWFName, Value: name},
		{Name: telemetry.AttribWorkflowNamespace, Value: namespace},
	})
}
