package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceScope string

const (
	ResourceScopeLocal      ResourceScope = "local"
	ResourceScopeNamespaced ResourceScope = "namespaced"
	ResourceScopeCluster    ResourceScope = "cluster"
)

// TemplateHolder is an object that holds templates; e.g. Workflow, WorkflowTemplate, and ClusterWorkflowTemplate
type TemplateHolder interface {
	GetNamespace() string
	GetName() string
	GroupVersionKind() schema.GroupVersionKind
	GetTemplateByName(name string) *Template
	GetResourceScope() ResourceScope
}

// WorkflowSpecHolder is an object that holds a WorkflowSpec; e.g., WorkflowTemplate, and ClusterWorkflowTemplate
type WorkflowSpecHolder interface {
	GetWorkflowSpec() *WorkflowSpec
}

// TemplateReferenceHolder is an object that holds a reference to other templates; e.g. WorkflowStep, DAGTask, and NodeStatus
type TemplateReferenceHolder interface {
	GetTemplateName() string
	GetTemplateRef() *TemplateRef
}

// SubmitOpts are workflow submission options
type SubmitOpts struct {
	// Name overrides metadata.name
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// GenerateName overrides metadata.generateName
	GenerateName string `json:"generateName,omitempty" protobuf:"bytes,2,opt,name=generateName"`
	// Entrypoint overrides spec.entrypoint
	Entrypoint string `json:"entryPoint,omitempty" protobuf:"bytes,4,opt,name=entrypoint"`
	// Parameters passes input parameters to workflow
	Parameters []string `json:"parameters,omitempty" protobuf:"bytes,5,rep,name=parameters"`
	// ParameterFile holds a reference to a parameter file. This option is not supported in API
	ParameterFile string `json:"parameterFile,omitempty" protobuf:"bytes,6,opt,name=parameterFile"`
	// ServiceAccount runs all pods in the workflow using specified ServiceAccount.
	ServiceAccount string `json:"serviceAccount,omitempty" protobuf:"bytes,7,opt,name=serviceAccount"`
	// DryRun validates the workflow on the client-side without creating it. This option is not supported in API
	DryRun bool `json:"dryRun,omitempty" protobuf:"varint,8,opt,name=dryRun"`
	// ServerDryRun validates the workflow on the server-side without creating it
	ServerDryRun bool `json:"serverDryRun,omitempty" protobuf:"varint,9,opt,name=serverDryRun"`
	// Labels adds to metadata.labels
	Labels string `json:"labels,omitempty" protobuf:"bytes,10,opt,name=labels"`
	// OwnerReference creates a metadata.ownerReference
	OwnerReference *metav1.OwnerReference `json:"ownerReference,omitempty" protobuf:"bytes,11,opt,name=ownerReference"`
}
