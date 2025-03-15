package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addPodMissingCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentPodMissing)
}

func (m *Metrics) incPodMissing(ctx context.Context, val int64, recentlyStarted bool, phase string) {
	m.AddInt(ctx, telemetry.InstrumentPodMissing.Name(), val, telemetry.InstAttribs{
		{Name: telemetry.AttribRecentlyStarted, Value: recentlyStarted},
		{Name: telemetry.AttribNodePhase, Value: phase},
	})
}

func (m *Metrics) PodMissingEnsure(ctx context.Context, recentlyStarted bool, phase string) {
	m.incPodMissing(ctx, 0, recentlyStarted, phase)
}

func (m *Metrics) PodMissingInc(ctx context.Context, recentlyStarted bool, phase string) {
	m.incPodMissing(ctx, 1, recentlyStarted, phase)
}
