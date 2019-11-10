package templateresolution

import (
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	typed "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// maxResolveDepth is the limit of template reference resolution.
const maxResolveDepth int = 10

// workflowTemplateInterfaceWrapper is an internal struct to wrap clientset.
type workflowTemplateInterfaceWrapper struct {
	clientset typed.WorkflowTemplateInterface
}

// Get retrieves the WorkflowTemplate of a given name.
func (wrapper *workflowTemplateInterfaceWrapper) Get(name string) (*wfv1.WorkflowTemplate, error) {
	return wrapper.clientset.Get(name, metav1.GetOptions{})
}

// WorkflowTemplateNamespaceLister helps get WorkflowTemplates.
type WorkflowTemplateNamespacedGetter interface {
	// Get retrieves the WorkflowTemplate from the indexer for a given name.
	Get(name string) (*wfv1.WorkflowTemplate, error)
}

// TemplateStorage is an interface of template storage getter and setter.
type TemplateStorage interface {
	GetStoredTemplate(templateScope string, holder wfv1.TemplateHolder) *wfv1.Template
	SetStoredTemplate(templateScope string, holder wfv1.TemplateHolder, tmpl *wfv1.Template) error
}

// Context is a context of template search.
type Context struct {
	// wftmplGetter is an interface to get WorkflowTemplates.
	wftmplGetter WorkflowTemplateNamespacedGetter
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateGetter
	// storage is an implementation of TemplateStorage.
	storage TemplateStorage
}

// NewContext returns new Context.
func NewContext(wftmplGetter WorkflowTemplateNamespacedGetter, tmplBase wfv1.TemplateGetter, storage TemplateStorage) *Context {
	return &Context{
		wftmplGetter: wftmplGetter,
		tmplBase:     tmplBase,
		storage:      storage,
	}
}

// NewContext returns new Context.
func NewContextFromClientset(clientset typed.WorkflowTemplateInterface, tmplBase wfv1.TemplateGetter, storage TemplateStorage) *Context {
	return &Context{
		wftmplGetter: &workflowTemplateInterfaceWrapper{clientset: clientset},
		tmplBase:     tmplBase,
		storage:      storage,
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
	wftmpl, err := ctx.wftmplGetter.Get(tmplRef.Name)
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
	log.Debugf("Getting the template of %s on %s", common.GetTemplateHolderString(tmplHolder), common.GetTemplateGetterString(ctx.tmplBase))

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

// GetCurrentTemplateBase returns the current template base of the context.
func (ctx *Context) GetCurrentTemplateBase() wfv1.TemplateGetter {
	return ctx.tmplBase
}

// GetTemplateBase returns a template base of a found template.
func (ctx *Context) GetTemplateBase(tmplHolder wfv1.TemplateHolder) (wfv1.TemplateGetter, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		wftmpl, err := ctx.wftmplGetter.Get(tmplRef.Name)
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
// This method processes inputs and arguments so the inputs of the final
//  resolved template include intermediate parameter passing.
// The other fields are just merged and shallower templates overwrite deeper.
func (ctx *Context) resolveTemplateImpl(tmplHolder wfv1.TemplateHolder, depth int) (*Context, *wfv1.Template, error) {
	// Avoid infinite references
	if depth > maxResolveDepth {
		return nil, nil, errors.Errorf(errors.CodeBadRequest, "template reference exceeded max depth (%d)", maxResolveDepth)
	}

	log.Debugf("Resolving %s on %s (%d)", common.GetTemplateHolderString(tmplHolder), common.GetTemplateGetterString(ctx.tmplBase), depth)

	var tmpl *wfv1.Template
	if ctx.storage != nil {
		// The template has been stored.
		tmpl = ctx.storage.GetStoredTemplate(ctx.tmplBase.GetTemplateScope(), tmplHolder)
	}
	if tmpl != nil {
		log.Debugf("Found stored template %s on %s", common.GetTemplateHolderString(tmplHolder), common.GetTemplateGetterString(ctx.tmplBase))
	} else {
		// Find template and context
		newTmpl, err := ctx.GetTemplate(tmplHolder)
		if err != nil {
			return nil, nil, err
		}
		tmpl = newTmpl
	}
	newTmplBase, err := ctx.GetTemplateBase(tmplHolder)
	if err != nil {
		return nil, nil, err
	}
	newTmplCtx := ctx.WithTemplateBase(newTmplBase)

	if ctx.storage != nil {
		err = ctx.storage.SetStoredTemplate(ctx.tmplBase.GetTemplateScope(), tmplHolder, tmpl)
		if err != nil {
			return nil, nil, err
		}
	}

	// Return a concrete template without digging into it.
	if tmpl.GetType() != wfv1.TemplateTypeUnknown {
		return newTmplCtx, tmpl, nil
	}

	// Dig into nested references with new template base.
	finalTmplCtx, resolvedTmpl, err := newTmplCtx.resolveTemplateImpl(tmpl, depth+1)
	if err != nil {
		return nil, nil, err
	}

	// Merge the referred template into the original.
	mergedTmpl, err := common.MergeReferredTemplate(tmpl, resolvedTmpl)
	if err != nil {
		return nil, nil, err
	}

	return finalTmplCtx, mergedTmpl, nil
}

// WithTemplateBase creates new context with a wfv1.TemplateGetter.
func (ctx *Context) WithTemplateBase(tmplBase wfv1.TemplateGetter) *Context {
	return NewContext(ctx.wftmplGetter, tmplBase, ctx.storage)
}

// WithLazyWorkflowTemplate creates new context with the wfv1.WorkflowTemplate of the given name with lazy loading.
func (ctx *Context) WithLazyWorkflowTemplate(namespace, name string) (*Context, error) {
	return NewContext(ctx.wftmplGetter, NewLazyWorkflowTemplate(ctx.wftmplGetter, namespace, name), ctx.storage), nil
}
