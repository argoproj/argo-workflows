package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceScope string

const (
	ResourceScopeLocal                   ResourceScope = "Local"
	ResourceScopeWorkflowTemplate        ResourceScope = "WorkflowTemplate"
	ResourceScopeClusterWorkflowTemplate ResourceScope = "ClusterWorkflowTemplate"
)

// TemplateGetter is an interface to get templates.
type TemplateGetter interface {
	GetNamespace() string
	GetName() string
	GroupVersionKind() schema.GroupVersionKind
	GetTemplateByName(name string) *Template
	GetTemplateScope() (ResourceScope, string)
	GetAllTemplates() []Template
}

// TemplateCaller is an object that can call other templates
type TemplateCaller interface {
	GetTemplateName() string
	GetTemplateRef() *TemplateRef
}

// TemplateStorage is an interface of template storage getter and setter.
type TemplateStorage interface {
	GetStoredTemplate(scope ResourceScope, resourceName string, caller TemplateCaller) *Template
	SetStoredTemplate(scope ResourceScope, resourceName string, caller TemplateCaller, tmpl *Template) (bool, error)
}
