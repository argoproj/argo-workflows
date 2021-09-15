package casbin

import (
	"context"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	Workflows = "workflows"
)

type WorkflowEnforced struct {
	delegate  v1alpha1.WorkflowInterface
	namespace string
	enforcer *CustomEnforcer
}

func (c WorkflowEnforcedInterface) Workflows(namespace string) v1alpha1.WorkflowInterface {
	return &WorkflowEnforced{c.delegate.ArgoprojV1alpha1().Workflows(namespace), namespace, GetCustomEnforcerInstance()}
}

func (c WorkflowEnforced) Create(ctx context.Context, workflow *wfv1.Workflow, opts metav1.CreateOptions) (*wfv1.Workflow, error) {
	if err :=c.enforcer.enforce(ctx, Workflows, c.namespace, workflow.GetGenerateName(), ActionCreate); err != nil {
		return nil, err
	}
	return c.delegate.Create(ctx, workflow, opts)
}

func (c WorkflowEnforced) Update(ctx context.Context, workflow *wfv1.Workflow, opts metav1.UpdateOptions) (*wfv1.Workflow, error) {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, workflow.Name, ActionUpdate); err != nil {
		return nil, err
	}
	return c.delegate.Update(ctx, workflow, opts)
}

func (c WorkflowEnforced) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, name, ActionDelete); err != nil {
		return err
	}
	return c.delegate.Delete(ctx, name, opts)
}

func (c WorkflowEnforced) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, listOpts.FieldSelector, ActionDeleteCollection); err != nil {
		return err
	}
	return c.delegate.DeleteCollection(ctx, opts, listOpts)
}

func (c WorkflowEnforced) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.Workflow, error) {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, name, ActionGet); err != nil {
		return nil, err
	}
	return c.delegate.Get(ctx, name, opts)
}

func (c WorkflowEnforced) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, "*", ActionList); err != nil {
		return nil, err
	}
	return c.delegate.List(ctx, opts)
}

func (c WorkflowEnforced) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, opts.FieldSelector, ActionWatch); err != nil {
		return nil, err
	}
	return c.delegate.Watch(ctx, opts)
}

func (c WorkflowEnforced) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.Workflow, err error) {
	if err := c.enforcer.enforce(ctx, Workflows, c.namespace, name, ActionPatch); err != nil {
		return nil, err
	}
	return c.delegate.Patch(ctx, name, pt, data, opts, subresources...)
}