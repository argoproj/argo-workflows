package workflowtemplate

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

// Store is a wrapper around informer
type ClientStore struct {
}

func NewClientStore() *ClientStore {
	return &ClientStore{}
}

func (wcs *ClientStore) Getter(ctx context.Context, namespace string) templateresolution.WorkflowTemplateNamespacedGetter {
	wfClient := auth.GetWfClient(ctx)
	return templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))
}
