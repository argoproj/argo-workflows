package workflowtemplate

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

// Store is a wrapper around informer
// if
type WorkflowTemplateClientStore struct {
}

func NewWorkflowTemplateClientStore() *WorkflowTemplateClientStore {
	return &WorkflowTemplateClientStore{}
}

func (wcs *WorkflowTemplateClientStore) Getter(ctx context.Context, namespace string) templateresolution.WorkflowTemplateNamespacedGetter {
	wfClient := auth.GetWfClient(ctx)
	return templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))
}
