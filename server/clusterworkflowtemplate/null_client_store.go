package clusterworkflowtemplate

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

// NullClusterWorkflowTemplateStore is a no-op implementation of ClusterWorkflowTemplateStore
// implements both informer and store interfaces
type NullClusterWorkflowTemplateStore struct{}

func NewNullClusterWorkflowTemplate() *NullClusterWorkflowTemplateStore {
	return &NullClusterWorkflowTemplateStore{}
}

func (NullClusterWorkflowTemplateStore) Getter(context.Context) templateresolution.ClusterWorkflowTemplateGetter {
	return &templateresolution.NullClusterWorkflowTemplateGetter{}
}

func (NullClusterWorkflowTemplateStore) Run(stopCh <-chan struct{}) {}
