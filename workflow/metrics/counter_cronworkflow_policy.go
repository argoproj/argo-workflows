package metrics

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addCronWfPolicyCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentCronworkflowsConcurrencypolicyTriggered)
}

func (m *Metrics) CronWfPolicy(ctx context.Context, name, namespace string, policy wfv1.ConcurrencyPolicy) {
	m.AddCronworkflowsConcurrencypolicyTriggered(ctx, 1, name, namespace, string(policy))
}
