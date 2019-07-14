package templateresolution

import (
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// maxResolveDepth is the limit of template reference resolution.
const maxResolveDepth int = 10

// Context is a context of template search.
type Context struct {
	// namespace is the namespace of template search.
	namespace string
	// wfClientset is the clientset to get workflow templates.
	wfClientset wfclientset.Interface
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateGetter
}

// NewContext returns new Context.
func NewContext(namespace string, wfClientset wfclientset.Interface, tmplBase wfv1.TemplateGetter) *Context {
	return &Context{
		namespace:   namespace,
		wfClientset: wfClientset,
		tmplBase:    tmplBase,
	}
}

// GetTemplateByName returns a template by name in the context.
func (ctx *Context) GetTemplateByName(name string) (*wfv1.Template, error) {
	tmpl := ctx.tmplBase.GetTemplateByName(name)
	if tmpl == nil {
		return nil, errors.Errorf(errors.CodeNotFound, "template %s not found", name)
	}
	return tmpl.DeepCopy(), nil
}

// GetTemplateFromRef returns a template found by a given template ref.
func (ctx *Context) GetTemplateFromRef(tmplRef *wfv1.TemplateRef) (*wfv1.Template, error) {
	wftmpl, err := ctx.wfClientset.ArgoprojV1alpha1().WorkflowTemplates(ctx.namespace).Get(tmplRef.Name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			return nil, errors.Errorf(errors.CodeNotFound, "workflow template %s not found", tmplRef.Name)
		}
		return nil, err
	}
	tmpl := wftmpl.GetTemplateByName(tmplRef.Template)
	if tmpl == nil {
		return nil, errors.Errorf(errors.CodeNotFound, "template %s not found in workflow template %s", tmplRef.Template, tmplRef.Name)
	}
	return tmpl.DeepCopy(), nil
}

// GetTemplate returns a template found by template name or template ref.
func (ctx *Context) GetTemplate(tmplHolder wfv1.TemplateHolder) (*wfv1.Template, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		return ctx.GetTemplateFromRef(tmplRef)
	} else {
		tmplName := tmplHolder.GetTemplateName()
		tmpl := ctx.tmplBase.GetTemplateByName(tmplName)
		if tmpl == nil {
			return nil, errors.Errorf(errors.CodeNotFound, "template %s not found", tmplName)
		}
		return tmpl.DeepCopy(), nil
	}
}

// GetTemplateBase returns a template base of a found template.
func (ctx *Context) GetTemplateBase(tmplHolder wfv1.TemplateHolder) (wfv1.TemplateGetter, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		wftmpl, err := ctx.wfClientset.ArgoprojV1alpha1().WorkflowTemplates(ctx.namespace).Get(tmplRef.Name, metav1.GetOptions{})
		if err != nil && apierr.IsNotFound(err) {
			return nil, errors.Errorf(errors.CodeNotFound, "workflow template %s not found", tmplRef.Name)
		}
		return wftmpl, err
	} else {
		return ctx.tmplBase, nil
	}
}

// ResolveTemplate digs into referenes and returns a merged template.
// This method is the public start point of template resolution.
func (ctx *Context) ResolveTemplate(tmplHolder wfv1.TemplateHolder) (*Context, *wfv1.Template, error) {
	return ctx.resolveTemplateImpl(tmplHolder, 0)
}

// resolveTemplateImpl digs into referenes and returns a merged template.
func (ctx *Context) resolveTemplateImpl(tmplHolder wfv1.TemplateHolder, depth int) (*Context, *wfv1.Template, error) {
	// Avoid infinite referenes
	if depth > maxResolveDepth {
		return nil, nil, errors.Errorf(errors.CodeBadRequest, "template reference exceeded max depth (%d)", maxResolveDepth)
	}

	// Find template and context
	tmpl, err := ctx.GetTemplate(tmplHolder)
	if err != nil {
		return nil, nil, err
	}
	newTmplBase, err := ctx.GetTemplateBase(tmplHolder)
	if err != nil {
		return nil, nil, err
	}
	newTmplCtx := ctx.WithTemplateBase(newTmplBase)

	// Return a concrete template without digging into it.
	if tmpl.GetType() != wfv1.TemplateTypeUnknown {
		return newTmplCtx, tmpl, nil
	}

	// Dig into nested references with new template base.
	finalTmplCtx, newTmpl, err := newTmplCtx.resolveTemplateImpl(tmpl, depth+1)
	if err != nil {
		return nil, nil, err
	}

	// Merge the referred template into the original.
	mergedTmpl, err := common.MergeReferredTemplate(tmpl, newTmpl)
	if err != nil {
		return nil, nil, err
	}

	return finalTmplCtx, mergedTmpl, nil
}

// WithTemplateBase creates new context with a wfv1.TemplateGetter.
func (ctx *Context) WithTemplateBase(tmplBase wfv1.TemplateGetter) *Context {
	return NewContext(ctx.namespace, ctx.wfClientset, tmplBase)
}
