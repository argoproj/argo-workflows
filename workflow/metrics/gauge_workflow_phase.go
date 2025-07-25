package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

// WorkflowPhaseCallback is the function prototype to provide this gauge with the phase of the pods
type WorkflowPhaseCallback func(ctx context.Context) map[string]int64

type workflowPhaseGauge struct {
	callback WorkflowPhaseCallback
	gauge    *telemetry.Instrument
}

func addWorkflowPhaseGauge(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentGauge)
	if err != nil {
		return err
	}

	name := telemetry.InstrumentGauge.Name()
	if m.callbacks.WorkflowPhase != nil {
		wfpGauge := workflowPhaseGauge{
			callback: m.callbacks.WorkflowPhase,
			gauge:    m.GetInstrument(name),
		}
		return wfpGauge.gauge.RegisterCallback(m.Metrics, wfpGauge.update)
	}
	return nil
	// TODO init all phases?
}

func (p *workflowPhaseGauge) update(ctx context.Context, o metric.Observer) error {
	phases := p.callback(ctx)
	for phase, val := range phases {
		p.gauge.ObserveInt(ctx, o, val, telemetry.InstAttribs{{Name: telemetry.AttribWorkflowStatus, Value: phase}})
	}
	return nil
}
