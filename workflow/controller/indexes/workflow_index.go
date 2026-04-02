package indexes

import (
	"context"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

var (
	indexWorkflowSemaphoreKeys = os.Getenv("INDEX_WORKFLOW_SEMAPHORE_KEYS") != "false"
)

func init() {
	logging.InitLogger().WithField("indexWorkflowSemaphoreKeys", indexWorkflowSemaphoreKeys).Info(context.Background(), "index config")
}

func MetaWorkflowIndexFunc(obj any) ([]string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	name, ok := m.GetLabels()[common.LabelKeyWorkflow]
	if !ok {
		return nil, nil
	}
	return []string{WorkflowIndexValue(m.GetNamespace(), name)}, nil
}

// MetaNodeIDIndexFunc takes a kubernetes object and returns either the
// namespace and its node id or the namespace and its name
func MetaNodeIDIndexFunc(obj any) ([]string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	if nodeID, ok := m.GetAnnotations()[common.AnnotationKeyNodeID]; ok {
		return []string{m.GetNamespace() + "/" + nodeID}, nil
	}

	return []string{m.GetNamespace() + "/" + m.GetName()}, nil
}

func WorkflowIndexValue(namespace, name string) string {
	return namespace + "/" + name
}

func WorkflowSemaphoreKeysIndexFunc() cache.IndexFunc {
	if !indexWorkflowSemaphoreKeys {
		return func(obj any) ([]string, error) {
			return nil, nil
		}
	}
	return func(obj any) ([]string, error) {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return nil, nil
		}
		completed, ok := un.GetLabels()[common.LabelKeyCompleted]
		if ok && completed != "false" {
			return nil, nil
		}
		wf, err := util.FromUnstructured(un)
		if err != nil {
			return nil, err
		}
		return wf.GetSemaphoreKeys(), nil
	}
}
