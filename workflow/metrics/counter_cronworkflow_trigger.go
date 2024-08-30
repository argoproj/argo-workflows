package metrics

import (
	"context"
)

const (
	nameCronTriggered = `cronworkflows_triggered_total`
)

func addCronWfTriggerCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		nameCronTriggered,
		"Total number of cron workflows triggered",
		"{cronworkflow}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) CronWfTrigger(ctx context.Context, name, namespace string) {
	m.addInt(ctx, nameCronTriggered, 1, instAttribs{
		{name: labelCronWFName, value: name},
		{name: labelWorkflowNamespace, value: namespace},
	})
}
