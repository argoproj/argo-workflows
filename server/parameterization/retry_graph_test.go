package parameterization

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func nestedRetryFixture() (*wfv1.Workflow, *wfv1.Workflow) {
	w, _ := partialFixture()
	args := func(v string) wfv1.Arguments {
		x := wfv1.AnyString(v)
		return wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "payload", Value: &x}}}
	}
	w.Spec.Templates[0].DAG.Tasks = []wfv1.DAGTask{{Name: "stable", Template: "group", Arguments: args("stable")}, {Name: "vary", Template: "group", Arguments: args("{{workflow.parameters.value}}")}, {Name: "report", Template: "change", Dependencies: []string{"vary"}}}
	w.Spec.Templates[3].Metadata.Annotations[p.Group+"/result-reuse"] = "same-inputs"
	w.Spec.Templates = append(w.Spec.Templates, wfv1.Template{Name: "group", Inputs: wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: "payload"}}}, DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "approve", Template: "approve"}, {Name: "keep", Template: "keep", Dependencies: []string{"approve"}}, {Name: "change", Template: "change", Dependencies: []string{"keep"}, Arguments: args("{{inputs.parameters.payload}}")}}}})
	w.Status.Nodes = wfv1.Nodes{}
	add := func(n wfv1.NodeStatus) { n.FinishedAt = metav1.NewTime(time.Unix(100, 0)); w.Status.Nodes[n.ID] = n }
	add(wfv1.NodeStatus{ID: "wf", Name: "wf", TemplateName: "main", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeFailed, Children: []string{"s", "v"}})
	for _, c := range []struct {
		id, name string
		phase    wfv1.NodePhase
	}{{"s", "stable", wfv1.NodeSucceeded}, {"v", "vary", wfv1.NodeFailed}} {
		prefix := "wf." + c.name
		add(wfv1.NodeStatus{ID: c.id, Name: prefix, TemplateName: "group", BoundaryID: "wf", Type: wfv1.NodeTypeDAG, Phase: c.phase, Children: []string{c.id + "a"}, OutboundNodes: []string{c.id + "c"}})
		add(wfv1.NodeStatus{ID: c.id + "a", Name: prefix + ".approve", TemplateName: "approve", BoundaryID: c.id, Type: wfv1.NodeTypeSuspend, Phase: wfv1.NodeSucceeded, Children: []string{c.id + "k"}})
		add(wfv1.NodeStatus{ID: c.id + "k", Name: prefix + ".keep", TemplateName: "keep", BoundaryID: c.id, Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, Children: []string{c.id + "c"}})
		add(wfv1.NodeStatus{ID: c.id + "c", Name: prefix + ".change", TemplateName: "change", BoundaryID: c.id, Type: wfv1.NodeTypePod, Phase: c.phase})
	}
	n := w.DeepCopy()
	v := wfv1.AnyString("new")
	n.Spec.Arguments.Parameters[0].Value = &v
	return w, n
}

func TestNestedParameterRetryPreservesOtherInvocationAndInnerApproval(t *testing.T) {
	w, n := nestedRetryFixture()
	before := w.DeepCopy()
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"report", "vary/change"}, plan.AffectedTasks)
	for _, id := range []string{"s", "sa", "sk", "sc", "va", "vk"} {
		require.Contains(t, plan.RetainedNodes, id)
	}
	require.NotContains(t, plan.RetainedNodes, "v")
	require.Equal(t, before, w)
}

func TestNestedRetryRejectsAmbiguousOrChangedTopology(t *testing.T) {
	tests := map[string]func(*wfv1.Workflow){
		"recursive call": func(w *wfv1.Workflow) { w.Spec.Templates[4].DAG.Tasks[2].Template = "group" },
		"nested loop": func(w *wfv1.Workflow) {
			w.Spec.Templates[0].DAG.Tasks[1].WithItems = []wfv1.Item{{Value: []byte(`"x"`)}}
		},
		"output changes loop": func(w *wfv1.Workflow) {
			w.Spec.Templates[0].DAG.Tasks[2].WithParam = "{{=tasks['vary'].outputs.parameters.items}}"
		},
		"wrong boundary": func(w *wfv1.Workflow) { n := w.Status.Nodes["vc"]; n.BoundaryID = "s"; w.Status.Nodes["vc"] = n },
		"unobserved unsafe downstream": func(w *wfv1.Workflow) {
			w.Spec.Templates = append(w.Spec.Templates, wfv1.Template{Name: "unsafe"})
			w.Spec.Templates[0].DAG.Tasks[2].Template = "unsafe"
		},
	}
	for name, edit := range tests {
		t.Run(name, func(t *testing.T) {
			w, _ := nestedRetryFixture()
			edit(w)
			n := w.DeepCopy()
			v := wfv1.AnyString("new")
			n.Spec.Arguments.Parameters[0].Value = &v
			_, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
			require.Error(t, err)
		})
	}
}

func TestRetryTracksEntrypointInputsAsWellAsGlobalParameters(t *testing.T) {
	w, n := partialFixture()
	for _, wf := range []*wfv1.Workflow{w, n} {
		wf.Spec.Templates[0].Inputs.Parameters = []wfv1.Parameter{{Name: "value"}}
		v := wfv1.AnyString("{{inputs.parameters.value}}")
		wf.Spec.Templates[0].DAG.Tasks[2].Arguments.Parameters[0].Value = &v
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
}

func TestRetryTracksPerInvocationInputDefaults(t *testing.T) {
	w, _ := nestedRetryFixture()
	group := &w.Spec.Templates[4]
	group.Inputs.Parameters[0].Default = wfv1.AnyStringPtr("{{workflow.parameters.value}}")
	// The stable invocation explicitly overrides the shared default.
	w.Spec.Templates[0].DAG.Tasks[1].Arguments = wfv1.Arguments{}
	n := w.DeepCopy()
	n.Spec.Arguments.Parameters[0].Value = wfv1.AnyStringPtr("new")
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"report", "vary/change"}, plan.AffectedTasks)
	for _, id := range []string{"s", "sa", "sk", "sc", "va", "vk"} {
		require.Contains(t, plan.RetainedNodes, id)
	}
	// A default used only in a condition invalidates that invocation too.
	for _, wf := range []*wfv1.Workflow{w, n} {
		wf.Spec.Templates[4].Inputs.Parameters = append(wf.Spec.Templates[4].Inputs.Parameters,
			wfv1.Parameter{Name: "derived", Default: wfv1.AnyStringPtr("{{inputs.parameters.payload}}")})
		wf.Spec.Templates[4].DAG.Tasks[2].When = "{{inputs.parameters.derived}} == new"
	}
	plan, err = planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"report", "vary/change"}, plan.AffectedTasks)
}

func TestStepsParameterRetryKeepsCompletedGroupsAndApprovals(t *testing.T) {
	w, _ := nestedRetryFixture()
	group := &w.Spec.Templates[4]
	for _, task := range group.DAG.Tasks {
		group.Steps = append(group.Steps, wfv1.ParallelSteps{Steps: []wfv1.WorkflowStep{{Name: task.Name, Template: task.Template, Arguments: task.Arguments}}})
	}
	group.DAG = nil
	for _, id := range []string{"s", "v"} {
		node := w.Status.Nodes[id]
		node.Type = wfv1.NodeTypeSteps
		node.Children = []string{id + "g0"}
		w.Status.Nodes[id] = node
		for i, suffix := range []string{"a", "k", "c"} {
			child := w.Status.Nodes[id+suffix]
			names := []string{"approve", "keep", "change"}
			child.Name = fmt.Sprintf("%s[%d].%s", node.Name, i, names[i])
			child.Children = nil
			if i < 2 {
				child.Children = []string{fmt.Sprintf("%sg%d", id, i+1)}
			}
			w.Status.Nodes[id+suffix] = child
			groupID := fmt.Sprintf("%sg%d", id, i)
			w.Status.Nodes[groupID] = wfv1.NodeStatus{ID: groupID, Name: fmt.Sprintf("%s[%d]", node.Name, i), Type: wfv1.NodeTypeStepGroup, BoundaryID: id, Phase: child.Phase, Children: []string{child.ID}, FinishedAt: child.FinishedAt}
		}
	}
	next := w.DeepCopy()
	next.Spec.Arguments.Parameters[0].Value = wfv1.AnyStringPtr("new")
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, next)
	require.NoError(t, err)
	require.Equal(t, []string{"report", "vary/change"}, plan.AffectedTasks)
	for _, id := range []string{"s", "sa", "sk", "sc", "sg0", "sg1", "sg2", "va", "vk", "vg0", "vg1"} {
		require.Contains(t, plan.RetainedNodes, id)
	}
}

func TestParameterTopologyRetryReplacesWholeLoop(t *testing.T) {
	w, n := partialFixture()
	for _, wf := range []*wfv1.Workflow{w, n} {
		root := &wf.Spec.Templates[0]
		root.Metadata.Annotations = map[string]string{p.Group + "/retry-loop-calls": `["change"]`}
		root.DAG.Tasks[2].WithParam = "{{workflow.parameters.value}}"
		root.DAG.Tasks[2].Arguments.Parameters[0].Value = wfv1.AnyStringPtr("{{item}}")
	}
	w.Spec.Arguments.Parameters[0].Value = wfv1.AnyStringPtr(`["a","b"]`)
	n.Spec.Arguments.Parameters[0].Value = wfv1.AnyStringPtr(`["b","c","d"]`)
	for id, node := range w.Status.Nodes {
		if node.Name != "wf.change" {
			continue
		}
		node.Type = wfv1.NodeTypeTaskGroup
		node.Phase = wfv1.NodeSucceeded
		node.Children = []string{"i0", "i1"}
		w.Status.Nodes[id] = node
		for i := range 2 {
			w.Status.Nodes[fmt.Sprintf("i%d", i)] = wfv1.NodeStatus{ID: fmt.Sprintf("i%d", i), Name: fmt.Sprintf("wf.change(%d:item)", i), TemplateName: node.TemplateName, BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded}
		}
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
	require.NotContains(t, plan.Selector, "id!=i0")
	require.NotContains(t, plan.Selector, "id!=i1")
	for id, node := range w.Status.Nodes {
		if node.Name == "wf.change" {
			require.NotContains(t, plan.Selector, "id!="+id)
		}
	}
	require.NotEmpty(t, plan.RetainedNodes)
	delete(w.Spec.Templates[0].Metadata.Annotations, p.Group+"/retry-loop-calls")
	delete(n.Spec.Templates[0].Metadata.Annotations, p.Group+"/retry-loop-calls")
	_, err = planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.Error(t, err)
}
