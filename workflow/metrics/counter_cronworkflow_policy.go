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
	m.AddInt(ctx, telemetry.InstrumentCronworkflowsConcurrencypolicyTriggered.Name(), 1, telemetry.InstAttribs{
		{Name: telemetry.AttribCronWFName, Value: name},
		{Name: telemetry.AttribCronWFNamespace, Value: namespace},
		{Name: telemetry.AttribConcurrencyPolicy, Value: string(policy)},
	})
}
