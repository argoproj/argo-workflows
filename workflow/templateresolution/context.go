package templateresolution

import (
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	typed "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

// maxResolveDepth is the limit of template reference resolution.
const maxResolveDepth int = 10

// workflowTemplateInterfaceWrapper is an internal struct to wrap clientset.
type workflowTemplateInterfaceWrapper struct {
	clientset typed.WorkflowTemplateInterface
}

func WrapWorkflowTemplateInterface(clientset v1alpha1.WorkflowTemplateInterface) WorkflowTemplateNamespacedGetter {
	return &workflowTemplateInterfaceWrapper{clientset: clientset}
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

// clusterWorkflowTemplateInterfaceWrapper is an internal struct to wrap clientset.
type clusterWorkflowTemplateInterfaceWrapper struct {
	clientset typed.ClusterWorkflowTemplateInterface
}

// WorkflowTemplateNamespaceLister helps get WorkflowTemplates.
type ClusterWorkflowTemplateGetter interface {
	// Get retrieves the WorkflowTemplate from the indexer for a given name.
	Get(name string) (*wfv1.ClusterWorkflowTemplate, error)
}

func WrapClusterWorkflowTemplateInterface(clusterClientset v1alpha1.ClusterWorkflowTemplateInterface) ClusterWorkflowTemplateGetter {
	return &clusterWorkflowTemplateInterfaceWrapper{clientset: clusterClientset}
}

// Get retrieves the WorkflowTemplate of a given name.
func (wrapper *clusterWorkflowTemplateInterfaceWrapper) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return wrapper.clientset.Get(name, metav1.GetOptions{})
}

// Context is a context of template search.
type Context struct {
	// wftmplGetter is an interface to get WorkflowTemplates.
	wftmplGetter WorkflowTemplateNamespacedGetter
	// cwftmplGetter is an interface to get ClusterWorkflowTemplates
	cwftmplGetter ClusterWorkflowTemplateGetter
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateHolder
	// storage is an implementation of TemplateStorage.
	storage wfv1.TemplateStorage
	// log is a logrus entry.
	log *logrus.Entry
}

// NewContext returns new Context.
func NewContext(wftmplGetter WorkflowTemplateNamespacedGetter, cwftmplGetter ClusterWorkflowTemplateGetter, tmplBase wfv1.TemplateHolder, storage wfv1.TemplateStorage) *Context {
	return &Context{
		wftmplGetter:  wftmplGetter,
		cwftmplGetter: cwftmplGetter,
		tmplBase:      tmplBase,
		storage:       storage,
		log:           log.WithFields(logrus.Fields{}),
	}
}

// NewContext returns new Context.
func NewContextFromClientset(wftmplClientset typed.WorkflowTemplateInterface, clusterWftmplClient typed.ClusterWorkflowTemplateInterface, tmplBase wfv1.TemplateHolder, storage wfv1.TemplateStorage) *Context {
	return &Context{
		wftmplGetter:  WrapWorkflowTemplateInterface(wftmplClientset),
		cwftmplGetter: WrapClusterWorkflowTemplateInterface(clusterWftmplClient),
		tmplBase:      tmplBase,
		storage:       storage,
		log:           log.WithFields(logrus.Fields{}),
	}
}

// GetTemplateByName returns a template by name in the context.
func (ctx *Context) GetTemplateByName(name string) (*wfv1.Template, error) {
	ctx.log.Debug("Getting the template by name")

	tmpl := ctx.tmplBase.GetTemplateByName(name)
	if tmpl == nil {
		return nil, errors.Errorf(errors.CodeNotFound, "template %s not found", name)
	}
	return tmpl.DeepCopy(), nil
}

func (ctx *Context) GetTemplateGetterFromRef(tmplRef *wfv1.TemplateRef) (wfv1.TemplateHolder, error) {
	if tmplRef.ClusterScope {
		return ctx.cwftmplGetter.Get(tmplRef.Name)
	}
	return ctx.wftmplGetter.Get(tmplRef.Name)
}

// GetTemplateFromRef returns a template found by a given template ref.
func (ctx *Context) GetTemplateFromRef(tmplRef *wfv1.TemplateRef) (*wfv1.Template, error) {
	ctx.log.Debug("Getting the template from ref")
	var template *wfv1.Template
	var wftmpl wfv1.TemplateHolder
	var err error
	if tmplRef.ClusterScope {
		wftmpl, err = ctx.cwftmplGetter.Get(tmplRef.Name)
	} else {
		wftmpl, err = ctx.wftmplGetter.Get(tmplRef.Name)
	}

	if err != nil {
		if apierr.IsNotFound(err) {
			return nil, errors.Errorf(errors.CodeNotFound, "workflow template %s not found", tmplRef.Name)
		}
		return nil, err
	}

	template = wftmpl.GetTemplateByName(tmplRef.Template)

	if template == nil {
		return nil, errors.Errorf(errors.CodeNotFound, "template %s not found in workflow template %s", tmplRef.Template, tmplRef.Name)
	}
	return template.DeepCopy(), nil
}

// GetTemplate returns a template found by template name or template ref.
func (ctx *Context) GetTemplate(tmplHolder wfv1.TemplateReferenceHolder) (*wfv1.Template, error) {
	ctx.log.Debug("Getting the template")

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
func (ctx *Context) GetCurrentTemplateBase() wfv1.TemplateHolder {
	return ctx.tmplBase
}

// ResolveTemplate digs into referenes and returns a merged template.
// This method is the public start point of template resolution.
func (ctx *Context) ResolveTemplate(tmplHolder wfv1.TemplateReferenceHolder) (*Context, *wfv1.Template, error) {
	return ctx.resolveTemplateImpl(tmplHolder, 0)
}

// resolveTemplateImpl digs into referenes and returns a merged template.
// This method processes inputs and arguments so the inputs of the final
//  resolved template include intermediate parameter passing.
// The other fields are just merged and shallower templates overwrite deeper.
func (ctx *Context) resolveTemplateImpl(tmplHolder wfv1.TemplateReferenceHolder, depth int) (*Context, *wfv1.Template, error) {
	ctx.log = ctx.log.WithFields(logrus.Fields{
		"depth": depth,
		"base":  common.GetTemplateGetterString(ctx.tmplBase),
		"tmpl":  common.GetTemplateHolderString(tmplHolder),
	})
	// Avoid infinite references
	if depth > maxResolveDepth {
		return nil, nil, errors.Errorf(errors.CodeBadRequest, "template reference exceeded max depth (%d)", maxResolveDepth)
	}

	ctx.log.Debug("Resolving the template")

	var tmpl *wfv1.Template
	if ctx.storage != nil {
		// Check if the template has been stored.
		scope := ctx.tmplBase.GetResourceScope()
		resourceName := ctx.tmplBase.GetName()
		tmpl = ctx.storage.GetStoredTemplate(scope, resourceName, tmplHolder)
	}
	if tmpl != nil {
		ctx.log.Debug("Found stored template")
	} else {
		// Find newly appeared template.
		newTmpl, err := ctx.GetTemplate(tmplHolder)
		if err != nil {
			return nil, nil, err
		}
		// Stored the found template.
		if ctx.storage != nil {
			scope := ctx.tmplBase.GetResourceScope()
			resourceName := ctx.tmplBase.GetName()
			stored, err := ctx.storage.SetStoredTemplate(scope, resourceName, tmplHolder, newTmpl)
			if err != nil {
				return nil, nil, err
			}
			if stored {
				ctx.log.Debug("Stored the template")
			}
		}
		tmpl = newTmpl
	}

	// Update the template base of the context.
	newTmplCtx, err := ctx.WithTemplateHolder(tmplHolder)
	if err != nil {
		return nil, nil, err
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

// WithTemplateHolder creates new context with a template base of a given template holder.
func (ctx *Context) WithTemplateHolder(tmplHolder wfv1.TemplateReferenceHolder) (*Context, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		tmplName := tmplRef.Name
		if tmplRef.ClusterScope {
			return ctx.WithClusterWorkflowTemplate(tmplName)
		} else {
			return ctx.WithWorkflowTemplate(tmplName)
		}
	}
	return ctx.WithTemplateBase(ctx.tmplBase), nil
}

// WithTemplateBase creates new context with a wfv1.TemplateHolder.
func (ctx *Context) WithTemplateBase(tmplBase wfv1.TemplateHolder) *Context {
	return NewContext(ctx.wftmplGetter, ctx.cwftmplGetter, tmplBase, ctx.storage)
}

// WithWorkflowTemplate creates new context with a wfv1.TemplateHolder.
func (ctx *Context) WithWorkflowTemplate(name string) (*Context, error) {
	wftmpl, err := ctx.wftmplGetter.Get(name)
	if err != nil {
		return nil, err
	}
	return ctx.WithTemplateBase(wftmpl), nil
}

// WithWorkflowTemplate creates new context with a wfv1.TemplateHolder.
func (ctx *Context) WithClusterWorkflowTemplate(name string) (*Context, error) {
	cwftmpl, err := ctx.cwftmplGetter.Get(name)
	if err != nil {
		return nil, err
	}
	return ctx.WithTemplateBase(cwftmpl), nil
}

// This function "localizes" a template reference to the local scope of the Workflow running it.
// If a template inside a WorkflowTemplate calls another template inside the same WorkflowTemplate, it does so with a local
// "template:" call. However, from the perspective of the original Workflow this is still an external "templateRef:" call.
// In this function we can convert that "local" call found within the WorkflowTemplate, to an "external" call that can be used
// anywhere.
func (ctx *Context) LocalizeTemplateReference(orgTmpl wfv1.TemplateReferenceHolder) wfv1.TemplateReferenceHolder {
	currentTemplateBase := ctx.GetCurrentTemplateBase()
	switch currentTemplateBase.GetResourceScope() {
	case wfv1.ResourceScopeLocal:
		// Context is already local, simply return the template reference as is
		return orgTmpl
	case wfv1.ResourceScopeNamespaced, wfv1.ResourceScopeCluster:
		// We are in an external context, if we are performing a local reference within this external context, localize it
		if orgTmpl.GetTemplateName() != "" {
			return &wfv1.WorkflowStep{
				TemplateRef: &wfv1.TemplateRef{
					Name:         currentTemplateBase.GetName(),
					Template:     orgTmpl.GetTemplateName(),
					ClusterScope: currentTemplateBase.GetResourceScope() == wfv1.ResourceScopeCluster,
				},
			}
		}
		// If we are performing another external reference in this external context, we can simply return it
		return orgTmpl
	default:
		// This should be unreachable
		return orgTmpl
	}
}
