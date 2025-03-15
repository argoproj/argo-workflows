package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addPodPhaseCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentPodsTotalCount)
}

func (m *Metrics) ChangePodPhase(ctx context.Context, phase, namespace string) {
	m.AddInt(ctx, telemetry.InstrumentPodsTotalCount.Name(), 1, telemetry.InstAttribs{
		{Name: telemetry.AttribPodPhase, Value: phase},
		{Name: telemetry.AttribPodNamespace, Value: namespace},
	})
}
