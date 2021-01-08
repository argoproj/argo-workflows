package indexes

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

func MetaWorkflowIndexFunc(obj interface{}) ([]string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, nil
	}
	name, ok := m.GetLabels()[common.LabelKeyWorkflow]
	if !ok {
		return nil, nil
	}
	return []string{WorkflowIndexValue(m.GetNamespace(), name)}, nil
}

func WorkflowIndexValue(namespace, name string) string {
	return namespace + "/" + name
}

func WorkflowSemaphoreKeysIndexFunc() cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return nil, nil
		}
		wf, err := util.FromUnstructured(un)
		if err != nil {
			return nil, nil
		}
		return wf.GetSemaphoreKeys(), nil
	}
}
