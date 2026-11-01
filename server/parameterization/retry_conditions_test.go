package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	native "github.com/argoproj/argo-workflows/v4/workflow/util"
)

func TestRetryConditionOnlyParameterChange(t *testing.T) {
	for _, condition := range []string{"{{workflow.parameters.value}} == new", "{{=workflow.parameters.value == 'new'}}"} {
		t.Run(condition, func(t *testing.T) {
			w, n := partialFixture()
			for _, wf := range []*wfv1.Workflow{w, n} {
				wf.Spec.Templates[0].DAG.Tasks[2].Arguments = wfv1.Arguments{}
				wf.Spec.Templates[0].DAG.Tasks[2].When = condition
			}
			plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
			require.NoError(t, err)
			require.Equal(t, []string{"change"}, plan.AffectedTasks)
			require.Contains(t, plan.RetainedNodes, "a")
			require.Contains(t, plan.RetainedNodes, "k")
		})
	}
}

func TestRetryConditionTracksOriginalRetryAndTransitiveOutputs(t *testing.T) {
	w, _ := partialFixture()
	root := &w.Spec.Templates[0]
	root.DAG.Tasks = append(root.DAG.Tasks,
		wfv1.DAGTask{Name: "ordered", Template: "keep", Dependencies: []string{"change"}},
		wfv1.DAGTask{Name: "consumer", Template: "change", Dependencies: []string{"ordered"}, When: "{{=tasks.change.outputs.parameters.value == workflow.parameters.value}}"})
	w.Spec.Templates[2].Metadata.Annotations[p.Group+"/retry-dependencies"] = "inputs"
	graph, err := newRetryGraph(&w.Spec, nil, "change")
	require.NoError(t, err)
	require.Equal(t, []string{"change", "consumer"}, graph.affectedLeaves())
	require.False(t, graph.root.children["ordered"].affected)
}

func TestRetryStepsConditionIsDataDependency(t *testing.T) {
	w, _ := partialFixture()
	w.Spec.Templates[0].DAG = nil
	w.Spec.Templates[0].Steps = []wfv1.ParallelSteps{
		{Steps: []wfv1.WorkflowStep{{Name: "producer", Template: "change"}}},
		{Steps: []wfv1.WorkflowStep{{Name: "consumer", Template: "change", When: "{{=steps.producer.outputs.parameters.value == workflow.parameters.value}}"}}},
	}
	graph, err := newRetryGraph(&w.Spec, nil, "producer")
	require.NoError(t, err)
	require.Equal(t, []string{"consumer", "producer"}, graph.affectedLeaves())
}

func TestRetryConditionRejectsUnboundedOrUnorderedProducers(t *testing.T) {
	for _, condition := range []string{
		"{{=tasks[workflow.parameters.value].outputs.parameters.value == 'new'}}",
		"{{tasks.change.outputs.parameters.value}} == new",
		"{{tasks.unknown.outputs.parameters.value}} == new",
	} {
		t.Run(condition, func(t *testing.T) {
			w, n := partialFixture()
			for _, wf := range []*wfv1.Workflow{w, n} {
				wf.Spec.Templates[0].DAG.Tasks[2].When = condition
			}
			_, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
			require.ErrorContains(t, err, "statically named upstream dependencies")
		})
	}
}

func TestRetryConditionPreservesUnchangedFalseDecision(t *testing.T) {
	w, n := partialFixture()
	for _, wf := range []*wfv1.Workflow{w, n} {
		wf.Spec.Templates[0].DAG.Tasks[1].When = "false"
		node := wf.Status.Nodes["k"]
		node.Type, node.Phase = wfv1.NodeTypeSkipped, wfv1.NodeSkipped
		wf.Status.Nodes["k"] = node
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), w, n)
	require.NoError(t, err)
	require.Contains(t, plan.RetainedNodes, "k")
}

func TestRetryNestedConditionRemovesOldBoundary(t *testing.T) {
	w, n := nestedRetryFixture()
	for _, wf := range []*wfv1.Workflow{w, n} {
		wf.Spec.Templates[0].DAG.Tasks[1].When = "{{workflow.parameters.value}} == old"
		wf.Spec.Templates[2].Metadata.Annotations[p.Group+"/retry-safe"] = "idempotent"
	}
	ctx := logging.TestContext(t.Context())
	plan, err := planPartialRetry(ctx, w, n)
	require.NoError(t, err)
	proof := p.NativeRetryProof{UID: string(w.UID), ResourceVersion: w.ResourceVersion, RetainedNodes: plan.RetainedNodes}
	after, _, err := native.FormulateRetryWorkflow(p.WithRetryProof(ctx, proof), w, true, plan.Selector, plan.Parameters)
	require.NoError(t, err)
	for _, id := range []string{"v", "va", "vk", "vc"} {
		require.NotContains(t, after.Status.Nodes, id)
	}
	for _, id := range []string{"s", "sa", "sk", "sc"} {
		require.Equal(t, w.Status.Nodes[id], after.Status.Nodes[id])
	}
}
