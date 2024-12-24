package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addWorkflowTemplateCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowtemplateTriggeredTotal)
}

func templateAttribs(name, namespace string, cluster bool) telemetry.InstAttribs {
	return telemetry.InstAttribs{
		{Name: telemetry.AttribTemplateName, Value: name},
		{Name: telemetry.AttribTemplateNamespace, Value: namespace},
		{Name: telemetry.AttribTemplateCluster, Value: cluster},
	}
}

func (m *Metrics) CountWorkflowTemplate(ctx context.Context, phase MetricWorkflowPhase, name, namespace string, cluster bool) {
	attribs := templateAttribs(name, namespace, cluster)
	attribs = append(attribs, telemetry.InstAttrib{Name: telemetry.AttribWorkflowPhase, Value: string(phase)})

	m.AddInt(ctx, telemetry.InstrumentWorkflowtemplateTriggeredTotal.Name(), 1, attribs)
}
