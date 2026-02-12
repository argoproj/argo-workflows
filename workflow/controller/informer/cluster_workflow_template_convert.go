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

func objectToClusterWorkflowTemplate(object runtime.Object) (*wfv1.ClusterWorkflowTemplate, error) {
	return interfaceToClusterWorkflowTemplate(object)
}

func objectsToClusterWorkflowTemplates(list []runtime.Object) []*wfv1.ClusterWorkflowTemplate {
	ret := make([]*wfv1.ClusterWorkflowTemplate, len(list))
	for i, object := range list {
		ret[i], _ = objectToClusterWorkflowTemplate(object)
	}
	return ret
}

// this function always tries to return a value, even if it is badly formed
func interfaceToClusterWorkflowTemplate(object any) (*wfv1.ClusterWorkflowTemplate, error) {
	v := &wfv1.ClusterWorkflowTemplate{}
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return v, fmt.Errorf("malformed cluster workflow template: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(object).String())
	}
	err := util.FromUnstructuredObj(un, v)
	if err != nil {
		return v, fmt.Errorf("malformed cluster workflow template \"%s\": %w", un.GetName(), err)
	}
	return v, nil
}

// Get ClusterWorkflowTemplates from Informer
type ClusterWorkflowTemplateFromInformerGetter struct {
	cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer
}

func (getter *ClusterWorkflowTemplateFromInformerGetter) Get(_ context.Context, name string) (*wfv1.ClusterWorkflowTemplate, error) {
	obj, exists, err := getter.cwftmplInformer.Informer().GetStore().GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("ClusterWorkflowTemplate Informer cannot find ClusterWorkflowTemplate of name %q", name)
	}
	cwfTmpl, err := interfaceToClusterWorkflowTemplate(obj)
	if err != nil {
		return nil, err
	}
	return cwfTmpl, nil
}

func NewClusterWorkflowTemplateFromInformerGetter(cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer) templateresolution.ClusterWorkflowTemplateGetter {
	return &ClusterWorkflowTemplateFromInformerGetter{cwftmplInformer: cwftmplInformer}
}
