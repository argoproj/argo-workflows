package templateresolution

import (
	"context"
	"fmt"
	"maps"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	typed "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	listers "github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// workflowTemplateInterfaceWrapper is an internal struct to wrap clientset.
type workflowTemplateInterfaceWrapper struct {
	clientset typed.WorkflowTemplateInterface
}

func WrapWorkflowTemplateInterface(clientset typed.WorkflowTemplateInterface) WorkflowTemplateNamespacedGetter {
	return &workflowTemplateInterfaceWrapper{clientset: clientset}
}

// Get retrieves the WorkflowTemplate of a given name.
func (wrapper *workflowTemplateInterfaceWrapper) Get(ctx context.Context, name string) (*wfv1.WorkflowTemplate, error) {
	return wrapper.clientset.Get(ctx, name, metav1.GetOptions{})
}

// WorkflowTemplateNamespacedGetter helps get WorkflowTemplates.
type WorkflowTemplateNamespacedGetter interface {
	// Get retrieves the WorkflowTemplate from the indexer for a given name.
	Get(ctx context.Context, name string) (*wfv1.WorkflowTemplate, error)
}

// clusterWorkflowTemplateInterfaceWrapper is an internal struct to wrap clientset.
type clusterWorkflowTemplateInterfaceWrapper struct {
	clientset typed.ClusterWorkflowTemplateInterface
}

// ClusterWorkflowTemplateGetter helps get WorkflowTemplates.
type ClusterWorkflowTemplateGetter interface {
	// Get retrieves the WorkflowTemplate from the indexer for a given name.
	Get(ctx context.Context, name string) (*wfv1.ClusterWorkflowTemplate, error)
}

func WrapClusterWorkflowTemplateInterface(clusterClientset typed.ClusterWorkflowTemplateInterface) ClusterWorkflowTemplateGetter {
	return &clusterWorkflowTemplateInterfaceWrapper{clientset: clusterClientset}
}

type NullClusterWorkflowTemplateGetter struct{}

func (n *NullClusterWorkflowTemplateGetter) Get(_ context.Context, name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return nil, errors.Errorf("", "invalid spec: clusterworkflowtemplates.argoproj.io `%s` is "+
		"forbidden: User cannot get resource 'clusterworkflowtemplates' in API group argoproj.io at the cluster scope", name)
}

type NullWorkflowTemplateNamespacedGetter struct{}

func (n *NullWorkflowTemplateNamespacedGetter) Get(_ context.Context, name string) (*wfv1.WorkflowTemplate, error) {
	return nil, errors.Errorf("", "invalid spec: workflowtemplates.argoproj.io `%s` is "+
		"forbidden: User cannot get resource 'workflowtemplates' in API group argoproj.io at the namespace scope", name)
}

// Get retrieves the WorkflowTemplate of a given name.
func (wrapper *clusterWorkflowTemplateInterfaceWrapper) Get(ctx context.Context, name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return wrapper.clientset.Get(ctx, name, metav1.GetOptions{})
}

// TemplateContext is a context of template search.
type TemplateContext struct {
	// wftmplGetter is an interface to get WorkflowTemplates.
	wftmplGetter WorkflowTemplateNamespacedGetter
	// cwftmplGetter is an interface to get ClusterWorkflowTemplates
	cwftmplGetter ClusterWorkflowTemplateGetter
	// tmplBase is the base of local template search.
	tmplBase wfv1.TemplateHolder
	// workflow is the Workflow where templates will be stored
	workflow *wfv1.Workflow
	// log is a logging entry.
	log logging.Logger
}

// NewContext returns new Context.
func NewContext(wftmplGetter WorkflowTemplateNamespacedGetter, cwftmplGetter ClusterWorkflowTemplateGetter, tmplBase wfv1.TemplateHolder, workflow *wfv1.Workflow, log logging.Logger) *TemplateContext {
	return &TemplateContext{
		wftmplGetter:  wftmplGetter,
		cwftmplGetter: cwftmplGetter,
		tmplBase:      tmplBase,
		workflow:      workflow,
		log:           log,
	}
}

// NewContextFromClientSet returns new Context.
func NewContextFromClientSet(wftmplClientset typed.WorkflowTemplateInterface, clusterWftmplClient typed.ClusterWorkflowTemplateInterface, tmplBase wfv1.TemplateHolder, workflow *wfv1.Workflow, log logging.Logger) *TemplateContext {
	return &TemplateContext{
		wftmplGetter:  WrapWorkflowTemplateInterface(wftmplClientset),
		cwftmplGetter: WrapClusterWorkflowTemplateInterface(clusterWftmplClient),
		tmplBase:      tmplBase,
		workflow:      workflow,
		log:           log,
	}
}

// GetTemplateByName returns a template by name in the context.
func (tplCtx *TemplateContext) GetTemplateByName(ctx context.Context, name string) (*wfv1.Template, error) {
	tplCtx.log.WithField("name", name).Debug(ctx, "Getting the template by name")

	tmpl := tplCtx.tmplBase.GetTemplateByName(name)
	if tmpl == nil {
		return nil, errors.Errorf(errors.CodeNotFound, "template %s not found", name)
	}

	podMetadata := tplCtx.tmplBase.GetPodMetadata()
	tplCtx.addPodMetadata(podMetadata, tmpl)

	return tmpl.DeepCopy(), nil
}

func (tplCtx *TemplateContext) GetTemplateGetterFromRef(ctx context.Context, tmplRef *wfv1.TemplateRef) (wfv1.TemplateHolder, error) {
	if tmplRef.ClusterScope {
		return tplCtx.cwftmplGetter.Get(ctx, tmplRef.Name)
	}
	return tplCtx.wftmplGetter.Get(ctx, tmplRef.Name)
}

// GetTemplateFromRef returns a template found by a given template ref.
func (tplCtx *TemplateContext) GetTemplateFromRef(ctx context.Context, tmplRef *wfv1.TemplateRef) (*wfv1.Template, error) {
	tplCtx.log.Debug(ctx, "Getting the template from ref")
	var template *wfv1.Template
	var wftmpl wfv1.TemplateHolder
	var err error
	if tmplRef.ClusterScope {
		wftmpl, err = tplCtx.cwftmplGetter.Get(ctx, tmplRef.Name)
	} else {
		wftmpl, err = tplCtx.wftmplGetter.Get(ctx, tmplRef.Name)
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

	podMetadata := wftmpl.GetPodMetadata()
	tplCtx.addPodMetadata(podMetadata, template)

	return template.DeepCopy(), nil
}

// GetTemplate returns a template found by template name or template ref.
func (tplCtx *TemplateContext) GetTemplate(ctx context.Context, h wfv1.TemplateReferenceHolder) (*wfv1.Template, error) {
	tplCtx.log.Debug(ctx, "Getting the template")
	if x := h.GetTemplate(); x != nil {
		return x, nil
	} else if x := h.GetTemplateRef(); x != nil {
		return tplCtx.GetTemplateFromRef(ctx, x)
	} else if x := h.GetTemplateName(); x != "" {
		return tplCtx.GetTemplateByName(ctx, x)
	}
	return nil, errors.Errorf(errors.CodeInternal, "failed to get a template")
}

// GetCurrentTemplateBase returns the current template base of the context.
func (tplCtx *TemplateContext) GetCurrentTemplateBase() wfv1.TemplateHolder {
	return tplCtx.tmplBase
}

func (tplCtx *TemplateContext) GetTemplateScope() string {
	return string(tplCtx.tmplBase.GetResourceScope()) + "/" + tplCtx.tmplBase.GetName()
}

// ResolveTemplate digs into referenes and returns a merged template.
// This method is the public start point of template resolution.
func (tplCtx *TemplateContext) ResolveTemplate(c context.Context, tmplHolder wfv1.TemplateReferenceHolder) (*TemplateContext, *wfv1.Template, bool, error) {
	return tplCtx.resolveTemplateImpl(c, tmplHolder)
}

// resolveTemplateImpl digs into references and returns a merged template.
// This method processes inputs and arguments so the inputs of the final
// resolved template include intermediate parameter passing.
// The other fields are just merged and shallower templates overwrite deeper.
func (tplCtx *TemplateContext) resolveTemplateImpl(ctx context.Context, tmplHolder wfv1.TemplateReferenceHolder) (*TemplateContext, *wfv1.Template, bool, error) {
	tplCtx.log = tplCtx.log.WithFields(logging.Fields{
		"base": common.GetTemplateGetterString(tplCtx.tmplBase),
		"tmpl": common.GetTemplateHolderString(tmplHolder),
	})
	tplCtx.log.Debug(ctx, "Resolving the template")

	templateStored := false
	var tmpl *wfv1.Template
	if tplCtx.workflow != nil {
		// Check if the template has been stored.
		scope := tplCtx.tmplBase.GetResourceScope()
		resourceName := tplCtx.tmplBase.GetName()
		tmpl = tplCtx.workflow.GetStoredTemplate(scope, resourceName, tmplHolder)
	}
	if tmpl != nil {
		tplCtx.log.Debug(ctx, "Found stored template")
	} else {
		// Find newly appeared template.
		newTmpl, err := tplCtx.GetTemplate(ctx, tmplHolder)
		if err != nil {
			return nil, nil, false, err
		}
		// Stored the found template.
		if tplCtx.workflow != nil {
			scope := tplCtx.tmplBase.GetResourceScope()
			resourceName := tplCtx.tmplBase.GetName()
			stored, err := tplCtx.workflow.SetStoredTemplate(scope, resourceName, tmplHolder, newTmpl)
			if err != nil {
				return nil, nil, false, err
			}
			if stored {
				tplCtx.log.Debug(ctx, "Stored the template")
				templateStored = true
			}
			err = tplCtx.workflow.SetStoredInlineTemplate(scope, resourceName, newTmpl)
			if err != nil {
				tplCtx.log.WithError(err).Error(ctx, "Failed to store the inline template")
			}
		}
		tmpl = newTmpl
	}

	// Update the template base of the context.
	newTmplCtx, err := tplCtx.WithTemplateHolder(ctx, tmplHolder)
	if err != nil {
		return nil, nil, false, err
	}

	if tmpl.GetType() == wfv1.TemplateTypeUnknown {
		return nil, nil, false, fmt.Errorf("template '%s' type is unknown", tmpl.Name)
	}

	return newTmplCtx, tmpl, templateStored, nil
}

// WithTemplateHolder creates new context with a template base of a given template holder.
func (tplCtx *TemplateContext) WithTemplateHolder(ctx context.Context, tmplHolder wfv1.TemplateReferenceHolder) (*TemplateContext, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		tmplName := tmplRef.Name
		if tmplRef.ClusterScope {
			return tplCtx.WithClusterWorkflowTemplate(ctx, tmplName)
		} else {
			return tplCtx.WithWorkflowTemplate(ctx, tmplName)
		}
	}
	return tplCtx.WithTemplateBase(tplCtx.tmplBase), nil
}

// WithTemplateBase creates new context with a wfv1.TemplateHolder.
func (tplCtx *TemplateContext) WithTemplateBase(tmplBase wfv1.TemplateHolder) *TemplateContext {
	return NewContext(tplCtx.wftmplGetter, tplCtx.cwftmplGetter, tmplBase, tplCtx.workflow, tplCtx.log)
}

// WithWorkflowTemplate creates new context with a wfv1.TemplateHolder.
func (tplCtx *TemplateContext) WithWorkflowTemplate(ctx context.Context, name string) (*TemplateContext, error) {
	wftmpl, err := tplCtx.wftmplGetter.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	return tplCtx.WithTemplateBase(wftmpl), nil
}

// WithClusterWorkflowTemplate creates new context with a wfv1.TemplateHolder.
func (tplCtx *TemplateContext) WithClusterWorkflowTemplate(ctx context.Context, name string) (*TemplateContext, error) {
	cwftmpl, err := tplCtx.cwftmplGetter.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	return tplCtx.WithTemplateBase(cwftmpl), nil
}

// addPodMetadata add podMetadata in workflow template level to template
func (tplCtx *TemplateContext) addPodMetadata(podMetadata *wfv1.Metadata, tmpl *wfv1.Template) {
	if podMetadata != nil {
		if tmpl.Metadata.Annotations == nil {
			tmpl.Metadata.Annotations = make(map[string]string)
		}
		maps.Copy(tmpl.Metadata.Annotations, podMetadata.Annotations)
		if tmpl.Metadata.Labels == nil {
			tmpl.Metadata.Labels = make(map[string]string)
		}
		maps.Copy(tmpl.Metadata.Labels, podMetadata.Labels)
	}
}

// Wrapper types for lister interfaces to adapt them to the new interface signatures

type workflowTemplateListerWrapper struct {
	lister listers.WorkflowTemplateNamespaceLister
}

func WrapWorkflowTemplateLister(lister listers.WorkflowTemplateNamespaceLister) WorkflowTemplateNamespacedGetter {
	return &workflowTemplateListerWrapper{lister: lister}
}

func (w *workflowTemplateListerWrapper) Get(ctx context.Context, name string) (*wfv1.WorkflowTemplate, error) {
	return w.lister.Get(name)
}

type clusterWorkflowTemplateListerWrapper struct {
	lister listers.ClusterWorkflowTemplateLister
}

func WrapClusterWorkflowTemplateLister(lister listers.ClusterWorkflowTemplateLister) ClusterWorkflowTemplateGetter {
	return &clusterWorkflowTemplateListerWrapper{lister: lister}
}

func (w *clusterWorkflowTemplateListerWrapper) Get(ctx context.Context, name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return w.lister.Get(name)
}
