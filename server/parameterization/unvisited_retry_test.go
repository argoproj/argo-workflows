package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func unvisitedRetryFixture() (*wfv1.Workflow, *wfv1.Workflow) {
	w, next := partialFixture()
	for _, wf := range []*wfv1.Workflow{w, next} {
		wf.Spec.Templates[0].DAG.Tasks = append(wf.Spec.Templates[0].DAG.Tasks, wfv1.DAGTask{Name: "future", Template: "future", Dependencies: []string{"change"}})
		wf.Spec.Templates = append(wf.Spec.Templates, wfv1.Template{Name: "future", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "old", Template: "change"}}}})
		wf.Status.Nodes["f"] = wfv1.NodeStatus{ID: "f", Name: "wf.future", TemplateName: "future", BoundaryID: "wf", Type: wfv1.NodeTypeSkipped, Phase: wfv1.NodeOmitted}
		node := wf.Status.Nodes["c"]
		node.Children = []string{"f"}
		wf.Status.Nodes["c"] = node
	}
	next.Spec.Templates[4].DAG.Tasks = []wfv1.DAGTask{{Name: "replacement", Template: "replacement"}}
	next.Spec.Templates = append(next.Spec.Templates, wfv1.Template{Name: "replacement", Metadata: wfv1.Metadata{Annotations: map[string]string{p.Group + "/retry-safe": "idempotent"}}})
	return w, next
}

func TestPartialRetryAdoptsOnlyUnvisitedTemplateImplementations(t *testing.T) {
	w, next := unvisitedRetryFixture()
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, next, "change")
	require.NoError(t, err)
	require.Contains(t, plan.RetainedNodes, "a")
	require.Contains(t, plan.RetainedNodes, "k")
	require.Contains(t, plan.AffectedTasks, "future/replacement")
	require.NotContains(t, plan.RetainedNodes, "f")
	for name, edit := range map[string]func(*wfv1.Workflow){
		"shared executed template": func(n *wfv1.Workflow) { n.Spec.Templates[3].Metadata.Annotations["changed"] = "true" },
		"visited root graph":       func(n *wfv1.Workflow) { n.Spec.Templates[0].DAG.Tasks[0].Dependencies = []string{"future"} },
		"identity":                 func(n *wfv1.Workflow) { n.Spec.ServiceAccountName = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := next.DeepCopy()
			edit(bad)
			_, err := planPartialRetry(logging.TestContext(t.Context()), w, bad, "change")
			require.Error(t, err)
		})
	}
	visited := w.DeepCopy()
	node := visited.Status.Nodes["f"]
	node.Type = wfv1.NodeTypeDAG
	node.Phase = wfv1.NodeFailed
	visited.Status.Nodes["f"] = node
	_, err = planPartialRetry(logging.TestContext(t.Context()), visited, next, "change")
	require.ErrorContains(t, err, "visited template future")
}
