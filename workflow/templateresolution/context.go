package templateresolution

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
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

type NullClusterWorkflowTemplateGetter struct{}

func (n *NullClusterWorkflowTemplateGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return nil, errors.Errorf("", "invalid spec: clusterworkflowtemplates.argoproj.io `%s` is "+
		"forbidden: User cannot get resource 'clusterworkflowtemplates' in API group argoproj.io at the cluster scope", name)
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
	// workflow is the Workflow where templates will be stored
	workflow *wfv1.Workflow
	// log is a logrus entry.
	log *logrus.Entry

	// #3707: inherit some father template values
	inheritedTemplate *InheritedMembers
}

type InheritedMembers struct {
	// NodeSelector is a selector to schedule this step of the workflow to be
	// run on the selected node(s). Overrides the selector set at the workflow level.
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,opt,name=nodeSelector"`

	// Affinity sets the pod's scheduling constraints
	// Overrides the affinity set at the workflow level (if any)
	Affinity *apiv1.Affinity `json:"affinity,omitempty" protobuf:"bytes,8,opt,name=affinity"`

	// Volumes is a list of volumes that can be mounted by containers in a template.
	// +patchStrategy=merge
	// +patchMergeKey=name
	Volumes []apiv1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,17,opt,name=volumes"`

	// InitContainers is a list of containers which run before the main container.
	// +patchStrategy=merge
	// +patchMergeKey=name
	InitContainers []wfv1.UserContainer `json:"initContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,18,opt,name=initContainers"`

	// Sidecars is a list of containers which run alongside the main container
	// Sidecars are automatically killed when the main container completes
	// +patchStrategy=merge
	// +patchMergeKey=name
	Sidecars []wfv1.UserContainer `json:"sidecars,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,19,opt,name=sidecars"`

	// Location in which all files related to the step will be stored (logs, artifacts, etc...).
	// Can be overridden by individual items in Outputs. If omitted, will use the default
	// artifact repository location configured in the controller, appended with the
	// <workflowname>/<nodename> in the key.
	ArchiveLocation *wfv1.ArtifactLocation `json:"archiveLocation,omitempty" protobuf:"bytes,20,opt,name=archiveLocation"`

	// Optional duration in seconds relative to the StartTime that the pod may be active on a node
	// before the system actively tries to terminate the pod; value must be positive integer
	// This field is only applicable to container and script templates.
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"bytes,21,opt,name=activeDeadlineSeconds"`

	// Tolerations to apply to workflow pods.
	// +patchStrategy=merge
	// +patchMergeKey=key
	Tolerations []apiv1.Toleration `json:"tolerations,omitempty" patchStrategy:"merge" patchMergeKey:"key" protobuf:"bytes,24,opt,name=tolerations"`

	// If specified, the pod will be dispatched by specified scheduler.
	// Or it will be dispatched by workflow scope scheduler if specified.
	// If neither specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,25,opt,name=schedulerName"`
}

// NewContext returns new Context.
func NewContext(wftmplGetter WorkflowTemplateNamespacedGetter, cwftmplGetter ClusterWorkflowTemplateGetter, tmplBase wfv1.TemplateHolder, workflow *wfv1.Workflow, opts ...Option) *Context {
	ctx := &Context{
		wftmplGetter:  wftmplGetter,
		cwftmplGetter: cwftmplGetter,
		tmplBase:      tmplBase,
		workflow:      workflow,
		log:           log.WithFields(logrus.Fields{}),
	}
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

type Option func(*Context)

func WithInheritedMembers(inherited *InheritedMembers) Option {
	return func(ctx *Context) {
		ctx.inheritedTemplate = inherited
	}
}
func WithTemplate(tmpl *wfv1.Template) Option {
	return func(ctx *Context) {
		ctx.inheritedTemplate = mergeTemplate(tmpl, ctx.inheritedTemplate)
	}
}

func mergeTemplate(tmpl *wfv1.Template, inherited *InheritedMembers) *InheritedMembers {
	newInherited := &InheritedMembers{}
	bytes, _ := json.Marshal(*tmpl)
	json.Unmarshal(bytes, newInherited)
	if inherited == nil {
		return newInherited
	}
	if len(newInherited.NodeSelector) == 0 {
		m := make(map[string]string, len(inherited.NodeSelector))
		for k, v := range inherited.NodeSelector {
			m[k] = v
		}
		newInherited.NodeSelector = m
	}
	if newInherited.Affinity == nil && inherited.Affinity != nil {
		newInherited.Affinity = inherited.Affinity.DeepCopy()
	}
	if len(newInherited.Volumes) == 0 {
		volumes := make([]apiv1.Volume, len(inherited.Volumes))
		copy(volumes, inherited.Volumes)
		newInherited.Volumes = volumes
	}
	if len(newInherited.InitContainers) == 0 {
		containers := make([]wfv1.UserContainer, len(inherited.InitContainers))
		copy(containers, inherited.InitContainers)
		newInherited.InitContainers = containers
	}
	if len(newInherited.Sidecars) == 0 {
		containers := make([]wfv1.UserContainer, len(inherited.Sidecars))
		copy(containers, inherited.Sidecars)
		newInherited.Sidecars = containers
	}
	if newInherited.ArchiveLocation == nil && inherited.ArchiveLocation != nil {
		newInherited.ArchiveLocation = inherited.ArchiveLocation.DeepCopy()
	}
	if newInherited.ActiveDeadlineSeconds == nil && inherited.ActiveDeadlineSeconds != nil {
		v := *inherited.ActiveDeadlineSeconds
		newInherited.ActiveDeadlineSeconds = &v
	}
	if len(newInherited.Tolerations) == 0 {
		tolerations := make([]apiv1.Toleration, len(inherited.Tolerations))
		copy(tolerations, inherited.Tolerations)
		newInherited.Tolerations = tolerations
	}
	if newInherited.SchedulerName == "" {
		newInherited.SchedulerName = inherited.SchedulerName
	}

	return newInherited
}

// NewContext returns new Context.
func NewContextFromClientset(wftmplClientset typed.WorkflowTemplateInterface, clusterWftmplClient typed.ClusterWorkflowTemplateInterface, tmplBase wfv1.TemplateHolder, workflow *wfv1.Workflow) *Context {
	return &Context{
		wftmplGetter:  WrapWorkflowTemplateInterface(wftmplClientset),
		cwftmplGetter: WrapClusterWorkflowTemplateInterface(clusterWftmplClient),
		tmplBase:      tmplBase,
		workflow:      workflow,
		log:           log.WithFields(logrus.Fields{}),
	}
}
func (ctx *Context) applyInheritedMembers(tmpl *wfv1.Template) {
	bytes, _ := json.Marshal(*ctx.inheritedTemplate)
	json.Unmarshal(bytes, tmpl)
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

func (ctx *Context) GetTemplateScope() string {
	return string(ctx.tmplBase.GetResourceScope()) + "/" + ctx.tmplBase.GetName()
}

// ResolveTemplate digs into referenes and returns a merged template.
// This method is the public start point of template resolution.
func (ctx *Context) ResolveTemplate(tmplHolder wfv1.TemplateReferenceHolder) (*Context, *wfv1.Template, bool, error) {
	return ctx.resolveTemplateImpl(tmplHolder, 0)
}

// resolveTemplateImpl digs into referenes and returns a merged template.
// This method processes inputs and arguments so the inputs of the final
//  resolved template include intermediate parameter passing.
// The other fields are just merged and shallower templates overwrite deeper.
func (ctx *Context) resolveTemplateImpl(tmplHolder wfv1.TemplateReferenceHolder, depth int) (*Context, *wfv1.Template, bool, error) {
	ctx.log = ctx.log.WithFields(logrus.Fields{
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
	newTmplCtx, err := ctx.WithTemplateHolder(tmplHolder, WithTemplate(tmpl.DeepCopy()))
	if err != nil {
		return nil, nil, false, err
	}

	// Return a concrete template without digging into it.
	if tmpl.GetType() != wfv1.TemplateTypeUnknown {
		newTmplCtx.applyInheritedMembers(tmpl)
		return newTmplCtx, tmpl, templateStored, nil
	}

	// Dig into nested references with new template base.
	finalTmplCtx, resolvedTmpl, templateStoredInCall, err := newTmplCtx.resolveTemplateImpl(tmpl, depth+1)
	if err != nil {
		return nil, nil, false, err
	}
	if templateStoredInCall {
		templateStored = true
	}

	// Merge the referred template into the original.
	mergedTmpl, err := common.MergeReferredTemplate(tmpl, resolvedTmpl)
	if err != nil {
		return nil, nil, false, err
	}
	finalTmplCtx.applyInheritedMembers(mergedTmpl)

	return finalTmplCtx, mergedTmpl, templateStored, nil
}

// WithTemplateHolder creates new context with a template base of a given template holder.
func (ctx *Context) WithTemplateHolder(tmplHolder wfv1.TemplateReferenceHolder, opts ...Option) (*Context, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		tmplName := tmplRef.Name
		if tmplRef.ClusterScope {
			return ctx.WithClusterWorkflowTemplate(tmplName, opts...)
		} else {
			return ctx.WithWorkflowTemplate(tmplName, opts...)
		}
	}
	return ctx.WithTemplateBase(ctx.tmplBase, opts...), nil
}

// WithTemplateBase creates new context with a wfv1.TemplateHolder.
func (ctx *Context) WithTemplateBase(tmplBase wfv1.TemplateHolder, opts ...Option) *Context {
	newOpts := []Option{WithInheritedMembers(ctx.inheritedTemplate)}
	return NewContext(ctx.wftmplGetter, ctx.cwftmplGetter, tmplBase, ctx.workflow, append(newOpts, opts...)...)
}

// WithWorkflowTemplate creates new context with a wfv1.TemplateHolder.
func (ctx *Context) WithWorkflowTemplate(name string, opts ...Option) (*Context, error) {
	wftmpl, err := ctx.wftmplGetter.Get(name)
	if err != nil {
		return nil, err
	}
	return ctx.WithTemplateBase(wftmpl, opts...), nil
}

// WithWorkflowTemplate creates new context with a wfv1.TemplateHolder.
func (ctx *Context) WithClusterWorkflowTemplate(name string, opts ...Option) (*Context, error) {
	cwftmpl, err := ctx.cwftmplGetter.Get(name)
	if err != nil {
		return nil, err
	}
	return ctx.WithTemplateBase(cwftmpl, opts...), nil
}
