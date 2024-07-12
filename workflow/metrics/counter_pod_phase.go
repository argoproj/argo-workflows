package metrics

import (
	"context"
)

const (
	namePodPhase = `pods_total_count`
)

func addPodPhaseCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		namePodPhase,
		"Total number of Pods that have entered each phase",
		"{pod}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) ChangePodPhase(ctx context.Context, phase, namespace string) {
	m.addInt(ctx, namePodPhase, 1, instAttribs{
		{name: labelPodPhase, value: phase},
		{name: labelPodNamespace, value: namespace},
	})
}
