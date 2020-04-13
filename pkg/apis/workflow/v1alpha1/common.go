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
	GetTemplates() []Template
}

// TemplateReferenceHolder is an object that holds a reference to other templates; e.g. WorkflowStep, DAGTask, and NodeStatus
type TemplateReferenceHolder interface {
	GetTemplateName() string
	GetTemplateRef() *TemplateRef
}

// SubmitOpts are workflow submission options
type SubmitOpts struct {
	Name           string                 `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`                      // --name
	GenerateName   string                 `json:"generateName,omitempty" protobuf:"bytes,2,opt,name=generateName"`      // --generate-name
	InstanceID     string                 `json:"instanceID,omitempty" protobuf:"bytes,3,opt,name=instanceID"`          // --instanceid
	Entrypoint     string                 `json:"entryPoint,omitempty" protobuf:"bytes,4,opt,name=entrypoint"`          // --entrypoint
	Parameters     []string               `json:"parameters,omitempty" protobuf:"bytes,5,rep,name=parameters"`          // --parameter
	ParameterFile  string                 `json:"parameterFile,omitempty" protobuf:"bytes,6,opt,name=parameterFile"`    // --parameter-file
	ServiceAccount string                 `json:"serviceAccount,omitempty" protobuf:"bytes,7,opt,name=serviceAccount"`  // --serviceaccount
	DryRun         bool                   `json:"dryRun,omitempty" protobuf:"varint,8,opt,name=dryRun"`                 // --dry-run
	ServerDryRun   bool                   `json:"serverDryRun,omitempty" protobuf:"varint,9,opt,name=serverDryRun"`     // --server-dry-run
	Labels         string                 `json:"labels,omitempty" protobuf:"bytes,10,opt,name=labels"`                 // --labels
	OwnerReference *metav1.OwnerReference `json:"ownerReference,omitempty" protobuf:"bytes,11,opt,name=ownerReference"` // useful if your custom controller creates argo workflow resources
}
