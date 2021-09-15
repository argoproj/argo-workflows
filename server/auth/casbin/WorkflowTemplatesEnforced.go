package casbin

import (
	"context"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type WorkflowTemplatesEnforced struct {
	delegate v1alpha1.WorkflowTemplateInterface
	enforcer *CustomEnforcer
}

const (
	WorkflowsTemplates = "workflowtemplates"
)


func (c WorkflowEnforcedInterface) WorkflowTemplates(namespace string) v1alpha1.WorkflowTemplateInterface {
	return &WorkflowTemplatesEnforced{c.delegate.ArgoprojV1alpha1().WorkflowTemplates(namespace),GetCustomEnforcerInstance()}
}

func (c WorkflowTemplatesEnforced) Create(ctx context.Context, workflowTemplate *wfv1.WorkflowTemplate, opts metav1.CreateOptions) (*wfv1.WorkflowTemplate, error) {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, workflowTemplate.Namespace, workflowTemplate.Name, ActionCreate); err != nil {
		return nil, err
	}
	return c.delegate.Create(ctx, workflowTemplate, opts)
}

func (c WorkflowTemplatesEnforced) Update(ctx context.Context, workflowTemplate *wfv1.WorkflowTemplate, opts metav1.UpdateOptions) (*wfv1.WorkflowTemplate, error) {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, workflowTemplate.Namespace, workflowTemplate.Name, ActionUpdate); err != nil {
		return nil, err
	}
	return c.delegate.Update(ctx, workflowTemplate, opts)
}

func (c WorkflowTemplatesEnforced) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, "", name, ActionDelete); err != nil {
		return nil
	}
	return c.delegate.Delete(ctx, name, opts)
}

func (c WorkflowTemplatesEnforced) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, "", "", ActionDeleteCollection); err != nil {
		return nil
	}
	return c.delegate.DeleteCollection(ctx, opts, listOpts)
}

func (c WorkflowTemplatesEnforced) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.WorkflowTemplate, error) {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, "", name, ActionGet); err != nil {
		return nil, err
	}
	return c.delegate.Get(ctx, name, opts)
}

func (c WorkflowTemplatesEnforced) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowTemplateList, error) {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, "*", "*", ActionList); err != nil {
		return nil, err
	}
	return c.delegate.List(ctx, opts)
}

func (c WorkflowTemplatesEnforced) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, "", "*", ActionWatch); err != nil {
		return nil, err
	}
	return c.delegate.Watch(ctx, opts)
}

func (c WorkflowTemplatesEnforced) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.WorkflowTemplate, err error) {
	if err := c.enforcer.enforce(ctx, WorkflowsTemplates, "", name, ActionWatch); err != nil {
		return nil, err
	}
	return c.delegate.Patch(ctx, name, pt, data, opts, subresources...)
}