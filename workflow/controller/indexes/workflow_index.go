package indexes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/argoproj/argo/v3/workflow/common"
)

const WorkflowIndex = "workflow"

func MetaWorkflowIndexFunc(obj interface{}) ([]string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return []string{}, fmt.Errorf("object has no meta: %v", err)
	}
	name, ok := m.GetLabels()[common.LabelKeyWorkflow]
	if !ok {
		return []string{}, fmt.Errorf("object has no workflow label")
	}
	return []string{WorkflowIndexValue(m.GetNamespace(), name)}, nil
}

func WorkflowIndexValue(namespace, name string) string {
	return namespace + "/" + name
}
