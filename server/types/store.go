package types

import "github.com/argoproj/argo-workflows/v3/workflow/templateresolution"

type WorkflowTemplateStore interface {
	Getter(namespace string) templateresolution.WorkflowTemplateNamespacedGetter
}
type ClusterWorkflowTemplateStore interface {
	Getter() templateresolution.ClusterWorkflowTemplateGetter
}
