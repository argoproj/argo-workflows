package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TemplateGetter is an interface to get templates.
type TemplateGetter interface {
	GetNamespace() string
	GetName() string
	GroupVersionKind() schema.GroupVersionKind
	GetTemplateByName(name string) *Template
	GetTemplateScope() string
	GetAllTemplates() []Template
}

// TemplateHolder is an interface for holders of templates.
type TemplateHolder interface {
	GetTemplateName() string
	GetTemplateRef() *TemplateRef
	IsResolvable() bool
}

// TemplateStorage is an interface of template storage getter and setter.
type TemplateStorage interface {
	GetStoredTemplate(templateScope string, holder TemplateHolder) *Template
	SetStoredTemplate(templateScope string, holder TemplateHolder, tmpl *Template) (bool, error)
}

// WorkflowTemplateInterface is an simplified TemplateGetter
type WorkflowTemplateInterface interface {
	GetTemplateByName(name string) *Template
	GetTemplateScope() string
}
