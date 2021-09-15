package casbin

import (
	"context"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type WorkflowEventBindingsEnforced struct {
	delegate v1alpha1.WorkflowEventBindingInterface
}

func (c WorkflowEnforcedInterface) WorkflowEventBindings(namespace string) v1alpha1.WorkflowEventBindingInterface {
	return &WorkflowEventBindingsEnforced{c.delegate.ArgoprojV1alpha1().WorkflowEventBindings(namespace)}
}

func (w WorkflowEventBindingsEnforced) Create(ctx context.Context, workflowEventBinding *wfv1.WorkflowEventBinding, opts metav1.CreateOptions) (*wfv1.WorkflowEventBinding, error) {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) Update(ctx context.Context, workflowEventBinding *wfv1.WorkflowEventBinding, opts metav1.UpdateOptions) (*wfv1.WorkflowEventBinding, error) {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.WorkflowEventBinding, error) {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowEventBindingList, error) {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (w WorkflowEventBindingsEnforced) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.WorkflowEventBinding, err error) {
	panic("implement me")
}