package metrics

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

const (
	nameCronPolicy = `cronworkflows_concurrencypolicy_triggered`
)

func addCronWfPolicyCounter(_ context.Context, m *Metrics) error {
	return m.CreateInstrument(telemetry.Int64Counter,
		nameCronPolicy,
		"Total number of times CronWorkflow concurrencyPolicy has triggered",
		"{cronworkflow}",
		telemetry.WithAsBuiltIn(),
	)
}

func (m *Metrics) CronWfPolicy(ctx context.Context, name, namespace string, policy wfv1.ConcurrencyPolicy) {
	m.AddInt(ctx, nameCronPolicy, 1, telemetry.InstAttribs{
		{Name: telemetry.AttribCronWFName, Value: name},
		{Name: telemetry.AttribWorkflowNamespace, Value: namespace},
		{Name: telemetry.AttribConcurrencyPolicy, Value: string(policy)},
	})
}
