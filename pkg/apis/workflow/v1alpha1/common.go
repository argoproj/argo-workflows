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

// TemplateReferenceHolder is an object that holds a reference to other templates; e.g. WorkflowStep, DAGTask, and NodeStatus
type TemplateReferenceHolder interface {
	GetTemplateName() string
	GetTemplateRef() *TemplateRef
}

// SubmitOpts are workflow submission options
type SubmitOpts struct {
	// Name overrides metadata.name in workflow.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// GenerateName overrides metadata.generateName in workflow.
	GenerateName string `json:"generateName,omitempty" protobuf:"bytes,2,opt,name=generateName"`
	// InstanceID adds controller's instance id label in workflow.
	InstanceID string `json:"instanceID,omitempty" protobuf:"bytes,3,opt,name=instanceID"`
	// Entrypoint overrides spec.entrypoint in workflow.
	Entrypoint string `json:"entryPoint,omitempty" protobuf:"bytes,4,opt,name=entrypoint"`
	// Parameters passes input parameters to workflow.
	Parameters []string `json:"parameters,omitempty" protobuf:"bytes,5,rep,name=parameters"`
	// ParameterFile holds parameter file path.
	// This option is not supported in API.
	ParameterFile string `json:"parameterFile,omitempty" protobuf:"bytes,6,opt,name=parameterFile"`
	// ServiceAccount runs all pods in the workflow using specified serviceaccount.
	ServiceAccount string `json:"serviceAccount,omitempty" protobuf:"bytes,7,opt,name=serviceAccount"`
	// DryRun validates the workflow on the client-side without creating it.
	// This option is not supported in API.
	DryRun bool `json:"dryRun,omitempty" protobuf:"varint,8,opt,name=dryRun"`
	// ServerDryRun validates the workflow on the Server-side without creating it.
	ServerDryRun bool `json:"serverDryRun,omitempty" protobuf:"varint,9,opt,name=serverDryRun"`
	// Labels adds metadata.labels in workflow.
	Labels string `json:"labels,omitempty" protobuf:"bytes,10,opt,name=labels"`
	// OwnerReference creates the metadata.ownerReference in workflow.
	OwnerReference *metav1.OwnerReference `json:"ownerReference,omitempty" protobuf:"bytes,11,opt,name=ownerReference"`
}
