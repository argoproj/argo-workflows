package metrics

import (
	"context"
)

const (
	nameWFTemplateTriggered = `workflowtemplate_triggered_total`
)

func addWorkflowTemplateCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		nameWFTemplateTriggered,
		"Total number of workflow templates triggered by workflowTemplateRef",
		"{workflow_template}",
		withAsBuiltIn(),
	)
}

func templateLabels(name, namespace string, cluster bool) instAttribs {
	return instAttribs{
		{name: labelTemplateName, value: name},
		{name: labelTemplateNamespace, value: namespace},
		{name: labelTemplateCluster, value: cluster},
	}
}

func (m *Metrics) CountWorkflowTemplate(ctx context.Context, phase MetricWorkflowPhase, name, namespace string, cluster bool) {
	labels := templateLabels(name, namespace, cluster)
	labels = append(labels, instAttrib{name: labelWorkflowPhase, value: string(phase)})

	m.addInt(ctx, nameWFTemplateTriggered, 1, labels)
}
