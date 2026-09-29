package controller

// Round 4 regression red tests, per pr-16290-round4-fix-plan/task-*.md.
// Shared helpers live in engine_regression_r4_helpers_test.go. These files
// must compile against wt-base (4389bbf96) too, so `basecheck` can overlay
// them there: use only functions/types present in both trees.

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/sync"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

// r4Parents returns the (sorted) names of every node that lists childName's
// node as a Children edge, i.e. childName's parents in the UI graph.
func r4Parents(woc *wfOperationCtx, childName string) []string {
	child, err := woc.wf.GetNodeByName(childName)
	if err != nil {
		return nil
	}
	var out []string
	for _, n := range woc.wf.Status.Nodes {
		for _, c := range n.Children {
			if c == child.ID {
				out = append(out, n.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// r4PodForNode matches the fake-clientset pod created for the named node.
func r4PodForNode(nodeName string) func(*apiv1.Pod) bool {
	return func(pod *apiv1.Pod) bool {
		return pod.Annotations[common.AnnotationKeyNodeName] == nodeName
	}
}

// r4WithReady marks every container of the pod Ready and Running, as a
// kubelet report for a live container would; this is what makes
// assessNodeStatus mark a daemon node Daemoned.
func r4WithReady(pod *apiv1.Pod, _ *wfOperationCtx) {
	pod.Status.ContainerStatuses = nil
	for _, c := range pod.Spec.Containers {
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, apiv1.ContainerStatus{
			Name:  c.Name,
			Ready: true,
			State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
		})
	}
}

const r4C79DAGDaemonRetry = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c79-dag-daemon-retry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: server
        template: dr
      - name: client
        template: c
        depends: server
      - name: last
        template: c
        depends: client
  - name: dr
    daemon: true
    retryStrategy:
      limit: 2
    container:
      image: busybox
      command: [sleep, "999999"]
  - name: c
    container:
      image: busybox
      command: [echo]
`

// TestRegressionR4_C79_DaemonRetryDependantParents ports
// TestProbe_v2x13_DAGDaemonRetryDependantParents (v2x13-1_test.go / C79).
// K8sTaskReconciler.Reconcile links every desired task on every Reconcile
// pass, not only the pass that creates its node. A running dependant
// (client) is therefore re-linked under its dependency's (server) current
// outbound nodes on every later pass: when the daemon dies and its retry
// creates server(1), client picks up server(1) as a second parent even
// though it started running under server(0). Base links a node only once,
// when it is first created.
func TestRegressionR4_C79_DaemonRetryDependantParents(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C79DAGDaemonRetry)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)

	const s0, s1, client, last = "r4-c79-dag-daemon-retry.server(0)", "r4-c79-dag-daemon-retry.server(1)", "r4-c79-dag-daemon-retry.client", "r4-c79-dag-daemon-retry.last"

	// server(0) comes up daemoned.
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(s0), r4WithReady)
	woc = r4Operate(t, ctx, controller, woc.wf)
	n, err := woc.wf.GetNodeByName(s0)
	require.NoError(t, err)
	require.True(t, n.IsDaemoned(), "server(0) should be daemoned")
	_, err = woc.wf.GetNodeByName(client)
	require.NoError(t, err, "client should have been created once server daemoned")

	// client starts, and its node is linked once under server(0).
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(client))
	woc = r4Operate(t, ctx, controller, woc.wf)
	require.Equal(t, []string{s0}, r4Parents(woc, client), "client should be linked only under server(0)")

	// server(0) dies; its retryStrategy creates server(1).
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode(s0), withExitCode(1))
	woc = r4Operate(t, ctx, controller, woc.wf)
	_, err = woc.wf.GetNodeByName(s1)
	require.NoError(t, err, "server(1) should have been created by the retry")

	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(s1), r4WithReady)
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}

	// client and last finish.
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(client))
	woc = r4Operate(t, ctx, controller, woc.wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(last))
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, []string{s0}, r4Parents(woc, client), "client must hang only under the attempt it started after")
	n1, err := woc.wf.GetNodeByName(s1)
	require.NoError(t, err)
	assert.Empty(t, n1.Children, "server(1) should have no children")
}

const r4C32DAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c32-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: slow
        template: echo
      - name: denied
        template: echo
      - name: after
        depends: denied
        template: echo
  - name: echo
    container:
      image: busybox
`

// r4HasParent reports whether any node lists childName's node as a child.
func r4HasParent(woc *wfOperationCtx, childName string) bool {
	child, err := woc.wf.GetNodeByName(childName)
	if err != nil {
		return false
	}
	for _, n := range woc.wf.Status.Nodes {
		if slices.Contains(n.Children, child.ID) {
			return true
		}
	}
	return false
}

// r4Reachable reports whether childName's node is reachable from the root
// node through Children edges, the way the UI graph is built.
func r4Reachable(woc *wfOperationCtx, childName string) bool {
	child, err := woc.wf.GetNodeByName(childName)
	if err != nil {
		return false
	}
	seen := map[string]bool{}
	queue := []string{woc.wf.NodeID(woc.wf.Name)}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		if id == child.ID {
			return true
		}
		queue = append(queue, woc.wf.Status.Nodes[id].Children...)
	}
	return false
}

// TestRegressionR4_C32_AdmissionDeniedLinkedAndRetryable ports
// TestProbe_v1x33_DAGAdmissionDeniedLinkedAndRetryable (v1x33-1_test.go /
// C32). When executeProcessedTemplate creates a task's node and then
// returns a non-throttle error (here, a pod-create rejection simulating an
// admission webhook denial), K8sTaskReconciler.Reconcile used to return
// before linkTasks, leaving the Error node unlinked from the DAG: the UI
// graph drops it and its dependants, and `argo retry` fails with "couldn't
// find parent node". Base links the node before handling any error from
// executing it.
func TestRegressionR4_C32_AdmissionDeniedLinkedAndRetryable(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C32DAG)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	r4RejectPodCreate(controller, func(pod *apiv1.Pod) bool {
		return pod.Annotations[common.AnnotationKeyNodeName] == "r4-c32-dag.denied"
	}, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, "denied",
		fmt.Errorf(`admission webhook "validation.gatekeeper.sh" denied the request: image not allowed`)))

	woc := r4Operate(t, ctx, controller, wf)
	// A few extra rounds with 'slow' still pending, to give the branch a
	// chance to converge before the sibling finishes.
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	require.True(t, r4Reachable(woc, "r4-c32-dag.denied"), "RUNNING: denied node is not reachable from the root (UI graph drops it)")

	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, func(*apiv1.Pod) bool { return true })
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}

	denied, err := woc.wf.GetNodeByName("r4-c32-dag.denied")
	require.NoError(t, err, "denied node missing")
	require.Equal(t, wfv1.NodeError, denied.Phase)
	require.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.True(t, r4HasParent(woc, "r4-c32-dag.denied"), "FINAL: denied node has no parent")
	assert.True(t, r4Reachable(woc, "r4-c32-dag.denied"), "FINAL: denied node is not reachable from the root (UI graph drops it)")
	assert.True(t, r4Reachable(woc, "r4-c32-dag.after"), "FINAL: dependant 'after' is not reachable from the root (UI graph drops it)")

	_, _, err = wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err, "argo retry")
	_, err = wfutil.FormulateResubmitWorkflow(ctx, woc.wf.DeepCopy(), true, nil)
	require.NoError(t, err, "argo resubmit --memoized")
}

// r4DriveToEnd succeeds every pod, then reconciles, until the workflow
// completes or rounds run out.
//
//nolint:revive // matches the r4 harness convention (t before ctx)
func r4DriveToEnd(t *testing.T, ctx context.Context, controller *WorkflowController, woc *wfOperationCtx, rounds int) *wfOperationCtx {
	t.Helper()
	for i := 0; i < rounds && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	return woc
}

// r4Unfulfilled lists the nodes that are not fulfilled, as "name=phase".
func r4Unfulfilled(woc *wfOperationCtx) []string {
	var out []string
	for _, n := range woc.wf.Status.Nodes {
		if !n.Fulfilled() {
			out = append(out, n.Name+"="+string(n.Phase))
		}
	}
	sort.Strings(out)
	return out
}

// r4MalformedNodes lists status entries with an empty key, name or type:
// `argo get` exits with "Missing node type in status node" on one and
// `argo retry` cannot find its parent.
func r4MalformedNodes(woc *wfOperationCtx) []string {
	var out []string
	for id, n := range woc.wf.Status.Nodes {
		if id == "" || n.Name == "" || n.Type == "" {
			out = append(out, fmt.Sprintf("key=%q name=%q type=%q phase=%s", id, n.Name, n.Type, n.Phase))
		}
	}
	sort.Strings(out)
	return out
}

const r4C17CIParallelism1 = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c17-ci
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 1
    dag:
      tasks:
      - name: build
        template: pipeline
      - name: deploy
        template: work
        depends: build
      - name: test
        template: pipeline
  - name: pipeline
    dag:
      tasks:
      - name: step
        template: work
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C17_CIParallelism1NestedDAG ports
// TestProbe_r1x16_CIParallelism1NestedDAG (r1x16-1_test.go / C17). With
// template parallelism 1, the dispatch pass stopped at the first task held
// back by parallelism (deploy, sorting before test), so the Running nested
// DAG test, which holds the only slot, was never dispatched again: its pod
// succeeded but test stayed Running and deploy was never created. Base
// continued past ErrParallelismReached.
func TestRegressionR4_C17_CIParallelism1NestedDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C17CIParallelism1)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	woc = r4DriveToEnd(t, ctx, controller, woc, 10)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

const r4C17ApprovalMutex = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c17-approval-mutex
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a-holder
        template: locked
      - name: b-gated
        template: gated
  - name: gated
    parallelism: 1
    dag:
      tasks:
      - name: approve
        template: approval
      - name: deploy
        template: work
        depends: approve
      - name: migrate
        template: locked
  - name: approval
    suspend: {}
  - name: locked
    container:
      image: busybox
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C17_ApprovalMutexWaiter ports
// TestProbe_v1x16_ApprovalMutexWaiter (v1x16-1_test.go / C17). migrate
// waits for a mutex held by a-holder and so holds the gated DAG's only
// parallelism slot while Pending. Once approve is resumed, deploy (sorting
// first) is held back by parallelism on every pass, and the pass stopped
// there, so migrate was never dispatched again to take the mutex once
// a-holder released it: the workflow hung. Base continued past
// ErrParallelismReached.
func TestRegressionR4_C17_ApprovalMutexWaiter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C17ApprovalMutex)
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == "locked" {
			wf.Spec.Templates[i].Synchronization = r4NamespacedMutex("r4-c17-m")
		}
	}
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	var err error
	controller.syncManager, err = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(string) {}, workflowExistenceFunc, false)
	require.NoError(t, err)

	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = r4Operate(t, ctx, controller, woc.wf)

	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	require.NoError(t, wfutil.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.Name, ""))
	woc = r4Operate(t, ctx, controller, woc.wf)
	woc = r4Operate(t, ctx, controller, woc.wf)

	woc = r4DriveToEnd(t, ctx, controller, woc, 15)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

const r4C31FanOut = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c31-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: echo
        withItems: [1, 2, 3]
      - name: sib
        template: echo
  - name: echo
    container:
      image: busybox
`

// TestRegressionR4_C31_FanOutItemPodRejected ports
// TestProbe_v1x32_DAGFanOutItemPodRejected (v1x32-1_test.go / C31). One
// item's pod is rejected by an admission webhook. The item's error was
// checked against the Running TaskGroup rather than the item, so it
// escalated to the boundary: the DAG ended Error in the first reconcile
// with the sibling's pod Pending, the later item was never created, and the
// rejected item's node was not linked under its TaskGroup. Base records the
// error on the item, lets the other items and the sibling run, and only
// then ends the DAG Error.
func TestRegressionR4_C31_FanOutItemPodRejected(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C31FanOut)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	r4RejectPodCreate(controller, r4PodForNode("r4-c31-dag.fan(1:2)"),
		apierr.NewBadRequest(`admission webhook "policy.example.com" denied the request`))

	woc := r4Operate(t, ctx, controller, wf)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "workflow ended while other pods were still running")
	assert.Equal(t, []string{"r4-c31-dag.fan"}, r4Parents(woc, "r4-c31-dag.fan(1:2)"), "rejected item not linked under its TaskGroup")

	woc = r4DriveToEnd(t, ctx, controller, woc, 5)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Empty(t, r4Unfulfilled(woc), "nodes left unfulfilled in a finished workflow")
	for _, name := range []string{"r4-c31-dag.fan(0:1)", "r4-c31-dag.fan(2:3)", "r4-c31-dag.sib"} {
		n, err := woc.wf.GetNodeByName(name)
		if assert.NoError(t, err, "node %s never created", name) {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase, name)
		}
	}
	rejected, err := woc.wf.GetNodeByName("r4-c31-dag.fan(1:2)")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, rejected.Phase)
	_, _, err = wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err, "argo retry")
}

const r4C31Semaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c31-sem
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: lock
        template: acquire
        withParam: '["1","2"]'
      - name: sib
        template: echo
  - name: acquire
    synchronization:
      semaphores:
        - configMapKeyRef:
            name: my-config
            key: template
    container:
      image: busybox
  - name: echo
    container:
      image: busybox
`

// TestRegressionR4_C31_FanOutMissingSemaphoreConfigMap ports
// TestProbe_v1x32_DAGFanOutMissingSemaphoreConfigMap (v1x32-1_test.go /
// C31). The items of an expanded task take a semaphore from a ConfigMap
// that does not exist. The first item's lock error escalated to the
// boundary, so the DAG ended Error in the first reconcile with the
// sibling's pod still Pending, and it stayed Pending in the finished
// workflow. Base records the error on each item and waits for the sibling.
func TestRegressionR4_C31_FanOutMissingSemaphoreConfigMap(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C31Semaphore)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "workflow ended while the sibling pod was running")

	woc = r4DriveToEnd(t, ctx, controller, woc, 5)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Empty(t, r4Unfulfilled(woc), "nodes left unfulfilled in a finished workflow")
	sib, err := woc.wf.GetNodeByName("r4-c31-sem.sib")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, sib.Phase)
}

const r4C56WFT = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: r4-c56-wft
  namespace: default
spec:
  templates:
  - name: t
    container:
      image: busybox
`

const r4C56DAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c56-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: first
        template: echo
      - name: A
        depends: first
        templateRef:
          name: r4-c56-wft
          template: t
      - name: B
        depends: first
        template: echo
  - name: echo
    container:
      image: busybox
`

// r4WaitForWFT waits until the WorkflowTemplate informer has (or no longer
// has) default/name.
func r4WaitForWFT(t *testing.T, controller *WorkflowController, name string, exists bool) {
	t.Helper()
	for range 2000 {
		if _, ok, _ := controller.wftmplInformer.Informer().GetStore().GetByKey("default/" + name); ok == exists {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("WorkflowTemplate %s: informer never reached exists=%v", name, exists)
}

// TestRegressionR4_C56_TemplateRefDeleted ports
// TestProbe_v1x34_DAGTemplateRefDeleted (v1x34-1_test.go / C56). The
// WorkflowTemplate behind task A's templateRef is deleted before A is
// dispatched. createDesiredTask marked a node that did not exist yet, so
// markNodePhase wrote an empty NodeStatus under key "", A got no node, and
// the error escalated: the DAG ended Error while B's pod was Pending, and
// `argo retry` failed even after the template was restored. Base records A
// as an Error node linked under first, lets B finish, and retry works.
func TestRegressionR4_C56_TemplateRefDeleted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C56DAG)
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(r4C56WFT))
	defer cancel()
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(wf.Namespace))
	require.NoError(t, validate.Workflow(ctx, wftmplGetter, nil, wf.DeepCopy(), nil, validate.Opts{}))

	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	wftmpls := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default")
	require.NoError(t, wftmpls.Delete(ctx, "r4-c56-wft", metav1.DeleteOptions{}))
	r4WaitForWFT(t, controller, "r4-c56-wft", false)

	woc = r4Operate(t, ctx, controller, woc.wf)
	a, err := woc.wf.GetNodeByName("r4-c56-dag.A")
	if assert.NoError(t, err, "task A has no node") {
		assert.Equal(t, wfv1.NodeError, a.Phase)
		assert.Equal(t, []string{"r4-c56-dag.first"}, r4Parents(woc, "r4-c56-dag.A"), "A not linked under its dependency")
	}
	if b, getErr := woc.wf.GetNodeByName("r4-c56-dag.B"); getErr == nil && !b.Fulfilled() {
		dagNode, getErr := woc.wf.GetNodeByName("r4-c56-dag")
		require.NoError(t, getErr)
		assert.False(t, dagNode.Fulfilled(), "DAG is %s while B is %s", dagNode.Phase, b.Phase)
	}
	assert.Empty(t, r4MalformedNodes(woc), "malformed nodes in status.nodes")

	woc = r4DriveToEnd(t, ctx, controller, woc, 3)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	b, err := woc.wf.GetNodeByName("r4-c56-dag.B")
	require.NoError(t, err, "B never created")
	assert.Equal(t, wfv1.NodeSucceeded, b.Phase)
	assert.Empty(t, r4MalformedNodes(woc), "malformed nodes in status.nodes")

	// argo retry once the template is restored.
	_, err = wftmpls.Create(ctx, wfv1.MustUnmarshalWorkflowTemplate(r4C56WFT), metav1.CreateOptions{})
	require.NoError(t, err)
	r4WaitForWFT(t, controller, "r4-c56-wft", true)
	retried, _, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err, "argo retry")
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, retried, metav1.UpdateOptions{})
	require.NoError(t, err)
	woc = r4Operate(t, ctx, controller, retried)
	woc = r4DriveToEnd(t, ctx, controller, woc, 4)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

const r4C61Omitted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c61-omitted
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail
      - name: B
        depends: A.Succeeded
        template: echo
  - name: fail
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hi"]
`

const r4C61TaskGroup = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c61-taskgroup
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        withItems: [1, 2]
  - name: echo
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo {{item}}"]
`

const r4C61RefWFT = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: r4-c61-wft
  namespace: default
spec:
  templates:
  - name: inner
    dag:
      tasks:
      - name: A
        template: fail
      - name: B
        depends: A.Succeeded
        template: echo
  - name: fail
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hi"]
`

const r4C61Ref = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c61-ref
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: X
        templateRef:
          name: r4-c61-wft
          template: inner
`

// TestRegressionR4_C61_OmittedAndTaskGroupTemplateName ports
// TestProbe_v1x21_OmittedNodeTemplateName, _TaskGroupTemplateName and
// _TemplateRefOmittedNode (v1x21-1_test.go / C61). The nodes the Engine
// creates itself (Omitted, TaskGroup, "Skipped, empty params", terminal
// Error) recorded the enclosing DAG/Steps template instead of the task's
// own: templateName main, or, under a DAG reached through templateRef, the
// boundary's templateRef and no templateName. Base gives each node its
// task's template.
func TestRegressionR4_C61_OmittedAndTaskGroupTemplateName(t *testing.T) {
	run := func(t *testing.T, manifest string, podPhase apiv1.PodPhase, rounds int, objects ...any) *wfOperationCtx {
		t.Helper()
		ctx := logging.TestContext(t.Context())
		wf := wfv1.MustUnmarshalWorkflow(manifest)
		cancel, controller := newController(ctx, append([]any{wf}, objects...)...)
		t.Cleanup(cancel)
		wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(wf.Namespace))
		require.NoError(t, validate.Workflow(ctx, wftmplGetter, nil, wf.DeepCopy(), nil, validate.Opts{}))
		woc := r4Operate(t, ctx, controller, wf)
		makePodsPhase(ctx, woc, podPhase)
		for range rounds {
			woc = r4Operate(t, ctx, controller, woc.wf)
		}
		return woc
	}
	node := func(t *testing.T, woc *wfOperationCtx, displayName string) *wfv1.NodeStatus {
		t.Helper()
		n := woc.wf.Status.Nodes.FindByDisplayName(displayName)
		require.NotNil(t, n, "node %q not found", displayName)
		return n
	}

	t.Run("OmittedNode", func(t *testing.T) {
		woc := run(t, r4C61Omitted, apiv1.PodFailed, 1)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		b := node(t, woc, "B")
		assert.Equal(t, wfv1.NodeOmitted, b.Phase)
		assert.Nil(t, b.TemplateRef)
		assert.Equal(t, "echo", b.TemplateName, "omitted node should record the task's template")
	})
	t.Run("TaskGroup", func(t *testing.T) {
		woc := run(t, r4C61TaskGroup, apiv1.PodSucceeded, 1)
		assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
		a := node(t, woc, "A")
		assert.Equal(t, wfv1.NodeTypeTaskGroup, a.Type)
		assert.Equal(t, "echo", a.TemplateName, "TaskGroup node should record the task's template")
	})
	t.Run("TemplateRefOmittedNode", func(t *testing.T) {
		woc := run(t, r4C61Ref, apiv1.PodFailed, 2, wfv1.MustUnmarshalWorkflowTemplate(r4C61RefWFT))
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		b := node(t, woc, "B")
		assert.Equal(t, wfv1.NodeOmitted, b.Phase)
		assert.Nil(t, b.TemplateRef, "omitted task uses template:, not templateRef")
		assert.Equal(t, "echo", b.TemplateName)
	})
}

const r4C22DAGNestedDependantOfDeadDaemon = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c22-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon
      - name: B
        template: inner
        dependencies: [A]
  - name: daemon
    daemon: true
    container:
      image: busybox
      command: [sleep, "999999"]
  - name: inner
    dag:
      tasks:
      - name: i1
        template: ok
      - name: i2
        template: ok
        depends: i1
  - name: ok
    container:
      image: busybox
      command: [echo, ok]
`

// TestRegressionR4_C22_NestedDependantOfDeadDaemon ports
// TestProbe_v1x14_DAGNestedDependantOfDeadDaemon (v1x14-1_test.go / C22). B
// (a nested DAG) depends on A (a daemon). Once A daemons, B starts and its
// inner task i1 runs. evaluateTaskResult/isReady re-evaluate the depends
// expression of every task whose node exists but is not yet fulfilled,
// including B, on every pass; once A's pod fails, "A.Succeeded ||
// A.Skipped || A.Daemoned" turns false and B, though already Running, gets
// ShouldRun=false and is never dispatched again, so its inner i2 (which
// depends on i1) is never created and the workflow hangs. Base keeps
// dispatching a task once its node exists, whatever its dependencies do
// next.
func TestRegressionR4_C22_NestedDependantOfDeadDaemon(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C22DAGNestedDependantOfDeadDaemon)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	// op mirrors the probe's v14Run.op(): a real kubelet reports Pending as
	// soon as it accepts a pod, but the fake clientset leaves a freshly
	// created pod's phase empty, which the pod assessor treats as
	// "Unexpected pod phase" (a harness artefact, see r4MoveNewPodsPending).
	op := func() {
		r4MoveNewPodsPending(ctx, woc)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}

	const a, b, i1, i2 = "r4-c22-dag.A", "r4-c22-dag.B", "r4-c22-dag.B.i1", "r4-c22-dag.B.i2"

	// A comes up daemoned, which starts B and its first inner task i1.
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(a), r4WithReady)
	op()
	an, err := woc.wf.GetNodeByName(a)
	require.NoError(t, err)
	require.True(t, an.IsDaemoned(), "A should be daemoned")
	_, err = woc.wf.GetNodeByName(i1)
	require.NoError(t, err, "i1 should have been created once A daemoned")

	// i1 starts running, then A's pod dies while i1 is still in flight.
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(i1))
	op()
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode(a))
	op()
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(i1))
	for range 3 {
		op()
	}

	// i2 depends only on i1, which has Succeeded; B (i2's boundary) must
	// still be reconciled to create it, even though A (which B itself
	// depends on) died in the meantime.
	_, err = woc.wf.GetNodeByName(i2)
	require.NoError(t, err, "i2 should have been created after i1 Succeeded, even though B's own dependency A died")
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(i2))
	for range 3 {
		op()
	}

	bn, err := woc.wf.GetNodeByName(b)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, bn.Phase)
	assert.True(t, woc.wf.Status.Phase.Completed(), "workflow phase %s", woc.wf.Status.Phase)
}
