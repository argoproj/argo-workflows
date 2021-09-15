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
	ClusterWorkflowTemplates = "clusterworkflowtemplates"
)

type ClusterWorkflowTemplatesEnforced struct {
	delegate v1alpha1.ClusterWorkflowTemplateInterface
	enforcer *CustomEnforcer
}

func (c WorkflowEnforcedInterface) ClusterWorkflowTemplates() v1alpha1.ClusterWorkflowTemplateInterface {
	return &ClusterWorkflowTemplatesEnforced{c.delegate.ArgoprojV1alpha1().ClusterWorkflowTemplates(), GetCustomEnforcerInstance()}
}

func (c ClusterWorkflowTemplatesEnforced) Create(ctx context.Context, clusterWorkflowTemplate *wfv1.ClusterWorkflowTemplate, opts metav1.CreateOptions) (*wfv1.ClusterWorkflowTemplate, error) {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", clusterWorkflowTemplate.Name, ActionCreate); err != nil {
		return nil, err
	}
	return c.delegate.Create(ctx, clusterWorkflowTemplate, opts)
}

func (c ClusterWorkflowTemplatesEnforced) Update(ctx context.Context, clusterWorkflowTemplate *wfv1.ClusterWorkflowTemplate, opts metav1.UpdateOptions) (*wfv1.ClusterWorkflowTemplate, error) {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", clusterWorkflowTemplate.Name, ActionUpdate); err != nil {
		return nil, err
	}
	return c.delegate.Update(ctx, clusterWorkflowTemplate, opts)
}

func (c ClusterWorkflowTemplatesEnforced) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", name, ActionDelete); err != nil {
		return nil
	}
	return c.delegate.Delete(ctx, name, opts)
}

func (c ClusterWorkflowTemplatesEnforced) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", listOpts.FieldSelector, ActionDeleteCollection); err != nil {
		return nil
	}
	return c.delegate.DeleteCollection(ctx, opts, listOpts)
}

func (c ClusterWorkflowTemplatesEnforced) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.ClusterWorkflowTemplate, error) {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", name, ActionGet); err != nil {
		return nil, err
	}
	return c.delegate.Get(ctx, name, opts)
}

func (c ClusterWorkflowTemplatesEnforced) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.ClusterWorkflowTemplateList, error) {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", opts.FieldSelector, ActionList); err != nil {
		return nil, err
	}
	return c.delegate.List(ctx, opts)
}

func (c ClusterWorkflowTemplatesEnforced) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", opts.FieldSelector, ActionWatch); err != nil {
		return nil, err
	}
	return c.delegate.Watch(ctx, opts)
}

func (c ClusterWorkflowTemplatesEnforced) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.ClusterWorkflowTemplate, err error) {
	if err := c.enforcer.enforce(ctx, ClusterWorkflowTemplates, "*", name, ActionPatch); err != nil {
		return nil, err
	}
	return c.delegate.Patch(ctx, name, pt, data, opts, subresources...)
}