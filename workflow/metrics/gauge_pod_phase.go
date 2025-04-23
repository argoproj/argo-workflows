package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

// PodPhaseCallback is the function prototype to provide this gauge with the phase of the pods
type PodPhaseCallback func() map[string]int64

type podPhaseGauge struct {
	callback PodPhaseCallback
	gauge    *telemetry.Instrument
}

func addPodPhaseGauge(ctx context.Context, m *Metrics) error {
	const namePodsPhase = `pods_gauge`
	err := m.CreateInstrument(telemetry.Int64ObservableGauge,
		namePodsPhase,
		"Number of Pods from Workflows currently accessible by the controller by status.",
		"{pod}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}

	if m.callbacks.PodPhase != nil {
		ppGauge := podPhaseGauge{
			callback: m.callbacks.PodPhase,
			gauge:    m.GetInstrument(namePodsPhase),
		}
		return ppGauge.gauge.RegisterCallback(m.Metrics, ppGauge.update)
	}
	return nil
}

func (p *podPhaseGauge) update(_ context.Context, o metric.Observer) error {
	phases := p.callback()
	for phase, val := range phases {
		p.gauge.ObserveInt(o, val, telemetry.InstAttribs{{Name: telemetry.AttribPodPhase, Value: phase}})
	}
	return nil
}
