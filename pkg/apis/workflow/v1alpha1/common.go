package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceScope string

const (
	ResourceScopeLocal      ResourceScope = "Local"
	ResourceScopeNamespaced ResourceScope = "Namespaced"
	ResourceScopeCluster    ResourceScope = "Cluster"
)

// TemplateHolder is an object that holds templates; e.g. Workflow, WorkflowTemplate, and ClusterWorkflowTemplate
type TemplateHolder interface {
	GetNamespace() string
	GetName() string
	GroupVersionKind() schema.GroupVersionKind
	GetTemplateByName(name string) *Template
	GetResourceScope() ResourceScope
	GetAllTemplates() []Template
}

// TemplateReferenceHolder is an object that holds a reference to other templates; e.g. WorkflowStep, DAGTask, and NodeStatus
type TemplateReferenceHolder interface {
	GetTemplateName() string
	GetTemplateRef() *TemplateRef
}

// TemplateStorage is an interface of template storage getter and setter.
type TemplateStorage interface {
	GetStoredTemplate(scope ResourceScope, resourceName string, caller TemplateReferenceHolder) *Template
	SetStoredTemplate(scope ResourceScope, resourceName string, caller TemplateReferenceHolder, tmpl *Template) (bool, error)
}
