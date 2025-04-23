package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

const (
	nameWFTemplateTriggered = `workflowtemplate_triggered_total`
)

func addWorkflowTemplateCounter(_ context.Context, m *Metrics) error {
	return m.CreateInstrument(telemetry.Int64Counter,
		nameWFTemplateTriggered,
		"Total number of workflow templates triggered by workflowTemplateRef",
		"{workflow_template}",
		telemetry.WithAsBuiltIn(),
	)
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

	m.AddInt(ctx, nameWFTemplateTriggered, 1, attribs)
}
