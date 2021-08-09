package templateresolution

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	typed "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// maxResolveDepth is the limit of template reference resolution.
const maxResolveDepth int = 10

// workflowTemplateInterfaceWrapper is an internal struct to wrap clientset.
type workflowTemplateInterfaceWrapper struct {
	clientset typed.WorkflowTemplateInterface
}

func WrapWorkflowTemplateInterface(clientset typed.WorkflowTemplateInterface) WorkflowTemplateNamespacedGetter {
	return &workflowTemplateInterfaceWrapper{clientset: clientset}
}

// Get retrieves the WorkflowTemplate of a given name.
func (wrapper *workflowTemplateInterfaceWrapper) Get(name string) (*wfv1.WorkflowTemplate, error) {
	ctx := context.TODO()
	return wrapper.clientset.Get(ctx, name, metav1.GetOptions{})
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

func WrapClusterWorkflowTemplateInterface(clusterClientset typed.ClusterWorkflowTemplateInterface) ClusterWorkflowTemplateGetter {
	return &clusterWorkflowTemplateInterfaceWrapper{clientset: clusterClientset}
}

type NullClusterWorkflowTemplateGetter struct{}

func (n *NullClusterWorkflowTemplateGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return nil, errors.Errorf("", "invalid spec: clusterworkflowtemplates.argoproj.io `%s` is "+
		"forbidden: User cannot get resource 'clusterworkflowtemplates' in API group argoproj.io at the cluster scope", name)
}

// Get retrieves the WorkflowTemplate of a given name.
func (wrapper *clusterWorkflowTemplateInterfaceWrapper) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	ctx := context.TODO()
	return wrapper.clientset.Get(ctx, name, metav1.GetOptions{})
}

// Context is a context of template search.
type Context struct {
	// wftmplGetter is an interface to get WorkflowTemplates.
	wftmplGetter WorkflowTemplateNamespacedGetter
	// cwftmplGetter is an interface to get ClusterWorkflowTemplates
	cwftmplGetter ClusterWorkflowTemplateGetter
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateHolder
	// workflow is the Workflow where templates will be stored
	workflow *wfv1.Workflow
	// log is a logrus entry.
	log *log.Entry
}

// NewContext returns new Context.
func NewContext(wftmplGetter WorkflowTemplateNamespacedGetter, cwftmplGetter ClusterWorkflowTemplateGetter, tmplBase wfv1.TemplateHolder, workflow *wfv1.Workflow) *Context {
	return &Context{
		wftmplGetter:  wftmplGetter,
		cwftmplGetter: cwftmplGetter,
		tmplBase:      tmplBase,
		workflow:      workflow,
		log:           log.WithFields(log.Fields{}),
	}
}

// NewContext returns new Context.
func NewContextFromClientset(wftmplClientset typed.WorkflowTemplateInterface, clusterWftmplClient typed.ClusterWorkflowTemplateInterface, tmplBase wfv1.TemplateHolder, workflow *wfv1.Workflow) *Context {
	return &Context{
		wftmplGetter:  WrapWorkflowTemplateInterface(wftmplClientset),
		cwftmplGetter: WrapClusterWorkflowTemplateInterface(clusterWftmplClient),
		tmplBase:      tmplBase,
		workflow:      workflow,
		log:           log.WithFields(log.Fields{}),
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
func (ctx *Context) GetTemplate(h wfv1.TemplateReferenceHolder) (*wfv1.Template, error) {
	ctx.log.Debug("Getting the template")
	if x := h.GetTemplate(); x != nil {
		return x, nil
	} else if x := h.GetTemplateRef(); x != nil {
		return ctx.GetTemplateFromRef(x)
	} else if x := h.GetTemplateName(); x != "" {
		return ctx.GetTemplateByName(x)
	}
	return nil, errors.Errorf(errors.CodeInternal, "failed to get a template")
}

// GetCurrentTemplateBase returns the current template base of the context.
func (ctx *Context) GetCurrentTemplateBase() wfv1.TemplateHolder {
	return ctx.tmplBase
}

func (ctx *Context) GetTemplateScope() string {
	return string(ctx.tmplBase.GetResourceScope()) + "/" + ctx.tmplBase.GetName()
}

// ResolveTemplate digs into referenes and returns a merged template.
// This method is the public start point of template resolution.
func (ctx *Context) ResolveTemplate(tmplHolder wfv1.TemplateReferenceHolder) (*Context, *wfv1.Template, bool, error) {
	return ctx.resolveTemplateImpl(tmplHolder, 0)
}

// resolveTemplateImpl digs into references and returns a merged template.
// This method processes inputs and arguments so the inputs of the final
//  resolved template include intermediate parameter passing.
// The other fields are just merged and shallower templates overwrite deeper.
func (ctx *Context) resolveTemplateImpl(tmplHolder wfv1.TemplateReferenceHolder, depth int) (*Context, *wfv1.Template, bool, error) {
	ctx.log = ctx.log.WithFields(log.Fields{
		"depth": depth,
		"base":  common.GetTemplateGetterString(ctx.tmplBase),
		"tmpl":  common.GetTemplateHolderString(tmplHolder),
	})
	// Avoid infinite references
	if depth > maxResolveDepth {
		return nil, nil, false, errors.Errorf(errors.CodeBadRequest, "template reference exceeded max depth (%d)", maxResolveDepth)
	}

	ctx.log.Debug("Resolving the template")

	templateStored := false
	var tmpl *wfv1.Template
	if ctx.workflow != nil {
		// Check if the template has been stored.
		scope := ctx.tmplBase.GetResourceScope()
		resourceName := ctx.tmplBase.GetName()
		tmpl = ctx.workflow.GetStoredTemplate(scope, resourceName, tmplHolder)
	}
	if tmpl != nil {
		ctx.log.Debug("Found stored template")
	} else {
		// Find newly appeared template.
		newTmpl, err := ctx.GetTemplate(tmplHolder)
		if err != nil {
			return nil, nil, false, err
		}
		// Stored the found template.
		if ctx.workflow != nil {
			scope := ctx.tmplBase.GetResourceScope()
			resourceName := ctx.tmplBase.GetName()
			stored, err := ctx.workflow.SetStoredTemplate(scope, resourceName, tmplHolder, newTmpl)
			if err != nil {
				return nil, nil, false, err
			}
			if stored {
				ctx.log.Debug("Stored the template")
				templateStored = true
			}
		}
		tmpl = newTmpl
	}

	// Update the template base of the context.
	newTmplCtx, err := ctx.WithTemplateHolder(tmplHolder)
	if err != nil {
		return nil, nil, false, err
	}

	if tmpl.GetType() == wfv1.TemplateTypeUnknown {
		return nil, nil, false, fmt.Errorf("template '%s' type is unknown", tmpl.Name)
	}

	return newTmplCtx, tmpl, templateStored, nil
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
	return NewContext(ctx.wftmplGetter, ctx.cwftmplGetter, tmplBase, ctx.workflow)
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
