package v1alpha1

import (
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
