package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func addWorkflowTemplateCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowtemplateTriggeredTotal)
}

func (m *Metrics) CountWorkflowTemplate(ctx context.Context, phase MetricWorkflowPhase, name, namespace string, cluster bool) {
	m.AddWorkflowtemplateTriggeredTotal(ctx, 1, name, namespace, cluster, string(phase))
}
