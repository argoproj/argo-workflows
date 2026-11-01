package parameterization

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func partialFixture() (*wfv1.Workflow, *wfv1.Workflow) {
	str := func(v string) *wfv1.AnyString { x := wfv1.AnyString(v); return &x }
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "uid", ResourceVersion: "17", Labels: map[string]string{}}, Spec: wfv1.WorkflowSpec{Entrypoint: "main", Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "value", Value: str("old")}}}, Templates: []wfv1.Template{
		{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{
			{Name: "approve", Template: "approve"},
			{Name: "keep", Template: "keep", Dependencies: []string{"approve"}},
			{Name: "change", Template: "change", Dependencies: []string{"keep"}, Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "value", Value: str("{{workflow.parameters.value}}")}}}},
		}}},
		{Name: "approve", Suspend: &wfv1.SuspendTemplate{}},
		{Name: "keep", Metadata: wfv1.Metadata{Annotations: map[string]string{p.Group + "/result-reuse": "same-inputs"}}},
		{Name: "change", Metadata: wfv1.Metadata{Annotations: map[string]string{p.Group + "/retry-safe": "idempotent"}}},
	}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowFailed, Nodes: wfv1.Nodes{}}}
	for _, n := range []wfv1.NodeStatus{
		{ID: "wf", Name: "wf", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeFailed, Children: []string{"a"}},
		{ID: "a", Name: "wf.approve", TemplateName: "approve", BoundaryID: "wf", Type: wfv1.NodeTypeSuspend, Phase: wfv1.NodeSucceeded, Children: []string{"k"}},
		{ID: "k", Name: "wf.keep", TemplateName: "keep", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, Children: []string{"c"}},
		{ID: "c", Name: "wf.change", TemplateName: "change", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeFailed},
	} {
		n.FinishedAt = metav1.NewTime(time.Unix(100, 0))
		wf.Status.Nodes[n.ID] = n
	}
	next := wf.DeepCopy()
	next.Spec.Arguments.Parameters[0].Value = str("new")
	return wf, next
}
func TestPartialRetryRetainsApprovalAndSuccessfulOutputs(t *testing.T) {
	wf, next := partialFixture()
	before := wf.DeepCopy()
	plan, err := planPartialRetry(logging.TestContext(t.Context()), wf, next)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
	require.Len(t, plan.RetainedNodes, 2)
	require.Contains(t, plan.RetainedNodes, "a")
	require.Contains(t, plan.RetainedNodes, "k")
	require.Equal(t, []string{"value=new"}, plan.Parameters)
	require.Equal(t, before, wf, "preview never mutates workflow")
}
func TestPartialRetryRejectsUnprovedChanges(t *testing.T) {
	tests := map[string]func(*wfv1.Workflow, *wfv1.Workflow){
		"running": func(w, n *wfv1.Workflow) { w.Status.Phase = wfv1.WorkflowRunning },
		"wrong invocation boundary": func(w, n *wfv1.Workflow) {
			node := w.Status.Nodes["k"]
			node.BoundaryID = "other"
			w.Status.Nodes["k"] = node
		},
		"wrong invocation template": func(w, n *wfv1.Workflow) {
			node := w.Status.Nodes["k"]
			node.TemplateName = "change"
			w.Status.Nodes["k"] = node
		},
		"undeclared loop invocation": func(w, n *wfv1.Workflow) {
			node := w.Status.Nodes["k"]
			node.Name = "wf.keep(0:value:x)"
			w.Status.Nodes["k"] = node
		},
		"template": func(w, n *wfv1.Workflow) { n.Spec.Templates[2].Name = "another" },
		"unrelated failure": func(w, n *wfv1.Workflow) {
			x := w.Status.Nodes["k"]
			x.Phase = wfv1.NodeFailed
			w.Status.Nodes["k"] = x
		},
		"reuse contract": func(w, n *wfv1.Workflow) {
			w.Spec.Templates[2].Metadata.Annotations = nil
			n.Spec.Templates[2].Metadata.Annotations = nil
		},
		"retry contract": func(w, n *wfv1.Workflow) {
			w.Spec.Templates[3].Metadata.Annotations = nil
			n.Spec.Templates[3].Metadata.Annotations = nil
		},
		"loop identity": func(w, n *wfv1.Workflow) {
			w.Spec.Templates[0].DAG.Tasks[2].WithParam = "{{workflow.parameters.value}}"
			n.Spec.Templates[0].DAG.Tasks[2].WithParam = "{{workflow.parameters.value}}"
		},
		"artifact": func(w, n *wfv1.Workflow) {
			w.Spec.Templates[2].Outputs.Artifacts = []wfv1.Artifact{{Name: "missing"}}
			n.Spec.Templates[2].Outputs.Artifacts = w.Spec.Templates[2].Outputs.Artifacts
		},
		"global implementation": func(w, n *wfv1.Workflow) {
			w.Spec.Templates[2].Metadata.Labels = map[string]string{"value": "{{workflow.parameters.value}}"}
			n.Spec.Templates[2].Metadata.Labels = w.Spec.Templates[2].Metadata.Labels
		},
	}
	for name, edit := range tests {
		t.Run(name, func(t *testing.T) {
			w, n := partialFixture()
			edit(w, n)
			_, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
			require.Error(t, err)
		})
	}
}
func TestPartialRetryIncludesDownstreamClosure(t *testing.T) {
	w, n := partialFixture()
	for _, x := range []*wfv1.Workflow{w, n} {
		x.Spec.Templates[0].DAG.Tasks = append(x.Spec.Templates[0].DAG.Tasks, wfv1.DAGTask{Name: "after", Template: "change", Dependencies: []string{"change"}})
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"after", "change"}, plan.AffectedTasks)
}

func TestPartialRetryUsesExactNativeReferences(t *testing.T) {
	wf, next := partialFixture()
	for _, w := range []*wfv1.Workflow{wf, next} {
		suffix := wfv1.AnyString("unchanged")
		w.Spec.Arguments.Parameters = append(w.Spec.Arguments.Parameters, wfv1.Parameter{Name: "valueExtra", Value: &suffix})
		value := wfv1.AnyString("{{workflow.parameters.valueExtra}}")
		w.Spec.Templates[0].DAG.Tasks[1].Arguments.Parameters = []wfv1.Parameter{{Name: "item", Value: &value}}
		quoted := wfv1.AnyString("{{=workflow['parameters']['value']}}")
		w.Spec.Templates[0].DAG.Tasks[2].Arguments.Parameters[0].Value = &quoted
		w.Spec.Templates[2].Metadata.Annotations["example/description"] = "workflow.parameters.value is documentation, not an input"
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), wf, next)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
	require.Contains(t, plan.RetainedNodes, "k")
}
