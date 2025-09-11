package informer

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func objectToWorkflowTemplate(object runtime.Object) (*wfv1.WorkflowTemplate, error) {
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("malformed workflow template: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(object).String())
	}
	wftmplRaw, ok := un.Object["workflowTemplate"]
	if !ok {
		return nil, fmt.Errorf("malformed workflow template: expected \"workflowTemplate\" key in unstructured object, got \"%s\"", reflect.TypeOf(object).String())
	}
	wftmpl, ok := wftmplRaw.(*wfv1.WorkflowTemplate)
	if ok {
		return wftmpl, nil
	}
	return nil, fmt.Errorf("malformed workflow template: expected \"*wfv1.WorkflowTemplate\", got \"%s\"", reflect.TypeOf(object).String())
}

func objectsToWorkflowTemplates(list []runtime.Object) []*wfv1.WorkflowTemplate {
	ret := make([]*wfv1.WorkflowTemplate, len(list))
	for i, object := range list {
		ret[i], _ = objectToWorkflowTemplate(object)
	}
	return ret
}

func interfaceToWorkflowTemplate(object interface{}) (*wfv1.WorkflowTemplate, error) {
	v := &wfv1.WorkflowTemplate{}
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return v, fmt.Errorf("malformed workflow template: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(object).String())
	}
	err := util.FromUnstructuredObj(un, v)
	if err != nil {
		return v, fmt.Errorf("malformed workflow template \"%s/%s\": %w", un.GetNamespace(), un.GetName(), err)
	}
	return v, nil
}

// Get WorkflowTemplates from Informer
type WorkflowTemplateFromInformerGetter struct {
	wftmplInformer wfextvv1alpha1.WorkflowTemplateInformer
	namespace      string
}

func (getter *WorkflowTemplateFromInformerGetter) Get(_ context.Context, name string) (*wfv1.WorkflowTemplate, error) {

	obj, exists, err := getter.wftmplInformer.Informer().GetStore().GetByKey(getter.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("WorkflowTemplate Informer cannot find WorkflowTemplate of name %q in namespace %q", name, getter.namespace)
	}
	wfTmpl, ok := obj.(*wfv1.WorkflowTemplate)
	if !ok {
		return nil, fmt.Errorf("malformed workflow template: expected \"*wfv1.WorkflowTemplate\", got \"%s\"", reflect.TypeOf(obj).String())
	}
	return wfTmpl, nil
}
func NewWorkflowTemplateFromInformerGetter(wftmplInformer wfextvv1alpha1.WorkflowTemplateInformer, namespace string) templateresolution.WorkflowTemplateNamespacedGetter {
	return &WorkflowTemplateFromInformerGetter{wftmplInformer: wftmplInformer, namespace: namespace}
}
