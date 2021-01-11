// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// WorkflowTemplateLister helps list WorkflowTemplates.
// All objects returned here must be treated as read-only.
type WorkflowTemplateLister interface {
	// List lists all WorkflowTemplates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.WorkflowTemplate, err error)
	// WorkflowTemplates returns an object that can list and get WorkflowTemplates.
	WorkflowTemplates(namespace string) WorkflowTemplateNamespaceLister
	WorkflowTemplateListerExpansion
}

// workflowTemplateLister implements the WorkflowTemplateLister interface.
type workflowTemplateLister struct {
	indexer cache.Indexer
}

// NewWorkflowTemplateLister returns a new WorkflowTemplateLister.
func NewWorkflowTemplateLister(indexer cache.Indexer) WorkflowTemplateLister {
	return &workflowTemplateLister{indexer: indexer}
}

// List lists all WorkflowTemplates in the indexer.
func (s *workflowTemplateLister) List(selector labels.Selector) (ret []*v1alpha1.WorkflowTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.WorkflowTemplate))
	})
	return ret, err
}

// WorkflowTemplates returns an object that can list and get WorkflowTemplates.
func (s *workflowTemplateLister) WorkflowTemplates(namespace string) WorkflowTemplateNamespaceLister {
	return workflowTemplateNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// WorkflowTemplateNamespaceLister helps list and get WorkflowTemplates.
// All objects returned here must be treated as read-only.
type WorkflowTemplateNamespaceLister interface {
	// List lists all WorkflowTemplates in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.WorkflowTemplate, err error)
	// Get retrieves the WorkflowTemplate from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.WorkflowTemplate, error)
	WorkflowTemplateNamespaceListerExpansion
}

// workflowTemplateNamespaceLister implements the WorkflowTemplateNamespaceLister
// interface.
type workflowTemplateNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all WorkflowTemplates in the indexer for a given namespace.
func (s workflowTemplateNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.WorkflowTemplate, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.WorkflowTemplate))
	})
	return ret, err
}

// Get retrieves the WorkflowTemplate from the indexer for a given namespace and name.
func (s workflowTemplateNamespaceLister) Get(name string) (*v1alpha1.WorkflowTemplate, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("workflowtemplate"), name)
	}
	return obj.(*v1alpha1.WorkflowTemplate), nil
}
