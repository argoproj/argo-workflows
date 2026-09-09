package indexes

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

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

// WorkflowActionIndexFunc indexes WorkflowActions by their target workflow's key. It reads the
// spec ref, not a label, so kubectl-created actions are indexed without any labelling requirement.
func WorkflowActionIndexFunc(obj any) ([]string, error) {
	a, ok := obj.(*wfv1.WorkflowAction)
	if !ok || a.Spec.WorkflowRef.Name == "" {
		return nil, nil
	}
	return []string{WorkflowIndexValue(a.Namespace, a.Spec.WorkflowRef.Name)}, nil
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

func WorkflowSemaphoreKeysIndexFunc(enabled bool) cache.IndexFunc {
	if !enabled {
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
