package casbin

import (
	"context"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type CronWorkflowsEnforced struct {
	delegate v1alpha1.CronWorkflowInterface
	namespace string
	enforcer *CustomEnforcer
}

const (
	CronWorkflows = "cronworkflows"
)

func (c WorkflowEnforcedInterface) CronWorkflows(namespace string) v1alpha1.CronWorkflowInterface {
	return &CronWorkflowsEnforced{c.delegate.ArgoprojV1alpha1().CronWorkflows(namespace), namespace,GetCustomEnforcerInstance()}
}

func (c CronWorkflowsEnforced) Create(ctx context.Context, cronWorkflow *wfv1.CronWorkflow, opts metav1.CreateOptions) (*wfv1.CronWorkflow, error) {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, cronWorkflow.Name, ActionCreate); err != nil {
		return nil, err
	}
	return c.delegate.Create(ctx, cronWorkflow, opts)
}

func (c CronWorkflowsEnforced) Update(ctx context.Context, cronWorkflow *wfv1.CronWorkflow, opts metav1.UpdateOptions) (*wfv1.CronWorkflow, error) {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, cronWorkflow.Name, ActionUpdate); err != nil {
		return nil, err
	}
	return c.delegate.Update(ctx, cronWorkflow, opts)
}

func (c CronWorkflowsEnforced) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, name, ActionDelete); err != nil {
		return err
	}
	return c.delegate.Delete(ctx, name, opts)
}

func (c CronWorkflowsEnforced) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, listOpts.FieldSelector, ActionDeleteCollection); err != nil {
		return err
	}
	return c.delegate.DeleteCollection(ctx, opts, listOpts)
}

func (c CronWorkflowsEnforced) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.CronWorkflow, error) {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, name, ActionGet); err != nil {
		return nil, err
	}
	return c.delegate.Get(ctx, name, opts)
}

func (c CronWorkflowsEnforced) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.CronWorkflowList, error) {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, "*", ActionList); err != nil {
		return nil, err
	}
	return c.delegate.List(ctx, opts)
}

func (c CronWorkflowsEnforced) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, "*", ActionWatch); err != nil {
		return nil, err
	}
	return c.delegate.Watch(ctx, opts)
}

func (c CronWorkflowsEnforced) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.CronWorkflow, err error) {
	if err := c.enforcer.enforce(ctx, CronWorkflows, c.namespace, name, ActionPatch); err != nil {
		return nil, err
	}
	return c.delegate.Patch(ctx, name, pt, data, opts, subresources...)
}