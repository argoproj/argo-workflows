package cache

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	extwfv1 "github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTemplateInformer struct {
	delegate informers.GenericInformer
}

var _ extwfv1.WorkflowTemplateInformer = &tolerantWorkflowTemplateInformer{}

type tolerantWorkflowTemplateLister struct {
	delegate cache.GenericLister
}

func (t *tolerantWorkflowTemplateLister) List(selector v1Label.Selector) ([]*wfv1.WorkflowTemplate, error) {
	list, err := t.delegate.List(selector)
	return objectsToWorkflowTemplates(list), err
}

type tolerantWorkflowTemplateNamespaceLister struct {
	delegate cache.GenericNamespaceLister
}

func (t *tolerantWorkflowTemplateNamespaceLister) Get(name string) (*wfv1.WorkflowTemplate, error) {
	object, err := t.delegate.Get(name)
	if err != nil {
		return nil, err
	}
	return objectToWorkflowTemplate(object)
}

func (t *tolerantWorkflowTemplateNamespaceLister) List(selector v1Label.Selector) ([]*wfv1.WorkflowTemplate, error) {
	list, err := t.delegate.List(selector)
	return objectsToWorkflowTemplates(list), err
}

func (t tolerantWorkflowTemplateLister) WorkflowTemplates(namespace string) v1alpha1.WorkflowTemplateNamespaceLister {
	return &tolerantWorkflowTemplateNamespaceLister{delegate: t.delegate.ByNamespace(namespace)}
}

func (t tolerantWorkflowTemplateInformer) Informer() cache.SharedIndexInformer {
	return t.delegate.Informer()
}

func (t tolerantWorkflowTemplateInformer) Lister() v1alpha1.WorkflowTemplateLister {
	return &tolerantWorkflowTemplateLister{delegate: t.delegate.Lister()}
}

func NewTolerantWorkflowTemplateInformer(dynamicInterface dynamic.Interface, defaultResync time.Duration, namespace, resource string) *tolerantWorkflowTemplateInformer {
	return &tolerantWorkflowTemplateInformer{delegate: dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicInterface, defaultResync, namespace, func(options *metav1.ListOptions) {}).
		ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: "v1alpha1", Resource: resource})}
}

func objectToWorkflowTemplate(object runtime.Object) (*wfv1.WorkflowTemplate, error) {
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to unstructured")
	}
	v := &wfv1.WorkflowTemplate{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	return v, err
}

func objectsToWorkflowTemplates(list []runtime.Object) []*wfv1.WorkflowTemplate {
	ret := make([]*wfv1.WorkflowTemplate, 0)
	for _, object := range list {
		v, err := objectToWorkflowTemplate(object)
		if err != nil {
			log.WithError(err).Error("failed convert unstructured workflow template")
			continue
		}
		ret = append(ret, v)
	}
	return ret
}
