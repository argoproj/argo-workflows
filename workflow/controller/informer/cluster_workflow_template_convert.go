package informer

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

// this function always tries to return a value, even if it is badly formed
func objectToClusterWorkflowTemplate(object runtime.Object) (*wfv1.ClusterWorkflowTemplate, error) {
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

func objectsToClusterWorkflowTemplates(list []runtime.Object) []*wfv1.ClusterWorkflowTemplate {
	ret := make([]*wfv1.ClusterWorkflowTemplate, len(list))
	for i, object := range list {
		ret[i], _ = objectToClusterWorkflowTemplate(object)
	}
	return ret
}

type ClusterWorkflowTemplateFromInformerGetter struct {
	cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer
}

func (getter *ClusterWorkflowTemplateFromInformerGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	obj, exists, err := getter.cwftmplInformer.Informer().GetStore().GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("ClusterWorkflowTemplate Informer cannot find ClusterWorkflowTemplate of name %q", name)
	}
	cwfTmpl, castOk := obj.(*wfv1.ClusterWorkflowTemplate)
	if !castOk {
		return nil, fmt.Errorf("ClusterWorkflowTemplate Informer found ClusterWorkflowTemplate of name %q but somehow it's not a WorkflowTemplate: %+v",
			name, obj)
	}
	return cwfTmpl, nil
}

func NewClusterWorkflowTemplateFromInformerGetter(cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer) templateresolution.ClusterWorkflowTemplateGetter {
	return &ClusterWorkflowTemplateFromInformerGetter{cwftmplInformer: cwftmplInformer}
}
