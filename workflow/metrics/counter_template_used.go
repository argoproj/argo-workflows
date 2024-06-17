package metrics

import (
	"context"
)

const (
	nameWorkflowTemplateUsed = `workflowtemplate_used_total`
)

func addWorkflowTemplateUsedCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		nameWorkflowTemplateUsed,
		"Total number of workflow templates used in any fashion",
		"{workflow_template}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) CountWorkflowTemplateUsed(ctx context.Context, name, namespace string, cluster bool) {
	m.addInt(ctx, nameWorkflowTemplateUsed, 1, templateLabels(name, namespace, cluster))
}
