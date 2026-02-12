package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addPodPhaseCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentPodsTotalCount)
}

func (m *Metrics) ChangePodPhase(ctx context.Context, phase, namespace string) {
	m.AddPodsTotalCount(ctx, 1, phase, namespace)
}
