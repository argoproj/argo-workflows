package templateresolution

import (
	"fmt"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// maxResolveDepth is the limit of template reference resolution.
const maxResolveDepth int = 10

// Context is a context of template search.
type Context struct {
	// wfClientset is the clientset to get workflow templates.
	wfClientset wfclientset.Interface
	// namespace is the namespace of template search.
	namespace string
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateGetter
}

// NewContext returns new Context.
func NewContext(wfClientset wfclientset.Interface, namespace string, tmplBase wfv1.TemplateGetter) *Context {
	return &Context{
		wfClientset: wfClientset,
		namespace:   namespace,
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
	log.Debugf("Getting the template of %s on %s", getHolderDebugString(tmplHolder), getGetterDebugString(ctx.tmplBase))

	tmplName := tmplHolder.GetTemplateName()
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		return ctx.GetTemplateFromRef(tmplRef)
	} else if tmplName != "" {
		return ctx.GetTemplateByName(tmplName)
	} else {
		if tmpl, ok := tmplHolder.(*wfv1.Template); ok {
			if tmpl.GetType() != wfv1.TemplateTypeUnknown {
				return tmpl.DeepCopy(), nil
			}
			return nil, errors.Errorf(errors.CodeNotFound, "template %s is not a concrete template", tmpl.Name)
		}
	}
	return nil, errors.Errorf(errors.CodeInternal, "failed to get a template")
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

	log.Debugf("Resolving %s on %s (%d)", getHolderDebugString(tmplHolder), getGetterDebugString(ctx.tmplBase), depth)

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
	return NewContext(ctx.wfClientset, ctx.namespace, tmplBase)
}

// getGetterDebugString returns a string for debugging.
func getGetterDebugString(getter wfv1.TemplateGetter) string {
	return fmt.Sprintf("%T (namespace=%s,name=%s)", getter, getter.GetNamespace(), getter.GetName())
}

// getHolderDebugString returns a string for debugging.
func getHolderDebugString(tmplHolder wfv1.TemplateHolder) string {
	tmplName := tmplHolder.GetTemplateName()
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		return fmt.Sprintf("%T TemplateRef(name=%s,template=%s)", tmplHolder, tmplRef.Name, tmplRef.Template)
	} else {
		return fmt.Sprintf("%T Template(%s)", tmplHolder, tmplName)
	}
}
