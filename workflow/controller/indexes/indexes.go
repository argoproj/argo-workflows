package indexes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/argoproj/argo/workflow/common"
)

const WorkflowIndex = "workflow"

func WorkflowIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	name, ok := meta.GetLabels()[common.LabelKeyWorkflow]
	if !ok {
		return []string{""}, fmt.Errorf("object has no workflow label")
	}
	return []string{WorkflowKey(meta.GetNamespace(), name)}, nil
}

func WorkflowKey(namespace, name string) string {
	return namespace + "/" + name
}
