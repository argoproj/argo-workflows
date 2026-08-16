package types

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

type WorkflowTemplateStore interface {
	Getter(ctx context.Context, namespace string) templateresolution.WorkflowTemplateNamespacedGetter
}
type ClusterWorkflowTemplateStore interface {
	Getter(ctx context.Context) templateresolution.ClusterWorkflowTemplateGetter
}
