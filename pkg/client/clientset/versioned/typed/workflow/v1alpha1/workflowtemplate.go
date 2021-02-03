// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	scheme "github.com/argoproj/argo/v3/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// WorkflowTemplatesGetter has a method to return a WorkflowTemplateInterface.
// A group's client should implement this interface.
type WorkflowTemplatesGetter interface {
	WorkflowTemplates(namespace string) WorkflowTemplateInterface
}

// WorkflowTemplateInterface has methods to work with WorkflowTemplate resources.
type WorkflowTemplateInterface interface {
	Create(ctx context.Context, workflowTemplate *v1alpha1.WorkflowTemplate, opts v1.CreateOptions) (*v1alpha1.WorkflowTemplate, error)
	Update(ctx context.Context, workflowTemplate *v1alpha1.WorkflowTemplate, opts v1.UpdateOptions) (*v1alpha1.WorkflowTemplate, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.WorkflowTemplate, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.WorkflowTemplateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.WorkflowTemplate, err error)
	WorkflowTemplateExpansion
}

// workflowTemplates implements WorkflowTemplateInterface
type workflowTemplates struct {
	client rest.Interface
	ns     string
}

// newWorkflowTemplates returns a WorkflowTemplates
func newWorkflowTemplates(c *ArgoprojV1alpha1Client, namespace string) *workflowTemplates {
	return &workflowTemplates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the workflowTemplate, and returns the corresponding workflowTemplate object, and an error if there is any.
func (c *workflowTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.WorkflowTemplate, err error) {
	result = &v1alpha1.WorkflowTemplate{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("workflowtemplates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of WorkflowTemplates that match those selectors.
func (c *workflowTemplates) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.WorkflowTemplateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.WorkflowTemplateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("workflowtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested workflowTemplates.
func (c *workflowTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("workflowtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a workflowTemplate and creates it.  Returns the server's representation of the workflowTemplate, and an error, if there is any.
func (c *workflowTemplates) Create(ctx context.Context, workflowTemplate *v1alpha1.WorkflowTemplate, opts v1.CreateOptions) (result *v1alpha1.WorkflowTemplate, err error) {
	result = &v1alpha1.WorkflowTemplate{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("workflowtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(workflowTemplate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a workflowTemplate and updates it. Returns the server's representation of the workflowTemplate, and an error, if there is any.
func (c *workflowTemplates) Update(ctx context.Context, workflowTemplate *v1alpha1.WorkflowTemplate, opts v1.UpdateOptions) (result *v1alpha1.WorkflowTemplate, err error) {
	result = &v1alpha1.WorkflowTemplate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("workflowtemplates").
		Name(workflowTemplate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(workflowTemplate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the workflowTemplate and deletes it. Returns an error if one occurs.
func (c *workflowTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("workflowtemplates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *workflowTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("workflowtemplates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched workflowTemplate.
func (c *workflowTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.WorkflowTemplate, err error) {
	result = &v1alpha1.WorkflowTemplate{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("workflowtemplates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
