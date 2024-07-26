package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// PodPhaseCallback is the function prototype to provide this gauge with the phase of the pods
type PodPhaseCallback func() map[string]int64

type podPhaseGauge struct {
	callback PodPhaseCallback
	gauge    *instrument
}

func addPodPhaseGauge(ctx context.Context, m *Metrics) error {
	const namePodsPhase = `pods_gauge`
	err := m.createInstrument(int64ObservableGauge,
		namePodsPhase,
		"Number of Pods from Workflows currently accessible by the controller by status.",
		"{pod}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}

	if m.callbacks.PodPhase != nil {
		ppGauge := podPhaseGauge{
			callback: m.callbacks.PodPhase,
			gauge:    m.allInstruments[namePodsPhase],
		}
		return m.allInstruments[namePodsPhase].registerCallback(m, ppGauge.update)
	}
	return nil
}

func (p *podPhaseGauge) update(_ context.Context, o metric.Observer) error {
	phases := p.callback()
	for phase, val := range phases {
		p.gauge.observeInt(o, val, instAttribs{{name: labelPodPhase, value: phase}})
	}
	return nil
}
