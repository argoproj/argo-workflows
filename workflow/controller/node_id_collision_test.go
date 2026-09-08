package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// assertAcyclic walks Children from the root and fails on any revisit.
func assertAcyclic(t *testing.T, wf *wfv1.Workflow) {
	t.Helper()
	visited := map[string]bool{}
	var walk func(id string, path []string)
	walk = func(id string, path []string) {
		for _, p := range path {
			require.NotEqual(t, p, id, "cycle in node graph: %v -> %s", path, id)
		}
		if visited[id] {
			return
		}
		visited[id] = true
		node, err := wf.Status.Nodes.Get(id)
		require.NoError(t, err)
		for _, child := range node.Children {
			walk(child, append(path, id))
		}
	}
	walk(wf.Name, nil)
}

func assertNodeIDInvariant(t *testing.T, wf *wfv1.Workflow) {
	t.Helper()
	for id, n := range wf.Status.Nodes {
		assert.Equal(t, id, n.ID)
		assert.Equal(t, wf.NodeID(n.HashName()), n.ID, "node %s", n.Name)
		found, err := wf.GetNodeByName(n.Name)
		require.NoError(t, err, "node %s not found by name", n.Name)
		assert.Equal(t, n.ID, found.ID, "lookup of %s returned a different node", n.Name)
		for _, childID := range n.Children {
			_, err := wf.Status.Nodes.Get(childID)
			require.NoError(t, err, "node %s has a child edge to %s, which does not exist", n.Name, childID)
		}
	}
}

// The workflow from https://github.com/argoproj/argo-workflows/issues/16376:
// the outer step group custom-job-thbh7[0] and the pod node
// custom-job-thbh7[0].custom-job[0].custom-job-main hash to the same ID.
var nodeIDCollisionIssueTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-debug
  namespace: default
spec:
  templates:
    - name: custom-job
      inputs:
        parameters:
          - name: message
      steps:
      - - name: custom-job-main
          template: custom-job-main
          arguments:
            parameters:
            - name: message
              value: "{{inputs.parameters.message}}"
    - name: custom-job-main
      inputs:
        parameters:
        - name: message
      container:
        image: argoproj/argosay:v2
        command: [echo]
        args: ["{{inputs.parameters.message}}"]
`

var nodeIDCollisionIssueWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: custom-job-thbh7
  namespace: default
spec:
  arguments:
    parameters:
      - name: MESSAGE
        value: hello
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: custom-job
        templateRef:
          name: workflow-template-debug
          template: custom-job
        arguments:
          parameters:
          - name: message
            value: "{{workflow.parameters.MESSAGE}}"
`

func TestNodeIDCollisionIssue16376(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(nodeIDCollisionIssueTemplate)
	wf := wfv1.MustUnmarshalWorkflow(nodeIDCollisionIssueWorkflow)
	outerSG := "custom-job-thbh7[0]"
	leafName := "custom-job-thbh7[0].custom-job[0].custom-job-main"
	require.Equal(t, wf.NodeID(outerSG), wf.NodeID(leafName), "the names this test relies on must collide")

	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assertAcyclic(t, woc.wf)
	assertNodeIDInvariant(t, woc.wf)
	assert.Len(t, woc.wf.Status.Nodes, 5)

	winner, err := woc.wf.GetNodeByName(outerSG)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeTypeStepGroup, winner.Type)
	assert.Equal(t, int32(0), winner.HashSuffix)
	leaf, err := woc.wf.GetNodeByName(leafName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeTypePod, leaf.Type)
	assert.Equal(t, int32(1), leaf.HashSuffix)
	assert.Equal(t, wf.NodeID(leafName+"~1"), leaf.ID)
	assert.NotEqual(t, winner.ID, leaf.ID)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, wfutil.GeneratePodName(wf.Name, leafName, "custom-job-main", leaf.ID, wfutil.GetWorkflowPodNameVersion(woc.wf)), pod.Name)
	assert.NotEqual(t, wfutil.GeneratePodName(wf.Name, outerSG, "custom-job-main", winner.ID, wfutil.GetWorkflowPodNameVersion(woc.wf)), pod.Name)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assertAcyclic(t, woc.wf)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	for _, n := range woc.wf.Status.Nodes {
		assert.Equal(t, wfv1.NodeSucceeded, n.Phase, n.Name)
	}
}

// Two sibling steps whose node names collide: collide[0].s488469 and
// collide[0].s1161224. FNV-1a has no finalisation step, so their
// descendants collide too: collide[0].s488469[0] with collide[0].s1161224[0]
// and so on down. The loser's whole subtree must resolve.
var nodeIDCollisionSubtreeWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: collide
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: s488469
        template: inner
      - name: s1161224
        template: inner
  - name: inner
    steps:
    - - name: x
        template: leaf
  - name: leaf
    container:
      image: argoproj/argosay:v2
`

func TestNodeIDCollisionSubtree(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(nodeIDCollisionSubtreeWorkflow)
	a, b := "collide[0].s488469", "collide[0].s1161224"
	require.Equal(t, wf.NodeID(a), wf.NodeID(b), "the names this test relies on must collide")
	require.Equal(t, wf.NodeID(a+"[0].x"), wf.NodeID(b+"[0].x"))

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assertAcyclic(t, woc.wf)
	assertNodeIDInvariant(t, woc.wf)
	// root, step group, 2 steps nodes, 2 inner step groups, 2 pods
	assert.Len(t, woc.wf.Status.Nodes, 8)

	winner, loser := a, b
	if n, err := woc.wf.GetNodeByName(a); assert.NoError(t, err) && n.HashSuffix != 0 {
		winner, loser = b, a
	}
	for _, rel := range []string{"", "[0]", "[0].x"} {
		w, err := woc.wf.GetNodeByName(winner + rel)
		require.NoError(t, err)
		assert.Equal(t, int32(0), w.HashSuffix, w.Name)
		l, err := woc.wf.GetNodeByName(loser + rel)
		require.NoError(t, err)
		assert.Equal(t, int32(1), l.HashSuffix, l.Name)
		assert.NotEqual(t, w.ID, l.ID)
	}

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 2)
	assert.NotEqual(t, pods.Items[0].Name, pods.Items[1].Name)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assertAcyclic(t, woc.wf)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Two colliding DAG tasks where one is deferred by parallelism. The edge for
// the deferred task must not be persisted before its node exists: the
// colliding sibling would claim the predicted slot on a later reconcile,
// leaving the DAG boundary holding an edge to a node that is not its child.
var nodeIDCollisionDeferredDAGWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: collide
  namespace: default
spec:
  entrypoint: main
  parallelism: 1
  templates:
  - name: main
    dag:
      tasks:
      - name: filler
        template: leaf
      - name: t2335786
        template: leaf
        depends: filler
      - name: t3074240
        template: leaf
  - name: leaf
    container:
      image: argoproj/argosay:v2
`

func TestNodeIDCollisionDeferredDAGTask(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(nodeIDCollisionDeferredDAGWorkflow)
	a, b := "collide.t2335786", "collide.t3074240"
	require.Equal(t, wf.NodeID(a), wf.NodeID(b), "the names this test relies on must collide")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 12 {
		woc.operate(ctx)
		if woc.wf.Status.Fulfilled() {
			break
		}
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	assertAcyclic(t, woc.wf)
	assertNodeIDInvariant(t, woc.wf)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	filler, err := woc.wf.GetNodeByName("collide.filler")
	require.NoError(t, err)
	taskA, err := woc.wf.GetNodeByName(a)
	require.NoError(t, err)
	taskB, err := woc.wf.GetNodeByName(b)
	require.NoError(t, err)
	assert.NotEqual(t, taskA.ID, taskB.ID)
	assert.ElementsMatch(t, []int32{0, 1}, []int32{taskA.HashSuffix, taskB.HashSuffix})

	// filler and t3074240 have no dependencies, so they are the DAG boundary's
	// children; t2335786 depends on filler, so it hangs off filler alone
	boundary, err := woc.wf.Status.Nodes.Get(woc.wf.Name)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{filler.ID, taskB.ID}, boundary.Children, "DAG boundary children")
	assert.Equal(t, []string{taskA.ID}, filler.Children, "filler children")
	assert.Empty(t, taskA.Children)
	assert.Empty(t, taskB.Children)
}
