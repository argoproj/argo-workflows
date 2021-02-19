package verify

import (
	_ "github.com/go-python/gpython/builtin"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/python"
)

func Workflow(wf *wfv1.Workflow) error {
	verify, ok := wf.GetAnnotations()[workflow.WorkflowFullName+"/verify.py"]
	if !ok {
		return nil
	}
	nodes := wfv1.Nodes{}
	for _, n := range wf.Status.Nodes {
		nodes[n.DisplayName] = n
	}
	return python.Run(verify, map[string]interface{}{
		"metadata": wf,
		"nodes":    nodes,
		"status":   wf.Status,
	})
}
