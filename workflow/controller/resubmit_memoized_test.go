package controller

// Regression tests for `argo resubmit --memoized` of a workflow whose failure sits inside a
// loop fanout. Each test runs the original workflow to failure through the operator, formulates the
// memoized resubmit exactly as the API server does, and then drives the new workflow to completion.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// setPodPhases acts like a pod controller, but decides the phase per pod from the node it belongs to.
// Returning an empty phase leaves the pod alone.
func setPodPhases(ctx context.Context, woc *wfOperationCtx, decide func(node *wfv1.NodeStatus) apiv1.PodPhase) {
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, pod := range pods.Items {
		nodeID := woc.nodeID(&pod)
		node := woc.wf.Status.Nodes[nodeID]
		phase := decide(&node)
		if phase == "" || pod.Status.Phase == phase {
			continue
		}
		pod.Status.Phase = phase
		if phase == apiv1.PodFailed {
			pod.Status.Message = "Pod failed"
		}
		updatedPod, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
		if err != nil {
			panic(err)
		}
		waitForInformer(ctx, woc.controller.PodController.TestingPodInformer(), updatedPod, func(obj any) bool {
			return obj.(*apiv1.Pod).Status.Phase == phase
		})
		if phase == apiv1.PodSucceeded {
			woc.wf.Status.MarkTaskResultComplete(ctx, nodeID)
		}
	}
}

func dumpNodes(t *testing.T, label string, wf *wfv1.Workflow) {
	t.Helper()
	names := make([]string, 0, len(wf.Status.Nodes))
	for _, n := range wf.Status.Nodes {
		names = append(names, n.Name)
	}
	sort.Strings(names)
	t.Logf("---- %s: workflow phase=%s", label, wf.Status.Phase)
	for _, name := range names {
		n, _ := wf.GetNodeByName(name)
		t.Logf("  %-40s type=%-10s phase=%-9s children=%d msg=%q", n.Name, n.Type, n.Phase, len(n.Children), n.Message)
	}
}

func podNames(ctx context.Context, woc *wfOperationCtx) []string {
	list, err := listPods(ctx, woc)
	if err != nil {
		panic(err)
	}
	var names []string
	for _, p := range list.Items {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names
}

// runToCompletion operates the workflow until it completes, deciding each pod's fate with decide.
func runToCompletion(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, decide func(node *wfv1.NodeStatus) apiv1.PodPhase) *wfOperationCtx {
	t.Helper()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for i := 0; i < 10 && !woc.wf.Status.Phase.Completed(); i++ {
		woc.operate(ctx)
		t.Logf("iteration %d: phase=%s pods=%v", i, woc.wf.Status.Phase, podNames(ctx, woc))
		setPodPhases(ctx, woc, decide)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
	return woc
}

func allSucceed(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded }

// failItemB fails every pod of fan-out item b and lets everything else succeed.
func failItemB(node *wfv1.NodeStatus) apiv1.PodPhase {
	if strings.Contains(node.Name, "(1:b)") {
		return apiv1.PodFailed
	}
	return apiv1.PodSucceeded
}

func requireNoDanglingReferences(t *testing.T, wf *wfv1.Workflow) {
	t.Helper()
	for _, n := range wf.Status.Nodes {
		for _, c := range n.Children {
			require.True(t, wf.Status.Nodes.Has(c), "node %s has a dangling child %s", n.Name, c)
		}
		for _, o := range n.OutboundNodes {
			require.True(t, wf.Status.Nodes.Has(o), "node %s has a dangling outbound node %s", n.Name, o)
		}
	}
}

// memoizedResubmit runs wf to failure with failFirst, formulates the memoized resubmit and returns
// a controller and workflow ready to be driven.
func memoizedResubmit(ctx context.Context, t *testing.T, wf *wfv1.Workflow, failFirst func(*wfv1.NodeStatus) apiv1.PodPhase) (context.CancelFunc, *WorkflowController, *wfv1.Workflow) {
	t.Helper()
	cancel, controller := newController(ctx, wf)
	woc := runToCompletion(ctx, t, controller, wf, failFirst)
	cancel()
	dumpNodes(t, "original run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	newWf, err := util.FormulateResubmitWorkflow(ctx, woc.wf, true, nil)
	require.NoError(t, err)
	dumpNodes(t, "after FormulateResubmitWorkflow(memoized)", newWf)
	requireNoDanglingReferences(t, newWf)

	cancel2, controller2 := newController(ctx, newWf)
	return cancel2, controller2, newWf
}

const memoizedFanoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: fanout
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: run
        withItems: [a, b, c]
        arguments:
          parameters:
          - name: item
            value: "{{item}}"
      - name: after
        template: run
        dependencies: [fan]
        arguments:
          parameters:
          - name: item
            value: after
  - name: run
    %s
    inputs:
      parameters:
      - name: item
    container:
      image: busybox
`

func TestMemoizedResubmitFanout(t *testing.T) {
	for name, retryStrategy := range map[string]string{
		"NoRetry":   "",
		"WithRetry": "retryStrategy: {limit: 1}",
	} {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(fmt.Sprintf(memoizedFanoutWf, retryStrategy))
			cancel, controller, newWf := memoizedResubmit(ctx, t, wf, failItemB)
			defer cancel()

			woc := runToCompletion(ctx, t, controller, newWf, allSucceed)
			dumpNodes(t, "resubmitted run finished", woc.wf)
			require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
			require.Len(t, podNames(ctx, woc), 2, "only b and after should run")
			for _, suffix := range []string{".fan(0:a)", ".fan(2:c)"} {
				n, err := woc.wf.GetNodeByName(newWf.Name + suffix)
				require.NoError(t, err)
				if retryStrategy == "" {
					require.Equal(t, wfv1.NodeTypeSkipped, n.Type, "%s should be memoized", suffix)
				} else {
					require.Equal(t, wfv1.NodeSucceeded, n.Phase, "%s should be kept as succeeded", suffix)
				}
			}
			for _, n := range woc.wf.Status.Nodes {
				require.NotEqual(t, wfv1.NodePending, n.Phase, "node %s left Pending", n.Name)
			}
		})
	}
}

// With limit: 1 a fresh run allows two attempts. Old attempts must not count against the
// resubmitted run's retry budget.
func TestMemoizedResubmitFanoutRetryBudget(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(fmt.Sprintf(memoizedFanoutWf, "retryStrategy: {limit: 1}"))
	cancel, controller, newWf := memoizedResubmit(ctx, t, wf, failItemB)
	defer cancel()

	bFailures := 0
	woc := runToCompletion(ctx, t, controller, newWf, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(node.Name, "(1:b)") && bFailures == 0 {
			bFailures++
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	dumpNodes(t, "resubmitted run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "b should have been allowed one retry, as in a fresh run")
	require.Len(t, podNames(ctx, woc), 3, "b(0), b(1) and after")
}

// Fanout over a nested DAG template (each item is a two-step DAG); the second step of b fails.
// Before memoized resubmit shared the retry reset logic this stayed Running forever: the omitted
// "after" node was deleted, but the leaf pods of every inner DAG still listed it as a child, so the
// inner DAGs' phase assessment never completed.
const memoizedNestedFanoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: nested
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: inner
        withItems: [a, b, c]
        arguments:
          parameters:
          - name: item
            value: "{{item}}"
      - name: after
        template: run
        dependencies: [fan]
        arguments:
          parameters:
          - name: item
            value: after
  - name: inner
    inputs:
      parameters:
      - name: item
    dag:
      tasks:
      - name: first
        template: run
        arguments:
          parameters:
          - name: item
            value: "{{inputs.parameters.item}}-first"
      - name: second
        template: run
        dependencies: [first]
        arguments:
          parameters:
          - name: item
            value: "{{inputs.parameters.item}}-second"
  - name: run
    inputs:
      parameters:
      - name: item
    container:
      image: busybox
`

func TestMemoizedResubmitNestedFanout(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(memoizedNestedFanoutWf)
	cancel, controller, newWf := memoizedResubmit(ctx, t, wf, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(node.Name, "(1:b).second") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	defer cancel()

	woc := runToCompletion(ctx, t, controller, newWf, allSucceed)
	dumpNodes(t, "resubmitted run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	require.Len(t, podNames(ctx, woc), 2, "only b.second and after should run")
	requireNoDanglingReferences(t, woc.wf)
}

// A "when"-skipped fanout item followed by a failed dependant. The skipped item is deleted from the
// memoized status and must be re-evaluated (and skipped again) rather than run.
const memoizedWhenFanoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: whenfan
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: run
        withItems: [a, b, c]
        when: "{{item}} != a"
        arguments:
          parameters:
          - name: item
            value: "{{item}}"
      - name: after
        template: run
        dependencies: [fan]
        arguments:
          parameters:
          - name: item
            value: after
  - name: run
    inputs:
      parameters:
      - name: item
    container:
      image: busybox
`

func TestMemoizedResubmitFanoutWhenSkipped(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(memoizedWhenFanoutWf)
	cancel, controller, newWf := memoizedResubmit(ctx, t, wf, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.HasSuffix(node.Name, ".after") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	defer cancel()

	woc := runToCompletion(ctx, t, controller, newWf, allSucceed)
	dumpNodes(t, "resubmitted run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	require.Len(t, podNames(ctx, woc), 1, "only 'after' should run; fan(0:a) must stay skipped")
}

// The same fanout expressed as steps with withItems, failing item b.
const memoizedStepsFanoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: stepsfan
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: fan
        template: inner
        withItems: [a, b, c]
        arguments:
          parameters:
          - name: item
            value: "{{item}}"
    - - name: after
        template: run
        arguments:
          parameters:
          - name: item
            value: after
  - name: inner
    inputs:
      parameters:
      - name: item
    steps:
    - - name: first
        template: run
        arguments:
          parameters:
          - name: item
            value: "{{inputs.parameters.item}}-first"
    - - name: second
        template: run
        arguments:
          parameters:
          - name: item
            value: "{{inputs.parameters.item}}-second"
  - name: run
    inputs:
      parameters:
      - name: item
    container:
      image: busybox
`

func TestMemoizedResubmitStepsNestedFanout(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(memoizedStepsFanoutWf)
	cancel, controller, newWf := memoizedResubmit(ctx, t, wf, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(node.Name, "(1:b)") && strings.Contains(node.Name, ".second") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	defer cancel()

	woc := runToCompletion(ctx, t, controller, newWf, allSucceed)
	dumpNodes(t, "resubmitted run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	require.Len(t, podNames(ctx, woc), 2, "only b.second and after should run")
	requireNoDanglingReferences(t, woc.wf)
}
