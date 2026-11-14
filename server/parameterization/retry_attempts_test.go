package parameterization

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func attemptRetryFixture() (*wfv1.Workflow, *wfv1.Workflow) {
	wf, next := partialFixture()
	limit := intstr.FromInt(1)
	for _, w := range []*wfv1.Workflow{wf, next} {
		for _, i := range []int{2, 3} {
			w.Spec.Templates[i].Container = &corev1.Container{Image: "alpine:3.22"}
			w.Spec.Templates[i].RetryStrategy = &wfv1.RetryStrategy{Limit: &limit}
		}
		for _, id := range []string{"k", "c"} {
			group := w.Status.Nodes[id]
			group.Type = wfv1.NodeTypeRetry
			children := group.Children
			group.Children = []string{id + "0", id + "1"}
			group.NodeFlag = &wfv1.NodeFlag{Retried: true}
			w.Status.Nodes[id] = group
			for index, childID := range group.Children {
				child := group
				child.ID, child.Type, child.Name = childID, wfv1.NodeTypePod, group.Name+"("+strconv.Itoa(index)+")"
				child.Children = nil
				child.NodeFlag = &wfv1.NodeFlag{Retried: true}
				child.Phase = wfv1.NodeFailed
				if index == 1 {
					child.Phase, child.Children = group.Phase, children
				}
				w.Status.Nodes[childID] = child
			}
		}
	}
	return wf, next
}

func TestPartialRetryNativeAttemptGroups(t *testing.T) {
	wf, next := attemptRetryFixture()
	before := wf.DeepCopy()
	plan, err := planPartialRetry(logging.TestContext(t.Context()), wf, next)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
	for _, id := range []string{"a", "k", "k0", "k1"} {
		require.Contains(t, plan.RetainedNodes, id)
	}
	for _, id := range []string{"c", "c0", "c1"} {
		require.NotContains(t, plan.RetainedNodes, id)
	}
	require.Equal(t, before, wf)
}

func TestPartialRetryParallelStepsAttemptGroups(t *testing.T) {
	wf, next := attemptRetryFixture()
	for _, w := range []*wfv1.Workflow{wf, next} {
		tasks := w.Spec.Templates[0].DAG.Tasks
		w.Spec.Templates[0].DAG = nil
		w.Spec.Templates[0].Steps = []wfv1.ParallelSteps{
			{Steps: []wfv1.WorkflowStep{{Name: "approve", Template: "approve"}}},
			{Steps: []wfv1.WorkflowStep{{Name: "keep", Template: "keep"}, {Name: "change", Template: "change", Arguments: tasks[2].Arguments}}},
		}
		root := w.Status.Nodes["wf"]
		root.Type, root.Children = wfv1.NodeTypeSteps, []string{"g0"}
		w.Status.Nodes["wf"] = root
		w.Status.Nodes["g0"] = wfv1.NodeStatus{ID: "g0", Name: "wf[0]", Type: wfv1.NodeTypeStepGroup, Phase: wfv1.NodeSucceeded, BoundaryID: "wf", Children: []string{"a"}}
		w.Status.Nodes["g1"] = wfv1.NodeStatus{ID: "g1", Name: "wf[1]", Type: wfv1.NodeTypeStepGroup, Phase: wfv1.NodeFailed, BoundaryID: "wf", Children: []string{"k", "c"}}
		a := w.Status.Nodes["a"]
		a.Name, a.Children = "wf[0].approve", []string{"g1"}
		w.Status.Nodes["a"] = a
		for _, id := range []string{"k", "c"} {
			group := w.Status.Nodes[id]
			group.Name = "wf[1]." + group.TemplateName
			w.Status.Nodes[id] = group
			for index, childID := range group.Children {
				child := w.Status.Nodes[childID]
				child.Name, child.Children = group.Name+"("+strconv.Itoa(index)+")", nil
				w.Status.Nodes[childID] = child
			}
		}
	}
	plan, err := planPartialRetry(logging.TestContext(t.Context()), wf, next)
	require.NoError(t, err)
	for _, id := range []string{"a", "g0", "k", "k0", "k1"} {
		require.Contains(t, plan.RetainedNodes, id)
	}
}

func TestPartialRetryAttemptIdentityRejectsAmbiguity(t *testing.T) {
	for _, test := range []string{"name", "boundary", "template", "missing", "running", "group failed", "duplicate"} {
		t.Run(test, func(t *testing.T) {
			wf, next := attemptRetryFixture()
			node := wf.Status.Nodes["k0"]
			switch test {
			case "name":
				node.Name = "wf.keep(3)"
			case "boundary":
				node.BoundaryID = "other"
			case "template":
				node.TemplateName = "change"
			case "running":
				node.Phase = wfv1.NodeRunning
			case "group failed":
				group := wf.Status.Nodes["k"]
				group.Phase = wfv1.NodeFailed
				wf.Status.Nodes["k"] = group
			case "duplicate":
				group := wf.Status.Nodes["k"]
				group.ID = "clone"
				wf.Status.Nodes["clone"] = group
			}
			wf.Status.Nodes["k0"] = node
			if test == "missing" {
				delete(wf.Status.Nodes, "k0")
			}
			_, err := planPartialRetry(logging.TestContext(t.Context()), wf, next)
			require.Error(t, err)
		})
	}
}
