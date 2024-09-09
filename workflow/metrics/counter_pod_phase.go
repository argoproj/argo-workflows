package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

const (
	namePodPhase = `pods_total_count`
)

func addPodPhaseCounter(_ context.Context, m *Metrics) error {
	return m.CreateInstrument(telemetry.Int64Counter,
		namePodPhase,
		"Total number of Pods that have entered each phase",
		"{pod}",
		telemetry.WithAsBuiltIn(),
	)
}

func (m *Metrics) ChangePodPhase(ctx context.Context, phase, namespace string) {
	m.AddInt(ctx, namePodPhase, 1, telemetry.InstAttribs{
		{Name: telemetry.AttribPodPhase, Value: phase},
		{Name: telemetry.AttribPodNamespace, Value: namespace},
	})
}
