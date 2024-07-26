package metrics

import (
	"context"
)

const (
	namePodMissing = `pod_missing`
)

func addPodMissingCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		namePodMissing,
		"Incidents of pod missing.",
		"{pod}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) incPodMissing(ctx context.Context, val int64, recentlyStarted bool, phase string) {
	m.addInt(ctx, namePodMissing, val, instAttribs{
		{name: labelRecentlyStarted, value: recentlyStarted},
		{name: labelNodePhase, value: phase},
	})
}

func (m *Metrics) PodMissingEnsure(ctx context.Context, recentlyStarted bool, phase string) {
	m.incPodMissing(ctx, 0, recentlyStarted, phase)
}

func (m *Metrics) PodMissingInc(ctx context.Context, recentlyStarted bool, phase string) {
	m.incPodMissing(ctx, 1, recentlyStarted, phase)
}
