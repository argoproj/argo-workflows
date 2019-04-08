package controller

import (
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// maxResolveDepth is the limit of template reference resolution.
const maxResolveDepth int = 10

// templateContext is a context of template search.
type templateContext struct {
	// namespace is the namespace of template search.
	namespace string
	// wfclientset is the clientset to get workflow templates.
	wfclientset wfclientset.Interface
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateGetter
}

// getTemplateFromRef returns a template found by a given template ref.
func (ctx *templateContext) getTemplateFromRef(tmplRef *wfv1.TemplateRef) (*wfv1.Template, error) {
	wftmpl, err := ctx.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(ctx.namespace).Get(tmplRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	tmpl := wftmpl.GetTemplateByName(tmplRef.Template)
	if tmpl == nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "template %s not found in workflow template %s", tmplRef.Template, tmplRef.Name)
	}
	return tmpl.DeepCopy(), nil
}

// getTemplate returns a template found by template name or template ref.
func (ctx *templateContext) getTemplate(tmplHolder wfv1.TemplateHolder) (*wfv1.Template, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		return ctx.getTemplateFromRef(tmplRef)
	} else {
		tmplName := tmplHolder.GetTemplateName()
		tmpl := ctx.tmplBase.GetTemplateByName(tmplName)
		if tmpl == nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "template %s not found", tmplName)
		}
		return tmpl.DeepCopy(), nil
	}
}

// getTemplateBase returns a template base of a found template.
func (ctx *templateContext) getTemplateBase(tmplHolder wfv1.TemplateHolder) (wfv1.TemplateGetter, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		return ctx.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(ctx.namespace).Get(tmplRef.Name, metav1.GetOptions{})
	} else {
		return ctx.tmplBase, nil
	}
}

// getTemplateAndContext returns a template found by template name or template ref with its search context.
func (ctx *templateContext) getTemplateAndContext(tmplHolder wfv1.TemplateHolder) (*templateContext, *wfv1.Template, error) {
	tmpl, err := ctx.getTemplate(tmplHolder)
	if err != nil {
		return nil, nil, err
	}
	tmplBase, err := ctx.getTemplateBase(tmplHolder)
	if err != nil {
		return nil, nil, err
	}
	newCtx := &templateContext{
		namespace:   ctx.namespace,
		wfclientset: ctx.wfclientset,
		tmplBase:    tmplBase,
	}
	return newCtx, tmpl, nil
}

// resolveTemplate digs into referenes and returns a merged template.
func (ctx *templateContext) resolveTemplate(tmplHolder wfv1.TemplateHolder, depth int) (*templateContext, *wfv1.Template, error) {
	// Avoid infinite referenes
	if depth > maxResolveDepth {
		return nil, nil, errors.Errorf(errors.CodeBadRequest, "template reference exceeded max depth (%d)", maxResolveDepth)
	}

	// Find template and context
	newTmplCtx, tmpl, err := ctx.getTemplateAndContext(tmplHolder)
	if err != nil {
		return nil, nil, err
	}

	// Return a concrete template without digging into it.
	if tmpl.GetType() != wfv1.TemplateTypeUnknown {
		return newTmplCtx, tmpl, nil
	}

	// Dig into nested references with new template base.
	finalTmplCtx, newTmpl, err := newTmplCtx.resolveTemplate(tmpl, depth+1)
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
