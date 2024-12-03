package v1alpha1

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// workflowTemplateGetter is used by the handlers to call the Argo Validate function
type workflowTemplateGetter struct {
	client    client.Client
	namespace string
	// the WorkflowTemplateNamespacedGetter functions do not take in context as arg
	// since we construct a new getter per request, we can store the request context
	// in this getter struct
	ctx context.Context
}

// Get implements WorkflowTemplateNamespacedGetter
func (w *workflowTemplateGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	wft := &wfv1.WorkflowTemplate{}
	err := w.client.Get(
		w.ctx,
		types.NamespacedName{
			Name:      name,
			Namespace: w.namespace,
		},
		wft,
	)
	return wft, err
}

// clusterWorkflowTemplateGetter is used by the handlers to call the Argo Validate function
type clusterWorkflowTemplateGetter struct {
	client client.Client
	// the ClusterWorkflowTemplateGetter functions do not take in context as arg
	// since we construct a new getter per request, we can store the request context
	// in this getter struct
	ctx context.Context
}

// Get implements ClusterWorkflowTemplateGetter
func (w *clusterWorkflowTemplateGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	cwft := &wfv1.ClusterWorkflowTemplate{}
	err := w.client.Get(
		w.ctx,
		types.NamespacedName{Name: name},
		cwft,
	)
	return cwft, err
}
