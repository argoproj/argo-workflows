package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

// PodPhaseCallback is the function prototype to provide this gauge with the phase of the pods
type PodPhaseCallback func(ctx context.Context) map[string]int64

type podPhaseGauge struct {
	callback PodPhaseCallback
	observe  func(ctx context.Context, o metric.Observer, val int64, nodePhase string)
}

func addPodPhaseGauge(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentPodsGauge)
	if err != nil {
		return err
	}

	if m.callbacks.PodPhase != nil {
		inst := m.GetInstrument(telemetry.InstrumentPodsGauge.Name())
		ppGauge := podPhaseGauge{
			callback: m.callbacks.PodPhase,
			observe:  m.ObservePodsGauge,
		}
		return inst.RegisterCallback(m.Metrics, ppGauge.update)
	}
	return nil
}

func (p *podPhaseGauge) update(ctx context.Context, o metric.Observer) error {
	phases := p.callback(ctx)
	for phase, val := range phases {
		p.observe(ctx, o, val, phase)
	}
	return nil
}
