package workflowtemplate

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// Store is a wrapper around informer
type WorkflowTemplateClientStore struct {
}

func NewWorkflowTemplateClientStore() *WorkflowTemplateClientStore {
	return &WorkflowTemplateClientStore{}
}

func (wcs *WorkflowTemplateClientStore) Getter(ctx context.Context, namespace string) templateresolution.WorkflowTemplateNamespacedGetter {
	wfClient := auth.GetWfClient(ctx)
	return templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))
}
