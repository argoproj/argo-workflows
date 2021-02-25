package verify

import (
	"fmt"

	_ "github.com/go-python/gpython/builtin"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/python"
)

const annotationName = workflow.WorkflowFullName + "/verify.py"

func Workflow(wf *wfv1.Workflow) error {
	verify, ok := wf.GetAnnotations()[annotationName]
	if !ok {
		return fmt.Errorf("cannot verify workflow: annotation %s not found", annotationName)
	}
	nodes := wfv1.Nodes{}
	for _, n := range wf.Status.Nodes {
		nodes[n.DisplayName] = n
	}
	return python.Run(verify, map[string]interface{}{
		"metadata": wf.ObjectMeta,
		"spec":     wf.Spec,
		"nodes":    nodes,
		"status":   wf.Status,
	})
}
