package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

// WorkflowPhaseCallback is the function prototype to provide this gauge with the phase of the pods
type WorkflowPhaseCallback func(ctx context.Context) map[string]int64

type workflowPhaseGauge struct {
	callback WorkflowPhaseCallback
	observe  func(ctx context.Context, o metric.Observer, val int64, workflowStatus string)
}

func addWorkflowPhaseGauge(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentGauge)
	if err != nil {
		return err
	}

	if m.callbacks.WorkflowPhase != nil {
		inst := m.GetInstrument(telemetry.InstrumentGauge.Name())
		wfpGauge := workflowPhaseGauge{
			callback: m.callbacks.WorkflowPhase,
			observe:  m.ObserveGauge,
		}
		return inst.RegisterCallback(m.Metrics, wfpGauge.update)
	}
	return nil
	// TODO init all phases?
}

func (p *workflowPhaseGauge) update(ctx context.Context, o metric.Observer) error {
	phases := p.callback(ctx)
	for phase, val := range phases {
		p.observe(ctx, o, val, phase)
	}
	return nil
}
