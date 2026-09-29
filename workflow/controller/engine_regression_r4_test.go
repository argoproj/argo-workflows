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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

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

// r4PodNodeNames lists the node names of the workflow's pods, sorted.
func r4PodNodeNames(ctx context.Context, t *testing.T, woc *wfOperationCtx) []string {
	t.Helper()
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	var names []string
	for _, p := range pods.Items {
		names = append(names, p.Annotations[common.AnnotationKeyNodeName])
	}
	sort.Strings(names)
	return names
}

// r4DeletePod deletes the pod of the named node, as a node loss or a force
// delete would, and waits for the pod informer to drop it.
func r4DeletePod(ctx context.Context, t *testing.T, woc *wfOperationCtx, nodeName string) {
	t.Helper()
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	found := false
	for _, pod := range pods.Items {
		if pod.Annotations[common.AnnotationKeyNodeName] != nodeName {
			continue
		}
		found = true
		require.NoError(t, woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}))
		key := pod.Namespace + "/" + pod.Name
		require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
			_, exists, err := woc.controller.PodController.TestingPodInformer().GetStore().GetByKey(key)
			return !exists, err
		}))
	}
	require.True(t, found, "no pod for %s", nodeName)
}

// r4Cycle reports a node that is its own descendant through Children, or "".
func r4Cycle(wf *wfv1.Workflow) string {
	state := map[string]int{}
	var visit func(id string) string
	visit = func(id string) string {
		switch state[id] {
		case 1:
			return id
		case 2:
			return ""
		}
		state[id] = 1
		for _, c := range wf.Status.Nodes[id].Children {
			if found := visit(c); found != "" {
				return found
			}
		}
		state[id] = 2
		return ""
	}
	for id := range wf.Status.Nodes {
		if found := visit(id); found != "" {
			return wf.Status.Nodes[found].Name
		}
	}
	return ""
}

const r4C4StepsFanOut = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c4-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 1
    steps:
    - - name: a
        template: gen
      - name: fan
        template: gen
        withItems: [p, q]
    - - name: use
        template: consume
        arguments:
          parameters:
          - name: in
            value: "{{steps.fan.outputs.parameters.out}}"
  - name: gen
    container:
      image: busybox
    outputs:
      parameters:
      - name: out
        valueFrom:
          path: /tmp/out
  - name: consume
    inputs:
      parameters:
      - name: in
    container:
      image: busybox
      args: ["{{inputs.parameters.in}}"]
`

// TestRegressionR4_C4_StepsFanOutThrottledConsumer ports
// TestProbe_v1x42_OwnStepsFanOutThrottledConsumer (v1x42-1_test.go / C4).
// Template parallelism 1 is taken by step a when the fan-out is first
// dispatched, so its TaskGroup is created with no item. Nothing expanded it
// again: the empty group was assessed Succeeded, its items never ran, and
// the consumer of its aggregated outputs waited forever. Base creates the
// held-back items on later reconciles.
func TestRegressionR4_C4_StepsFanOutThrottledConsumer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C4StepsFanOut)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	out := withOutputs(ctx, wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "out", Value: wfv1.AnyStringPtr("v")}}})

	woc := r4Operate(t, ctx, controller, wf)
	for i := 0; i < 12 && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded, out)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}

	assert.Equal(t, []string{"r4-c4-steps[0].a", "r4-c4-steps[0].fan(0:p)", "r4-c4-steps[0].fan(1:q)", "r4-c4-steps[1].use"}, r4PodNodeNames(ctx, t, woc))
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
	use, err := woc.wf.GetNodeByName("r4-c4-steps[1].use")
	if assert.NoError(t, err) && assert.NotNil(t, use.Inputs) {
		assert.Equal(t, `["v","v"]`, use.Inputs.Parameters[0].Value.String())
	}
}

const r4C18ParallelismLimitDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c18-plimit
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 2
    dag:
      tasks:
      - name: sleep
        template: sleep
        withItems: [a, b, c, d, e, f]
  - name: sleep
    container:
      image: busybox
      command: [sh, -c, sleep 10]
`

// TestRegressionR4_C18_LongItemHoldsBackWindowDAG ports
// TestProbe_v1x72_LongItemHoldsBackWindowDAG (v1x72-1_test.go / C18). Item
// b runs long under template parallelism 2 while every other item finishes
// as soon as it has a pod. The missing items were only created when the
// whole group was dispatched, which needed every created item to be
// finished, so b held back the rest of the fan-out. Base fills each freed
// slot on the next reconcile (a sliding window).
func TestRegressionR4_C18_LongItemHoldsBackWindowDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C18ParallelismLimitDAG)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	for range 8 {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if strings.Contains(n.Name, "(1:b)") {
				return apiv1.PodRunning
			}
			return apiv1.PodSucceeded
		})
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	assert.Len(t, r4PodNodeNames(ctx, t, woc), 6, "every item should have had a pod while the long item b still runs")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	woc = r4DriveToEnd(t, ctx, controller, woc, 4)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

const r4C1RetryOmittedRetryTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c1-retry-omitted
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: echo
        withItems: [p]
      - name: b
        template: retried
        depends: a
  - name: echo
    container:
      image: alpine
      command: [echo, hi]
  - name: retried
    retryStrategy:
      limit: 1
    container:
      image: alpine
      command: [echo, hi]
`

// TestRegressionR4_C1_RetryWorkflowWithOmittedRetryTask ports
// TestProbe_v3x13_RetryWorkflowWithOmittedRetryTask (v3x13-1_test.go / C1).
// The fan-out's only item fails, which omits its dependant b, and `argo
// retry` deletes the item node. The now childless TaskGroup was assessed
// Succeeded without re-running the item, b was linked under it, and on the
// next pass b was linked under its own attempt: b -> b(0) -> b, a cycle the
// next reconcile recursed on until the controller died. Base re-runs the
// item and b, keeps the graph acyclic and Succeeds.
func TestRegressionR4_C1_RetryWorkflowWithOmittedRetryTask(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C1RetryOmittedRetryTask)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	op := func(woc *wfOperationCtx) *wfOperationCtx {
		woc = r4Operate(t, ctx, controller, woc.wf)
		r4MoveNewPodsPending(ctx, woc)
		return woc
	}

	woc := r4Operate(t, ctx, controller, wf)
	r4MoveNewPodsPending(ctx, woc)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	for range 2 {
		woc = op(woc)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	b := woc.wf.Status.Nodes.FindByDisplayName("b")
	require.NotNil(t, b)
	require.Equal(t, wfv1.NodeOmitted, b.Phase)

	// argo retry
	retried, podsToDelete, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err)
	require.Empty(t, r4Cycle(retried), "node graph after argo retry has a cycle")
	for _, p := range podsToDelete {
		require.NoError(t, controller.kubeclientset.CoreV1().Pods(wf.Namespace).Delete(ctx, p, metav1.DeleteOptions{}))
	}
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, retried, metav1.UpdateOptions{})
	require.NoError(t, err)

	woc.wf = retried
	for i := range 4 {
		woc = op(woc)
		// A cycle here makes the next reconcile's childrenFulfilled call
		// recurse until the controller dies; stop before that.
		require.Empty(t, r4Cycle(woc.wf), "after reconcile %d the node graph has a cycle", i)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
	for _, name := range []string{"r4-c1-retry-omitted.a(0:p)", "r4-c1-retry-omitted.b"} {
		n, err := woc.wf.GetNodeByName(name)
		if assert.NoError(t, err, name) {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase, name)
		}
	}
}

const r4C11DAGItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c11-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: work
        arguments:
          parameters:
          - name: v
            value: "{{item}}"
        withItems: [p, q]
      - name: B
        template: ok
        depends: A
  - name: ok
    container:
      image: alpine
      command: [sh, -c, "exit 0"]
  - name: work
    inputs:
      parameters:
      - name: v
        value: "x"
    container:
      image: alpine
      command: [sh, -c, "echo {{inputs.parameters.v}}"]
`

// r4IncompleteTaskResult is what the wait container writes as soon as its
// pod starts: an incomplete WorkflowTaskResult for the pod's node.
func r4IncompleteTaskResult(ctx context.Context) with {
	return func(pod *apiv1.Pod, woc *wfOperationCtx) {
		nodeID := woc.nodeID(pod)
		trs := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(woc.wf.Namespace)
		if _, err := trs.Get(ctx, nodeID, metav1.GetOptions{}); err == nil {
			return
		}
		created, err := trs.Create(ctx, &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{
			Name: nodeID,
			Labels: map[string]string{
				common.LabelKeyWorkflow:               woc.wf.Name,
				common.LabelKeyReportOutputsCompleted: "false",
			},
		}}, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		waitForInformer(ctx, woc.controller.taskResultInformer, created, func(any) bool { return true })
	}
}

// r4CompleteTaskResult marks the named node's WorkflowTaskResult complete,
// as the wait container does once it has reported the outputs.
func r4CompleteTaskResult(ctx context.Context, t *testing.T, woc *wfOperationCtx, nodeName string) {
	t.Helper()
	trs := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(woc.wf.Namespace)
	tr, err := trs.Get(ctx, woc.wf.NodeID(nodeName), metav1.GetOptions{})
	require.NoError(t, err)
	tr.Labels[common.LabelKeyReportOutputsCompleted] = "true"
	updated, err := trs.Update(ctx, tr, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, woc.controller.taskResultInformer, updated, func(obj any) bool {
		return obj.(*wfv1.WorkflowTaskResult).Labels[common.LabelKeyReportOutputsCompleted] == "true"
	})
}

// TestRegressionR4_C11_ItemPodDeleted ports TestProbe_v3x6_DAGItemPodDeleted
// (v3x6-1_test.go / C11). The running pod of item A(0:p) is deleted. Only
// Pending items and running nested DAG/Steps items were dispatched again,
// so the item's pod was never recreated: A(0:p) ended Error "pod deleted",
// B was Omitted and the workflow ended Error. Base re-enters every
// unfinished item on each reconcile, recreates the pod and Succeeds.
func TestRegressionR4_C11_ItemPodDeleted(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C11DAGItems)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	const lost, other, b = "r4-c11-dag.A(0:p)", "r4-c11-dag.A(1:q)", "r4-c11-dag.B"

	woc := r4Operate(t, ctx, controller, wf)
	op := func() {
		r4MoveNewPodsPending(ctx, woc)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, func(*apiv1.Pod) bool { return true }, r4IncompleteTaskResult(ctx))
	op()
	r4DeletePod(ctx, t, woc, lost)
	for range 3 {
		op()
	}
	require.Contains(t, r4PodNodeNames(ctx, t, woc), lost, "the deleted item's pod was not recreated")

	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(lost))
	op()
	for _, name := range []string{lost, other} {
		r4CompleteTaskResult(ctx, t, woc, name)
		r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(name), withExitCode(0))
	}
	for range 2 {
		op()
	}
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(b))
	for range 2 {
		op()
	}

	for _, name := range []string{lost, b} {
		n, err := woc.wf.GetNodeByName(name)
		if assert.NoError(t, err, name) {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase, name)
		}
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

const r4C26ItemsSuspendSteps = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c26-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: wait
        withItems: [p, q]
  - name: wait
    suspend:
      duration: "30"
`

// TestRegressionR4_C26_ItemsSuspendDurationSteps ports
// TestProbe_v1x44_ItemsSuspendDurationSteps (v1x44-1_test.go / C26),
// backdating the items' start instead of sleeping. A suspend item with a
// duration was never dispatched again, so it never saw its duration pass
// and the workflow hung. Base re-enters every unfinished item on each
// reconcile and resumes them.
func TestRegressionR4_C26_ItemsSuspendDurationSteps(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C26ItemsSuspendSteps)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	wfcs := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	stored, err := wfcs.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	suspends := 0
	for id, n := range stored.Status.Nodes {
		if n.Type == wfv1.NodeTypeSuspend {
			n.StartedAt = metav1.NewTime(time.Now().Add(-time.Minute))
			stored.Status.Nodes[id] = n
			suspends++
		}
	}
	require.Equal(t, 2, suspends, "both suspend items should exist")
	_, err = wfcs.Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)

	for i := 0; i < 5 && !woc.wf.Status.Phase.Completed(); i++ {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	for _, n := range woc.wf.Status.Nodes {
		if n.Type == wfv1.NodeTypeSuspend {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase, n.Name)
		}
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

const r4C21ExpandedDaemonsDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c21
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: srv
        template: server
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withItems: [a, b]
      - name: client
        depends: srv
        template: work
  - name: server
    daemon: true
    inputs:
      parameters:
      - name: msg
    container:
      image: busybox
  - name: work
    container:
      image: busybox
`

const r4C21ExpandedDaemonsSteps = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c21
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: srv
        template: server
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withItems: [a, b]
    - - name: client
        template: work
  - name: server
    daemon: true
    inputs:
      parameters:
      - name: msg
    container:
      image: busybox
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C21_C77_ExpandedDaemons ports
// TestProbe_v1x41_ExpandedSameOperate (v1x41-1_test.go / C77) and
// TestProbe_r1x13_DAGWithParamDaemons (r1x13-1_test.go / C21) into one
// scenario per template type. The items of an expanded task are daemons.
// Each item was reported Running and not fulfilled for dependants, so the
// dependant started one reconcile late (C77) and, once it finished, the
// DAG/Steps stayed Running forever with the daemons alive (C21). Base
// starts the dependant in the reconcile the daemons become ready, then
// Succeeds and kills the daemons.
func TestRegressionR4_C21_C77_ExpandedDaemons(t *testing.T) {
	for _, tc := range []struct{ kind, manifest, client string }{
		{"dag", r4C21ExpandedDaemonsDAG, "r4-c21.client"},
		{"steps", r4C21ExpandedDaemonsSteps, "r4-c21[1].client"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(tc.manifest)
			require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
			cancel, controller := newController(ctx, wf)
			defer cancel()

			woc := r4Operate(t, ctx, controller, wf)
			require.Len(t, r4PodNodeNames(ctx, t, woc), 2)
			// Both daemons come up ready: the client starts in this reconcile.
			r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, func(*apiv1.Pod) bool { return true }, r4WithReady)
			woc = r4Operate(t, ctx, controller, woc.wf)
			_, err := woc.wf.GetNodeByName(tc.client)
			require.NoError(t, err, "client should start in the reconcile its daemons became ready (C77)")

			// The client finishes: the template completes and the daemons are killed.
			r4MoveNewPodsPending(ctx, woc)
			r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(tc.client))
			for i := 0; i < 5 && !woc.wf.Status.Phase.Completed(); i++ {
				woc = r4Operate(t, ctx, controller, woc.wf)
			}
			assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
			for _, n := range woc.wf.Status.Nodes {
				if n.TemplateName == "server" {
					assert.False(t, n.IsDaemoned(), "%s should have been killed", n.Name)
				}
			}
		})
	}
}

const r4C5MissingOutputFromFailedStep = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c5-miss-out
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: gen
        continueOn:
          failed: true
    - - name: consume
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{item}}-{{steps.gen.outputs.parameters.p}}"
        withItems: [a, b]
  - name: gen
    container:
      image: alpine
      command: [sh, -c]
      args: ["exit 1"]
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

// TestRegressionR4_C5_ExpandedStepMissingOutputFromFailedStep ports
// TestProbe_v3x9_ExpandedStepMissingOutputFromFailedStep (v3x9-1_test.go /
// C5, lead 3). The items of an expanded step reference an output that the
// failed (continueOn) step never produced, so no item can be created. The
// childless TaskGroup was then assessed Succeeded and the workflow
// Succeeded without running any item. Base waits for the reference (the
// workflow stays Running); either way it must not Succeed without them.
func TestRegressionR4_C5_ExpandedStepMissingOutputFromFailedStep(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C5MissingOutputFromFailedStep)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodFailed, withExitCode(1))
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	hasConsume := slices.ContainsFunc(r4PodNodeNames(ctx, t, woc), func(n string) bool { return strings.Contains(n, "consume") })
	if woc.wf.Status.Phase == wfv1.WorkflowSucceeded {
		assert.True(t, hasConsume, "workflow Succeeded but no consume item ran")
	}
}

const r4C17FanOutNestedDAGs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c17-fanout
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 2
    dag:
      tasks:
      - name: prepare
        template: pipeline
      - name: notify
        template: work
        depends: prepare
      - name: process
        template: pipeline
        withItems: [x, y, z]
  - name: pipeline
    dag:
      tasks:
      - name: step
        template: work
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C17_FanOutNestedDAGs ports TestProbe_r1x16_FanOutNestedDAGs
// (r1x16-1_test.go), a guard that passes before and after the TaskGroup
// dispatch change: under template parallelism 2, a task held back by
// parallelism (notify) must not stop the fan-out's nested DAG items from
// being re-entered, or the workflow deadlocks.
func TestRegressionR4_C17_FanOutNestedDAGs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C17FanOutNestedDAGs)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	woc = r4DriveToEnd(t, ctx, controller, woc, 14)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

const r4Lead8FanOutTimeouts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-l8-fan
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    failFast: false
    dag:
      tasks:
      - {name: fan, template: slow, withItems: [0, 1, 2, 3, 4]}
      - {name: x, template: c}
      - {name: yy, template: c, depends: x}
  - name: c
    container: {image: busybox, command: [echo]}
  - name: slow
    inputs: {parameters: [{name: item, value: "{{item}}"}]}
    pendingTimeout: 1s
    container: {image: busybox, command: [echo, "{{inputs.parameters.item}}"]}
`

// TestRegressionR4_Lead8_DAGFanOutTimeoutsHeadBetter ports
// TestProbe_lead8_DAGFanOutTimeoutsHeadBetter (lead8-1_test.go), a guard
// for behaviour the branch does better than base (base fails it: it times
// out one item per reconcile). Five items hit their pendingTimeout in the
// same reconcile: each timeout stays on its own item, every item ends
// Failed "timeout", and yy is still created in that reconcile.
func TestRegressionR4_Lead8_DAGFanOutTimeoutsHeadBetter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4Lead8FanOutTimeouts)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.HasSuffix(n.Name, ".x") {
			return apiv1.PodSucceeded
		}
		return apiv1.PodPending
	})
	time.Sleep(1500 * time.Millisecond)
	woc = r4Operate(t, ctx, controller, woc.wf)
	failed := 0
	for _, n := range woc.wf.Status.Nodes {
		if strings.Contains(n.Name, ".fan(") && n.Type == wfv1.NodeTypePod {
			assert.Equal(t, wfv1.NodeFailed, n.Phase, n.Name)
			assert.Equal(t, "timeout", n.Message, n.Name)
			failed++
		}
	}
	assert.Equal(t, 5, failed)
	require.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("yy"), "yy created in the same reconcile")

	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode("r4-l8-fan.yy"))
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	yy := woc.wf.Status.Nodes.FindByDisplayName("yy")
	require.NotNil(t, yy)
	assert.Equal(t, wfv1.NodeSucceeded, yy.Phase)
}

const r4P3RetryEmptiesFanOut = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p3-retry-empty
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: list
      value: '["a","b"]'
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: echo
        withParam: "{{workflow.parameters.list}}"
  - name: echo
    container:
      image: alpine
      command: [echo, hi]
`

// TestRegressionR4_P3_RetryParameterEmptiesFanOut encodes decision P3, not
// a base behaviour (base could panic here). One item of the fan-out fails,
// and `argo retry --parameter list=[]` resets its TaskGroup to Running with
// only the succeeded item left and no items to expand into. The branch
// marked the group Succeeded with no message, by the same empty-group
// assessment as C4; once the group is dispatched until it finishes, its
// dispatch would create the "Skipped, empty params" node over the existing
// group and panic. It is now completed as an empty expansion would be:
// Skipped where its phase allows it, otherwise (a Running group) Succeeded,
// with the same message; the workflow Succeeds.
func TestRegressionR4_P3_RetryParameterEmptiesFanOut(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4P3RetryEmptiesFanOut)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode("r4-p3-retry-empty.fan(0:a)"))
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode("r4-p3-retry-empty.fan(1:b)"))
	for range 2 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	retried, podsToDelete, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", []string{"list=[]"})
	require.NoError(t, err)
	for _, p := range podsToDelete {
		require.NoError(t, controller.kubeclientset.CoreV1().Pods(wf.Namespace).Delete(ctx, p, metav1.DeleteOptions{}))
	}
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, retried, metav1.UpdateOptions{})
	require.NoError(t, err)

	for range 2 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	fan, err := woc.wf.GetNodeByName("r4-p3-retry-empty.fan")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeTypeTaskGroup, fan.Type)
	assert.Equal(t, wfv1.NodeSucceeded, fan.Phase)
	assert.Equal(t, "Skipped, empty params", fan.Message)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

// r4PodStartOrder drives the workflow to completion, one round at a time,
// rebuilding the wfOperationCtx from stored status every round (r4Operate).
// Each round it records the display names of pods created that round
// (sorted within the round, since they started together), then sets pods
// named in failing to Failed and everything else to Succeeded before
// re-operating. Matches v1x20RunOrder / probeR1x20Run (v1x20-1_test.go,
// r1x20-1_test.go).
//
//nolint:revive // matches the r4 harness convention (t before ctx)
func r4PodStartOrder(t *testing.T, ctx context.Context, controller *WorkflowController, woc *wfOperationCtx, rounds int, failing map[string]bool) ([]string, *wfOperationCtx) {
	t.Helper()
	seen := map[string]bool{}
	var order []string
	for i := 0; i < rounds && !woc.wf.Status.Phase.Completed(); i++ {
		var round []string
		for _, n := range woc.wf.Status.Nodes {
			if n.Type == wfv1.NodeTypePod && !seen[n.ID] {
				seen[n.ID] = true
				round = append(round, n.DisplayName)
			}
		}
		sort.Strings(round)
		order = append(order, round...)
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if failing[n.DisplayName] {
				return apiv1.PodFailed
			}
			return apiv1.PodSucceeded
		})
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	return order, woc
}

const r4C73StepsFailFastValidateApply = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c73-steps-ff
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 1
    failFast: true
    steps:
    - - name: validate
        template: work
      - name: apply
        template: work
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C73_StepsFailFastValidateApply ports
// TestProbe_r1x20_StepsFailFastValidateApply (r1x20-1_test.go / C73).
// HEAD's converge dispatched ready tasks in sorted-key order, so under template
// parallelism 1 "apply" (sorts before "validate") could win the only slot
// ahead of "validate", defeating failFast. Base dispatches Steps in
// declaration order, so "validate" always runs first; once it fails,
// failFast must stop "apply" from ever starting. Fixed by the ordered walk
// (T1.5), which dispatches Steps as written.
func TestRegressionR4_C73_StepsFailFastValidateApply(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C73StepsFailFastValidateApply)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	order, woc := r4PodStartOrder(t, ctx, controller, woc, 8, map[string]bool{"validate": true})
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("apply"), "apply must not start after validate failed")
	assert.Equal(t, []string{"validate"}, order)
}

const r4C73DAGTemplateParallelismOrder = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c73-dag-order
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 1
    dag:
      tasks:
      - name: b
        template: work
      - name: a
        template: work
        dependencies: [z]
      - name: z
        template: work
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C73_DAGTemplateParallelismOrder ports
// TestProbe_v1x20_DAGTemplateParallelismOrder (v1x20-1_test.go / C73).
// Under DAG template parallelism 1, base walked from the leaf (z's chain
// first, dependency order, then the independent b): z, a, b. HEAD's
// sorted-key dispatch instead started whichever ready task sorts first by
// name, alphabetically: a and b tie for readiness before z finishes, but a
// depends on z so only b and z are ready first, and b < z alphabetically.
// Fixed by the ordered walk (T1.5) over dag.PullOrder's order (T1.6).
func TestRegressionR4_C73_DAGTemplateParallelismOrder(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C73DAGTemplateParallelismOrder)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	order, woc := r4PodStartOrder(t, ctx, controller, woc, 8, nil)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, []string{"z", "a", "b"}, order)
}

const r4C73StepGroupChildrenOrder = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c73-sg-children
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: zeta
        template: work
      - name: mid
        template: work
      - name: alpha
        template: work
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C73_StepGroupChildrenOrder ports
// TestProbe_v1x20_StepGroupChildrenOrder (v1x20-1_test.go / C73). Without
// parallelism every step in the group is ready at once, but HEAD's
// converge still dispatched (and so linked, via addChildNode) in
// sorted-key order: alpha, mid, zeta instead of the declared zeta, mid,
// alpha. The StepGroup's Children order drives the UI's collapsed-view
// first/last step. Fixed by the ordered walk (T1.5).
func TestRegressionR4_C73_StepGroupChildrenOrder(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C73StepGroupChildrenOrder)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	sg, err := woc.wf.GetNodeByName("r4-c73-sg-children[0]")
	require.NoError(t, err)
	var names []string
	for _, id := range sg.Children {
		n, err := woc.wf.Status.Nodes.Get(id)
		require.NoError(t, err)
		names = append(names, n.DisplayName)
	}
	assert.Equal(t, []string{"zeta", "mid", "alpha"}, names)
}

const r4C54ChainedOmittedReverseOrder = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c54-chain
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: C
        template: work
        depends: B.Succeeded
      - name: B
        template: work
        depends: A.Succeeded
      - name: A
        template: work
  - name: work
    container:
      image: busybox
`

// TestRegressionR4_C54_ChainedOmittedReverseOrderLinked ports
// TestProbe_v1x17_ChainedOmittedReverseOrderLinked (v1x17-1_test.go / C54).
// C depends on B depends on A, declared in that (dependant-first) order; A
// fails, omitting B and then C in the same pass. createOmittedNodes walked
// tasks in declaration order, so it tried to link C under B before B's own
// node existed, leaving C permanently unlinked ("couldn't find parent
// node" on retry; dropped from the UI graph). dag.PullOrder makes the
// Engine visit A, then B, then C: each Omitted node's dependency node
// already exists when it is created and linked.
func TestRegressionR4_C54_ChainedOmittedReverseOrderLinked(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C54ChainedOmittedReverseOrder)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode("r4-c54-chain.A"))
	woc = r4Operate(t, ctx, controller, woc.wf)
	woc = r4Operate(t, ctx, controller, woc.wf)

	b, err := woc.wf.GetNodeByName("r4-c54-chain.B")
	require.NoError(t, err)
	c, err := woc.wf.GetNodeByName("r4-c54-chain.C")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeOmitted, b.Phase)
	assert.Equal(t, wfv1.NodeOmitted, c.Phase)
	assert.Equal(t, []string{"r4-c54-chain.A"}, r4Parents(woc, "r4-c54-chain.B"))
	assert.Equal(t, []string{"r4-c54-chain.B"}, r4Parents(woc, "r4-c54-chain.C"), "C must hang off B, not be left unlinked")
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestRegressionR4_C54_ChainedOmittedReverseOrderRetry ports
// TestProbe_v1x17_ChainedOmittedReverseOrderRetry (v1x17-1_test.go / C54).
// Same chain as Linked; after `argo retry`, once A and then B succeed, C
// must run (not stay Omitted). At HEAD FormulateRetryWorkflow's graph walk
// hits C's missing parent link and the fix must make retry proceed to a
// fresh C.
func TestRegressionR4_C54_ChainedOmittedReverseOrderRetry(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C54ChainedOmittedReverseOrder)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode("r4-c54-chain.A"))
	woc = r4Operate(t, ctx, controller, woc.wf)
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	retried, podsToDelete, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err, "argo retry")
	for _, p := range podsToDelete {
		require.NoError(t, controller.kubeclientset.CoreV1().Pods(wf.Namespace).Delete(ctx, p, metav1.DeleteOptions{}))
	}
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, retried, metav1.UpdateOptions{})
	require.NoError(t, err)

	woc = r4Operate(t, ctx, controller, woc.wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode("r4-c54-chain.A"))
	woc = r4Operate(t, ctx, controller, woc.wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode("r4-c54-chain.B"))
	woc = r4Operate(t, ctx, controller, woc.wf)

	c, err := woc.wf.GetNodeByName("r4-c54-chain.C")
	require.NoError(t, err)
	assert.NotEqual(t, wfv1.NodeOmitted, c.Phase, "C must run once B succeeds, not stay Omitted")

	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode("r4-c54-chain.C"))
	woc = r4Operate(t, ctx, controller, woc.wf)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

// r4OneCreatePerPod runs manifest to completion under a 10ms pod-watch lag,
// reconciling up to eight times, and fails if any pod gets more than one
// Create call in one reconcile or any Create is answered AlreadyExists. The
// lag keeps a pod created earlier in a reconcile out of the pod informer, so
// a second dispatch of its still-Pending node in the same reconcile calls
// the API server's pod Create again, as lead 5 (C90) describes.
func r4OneCreatePerPod(t *testing.T, manifest string) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf, func(c *WorkflowController) { r4DelayPodWatch(c, 10*time.Millisecond) })
	defer cancel()
	calls := r4CountPodCalls(controller)

	woc := r4Operate(t, ctx, controller, wf)
	for round := 0; ; round++ {
		calls.mu.Lock()
		for name, n := range calls.creates {
			assert.Equal(t, 1, n, "round %d: pod %s: Create calls in one reconcile", round, name)
		}
		assert.Zero(t, calls.alreadyExists, "round %d: AlreadyExists responses from pod Create", round)
		calls.creates, calls.alreadyExists = map[string]int{}, 0
		calls.mu.Unlock()
		if woc.wf.Status.Phase.Completed() || round == 8 {
			break
		}
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "nodes left unfulfilled: %v", r4Unfulfilled(woc))
}

// TestRegressionR4_C90_NestedOneCreatePerPod ports
// TestProbe_lead5_NestedOneCreatePerPodPerReconcile (lead5-1_test.go /
// C90): a pod three Steps/DAG levels down. The fixed-point loop re-enters
// each running nested level on a later pass of the level above (C20), and
// each re-entry dispatches the still-Pending pod node again. Base visits
// each task once per reconcile.
func TestRegressionR4_C90_NestedOneCreatePerPod(t *testing.T) {
	r4OneCreatePerPod(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c90-nested
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - {name: l1, template: n1}
  - name: n1
    dag:
      tasks:
      - {name: l2, template: n2}
  - name: n2
    steps:
    - - {name: l3, template: n3}
  - name: n3
    steps:
    - - {name: leaf, template: c}
  - name: c
    container:
      image: busybox
      command: [sh, -c, "true"]
`)
}

// TestRegressionR4_C90_ItemsOneCreatePerPod ports
// TestProbe_lead5_ItemsOneCreatePerPodPerReconcile (lead5-1_test.go /
// C90): a withItems fan-out whose TaskGroup is dispatched on every pass of
// the fixed-point loop, re-entering its still-Pending items.
func TestRegressionR4_C90_ItemsOneCreatePerPod(t *testing.T) {
	r4OneCreatePerPod(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c90-items
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: c
        withItems: [1, 2, 3]
  - name: c
    container:
      image: busybox
      command: [sh, -c, "true"]
`)
}

// r4NestedChain is a workflow whose entrypoint descends through depth nested
// Steps (kind "steps") or DAG (kind "dag") templates to a single pod.
func r4NestedChain(name, kind string, depth int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: argoproj.io/v1alpha1\nkind: Workflow\nmetadata:\n  name: %s\n  namespace: default\nspec:\n  entrypoint: l0\n  templates:\n", name)
	for i := range depth {
		next := fmt.Sprintf("l%d", i+1)
		if i == depth-1 {
			next = "leaf"
		}
		if kind == "steps" {
			fmt.Fprintf(&b, "  - name: l%d\n    steps:\n    - - name: s\n        template: %s\n", i, next)
		} else {
			fmt.Fprintf(&b, "  - name: l%d\n    dag:\n      tasks:\n      - name: t\n        template: %s\n", i, next)
		}
	}
	b.WriteString("  - name: leaf\n    container:\n      image: busybox\n      command: [sleep, \"10\"]\n")
	return b.String()
}

// TestRegressionR4_C20_NestedReconcileLinearInDepth ports
// TestProbe_tri9_NestedReconcileTimeLinearInDepth (tri9-1_test.go / C20),
// at depth 8. Every level of the fixed-point loop dispatches a running
// nested template once per pass, and it takes two passes to find nothing
// new, so a reconcile of a running chain doubles with each level. Base
// reconciles each level once: a few milliseconds at this depth.
func TestRegressionR4_C20_NestedReconcileLinearInDepth(t *testing.T) {
	for _, kind := range []string{"steps", "dag"} {
		t.Run(kind, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(r4NestedChain("r4-c20-"+kind, kind, 8))
			cancel, controller := newController(ctx, wf, func(c *WorkflowController) { c.maxOperationTime = time.Hour })
			defer cancel()
			woc := r4Operate(t, ctx, controller, wf)
			makePodsPhase(ctx, woc, apiv1.PodRunning)
			// The fastest of three reconciles of the same running state, so
			// a scheduling hiccup does not decide the result.
			took := time.Duration(1<<63 - 1)
			for range 3 {
				start := time.Now()
				woc = r4Operate(t, ctx, controller, woc.wf)
				took = min(took, time.Since(start))
				require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, woc.wf.Status.Message)
			}
			t.Logf("depth 8 %s: reconcile took %v", kind, took)
			assert.Less(t, took, r4C20Limit, "reconciling 8 nested %s levels with one running pod", kind)
		})
	}
}

const r4C6WhenStatusWithItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c6-items
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: echo
        withItems: [1, 2]
    - - name: b
        template: echo
        when: "{{steps.a.status}} == Succeeded"
  - name: echo
    container:
      image: alpine
      command: [echo]
`

// TestRegressionR4_C6_WhenStatusWithItems ports
// TestProbe_v1x23_WhenStatusWithItems (v1x23-1_test.go / C6). An expanded
// step's {{steps.a.status}} is its StepGroup's phase (scopeNodeForTask), so
// b's when clause must see group [0] recorded Succeeded before b is
// evaluated. HEAD records StepGroup phases only after the dispatch loop, so
// b sees "Running", is skipped, and the workflow reports Succeeded without
// running b.
func TestRegressionR4_C6_WhenStatusWithItems(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C6WhenStatusWithItems)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	woc = r4DriveToEnd(t, ctx, controller, woc, 6)
	b, err := woc.wf.GetNodeByName("r4-c6-items[1].b")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeTypePod, b.Type, "b: phase=%s msg=%s", b.Phase, b.Message)
	assert.Equal(t, wfv1.NodeSucceeded, b.Phase)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 3, "a(1), a(2) and b must each have run a pod")
}

const r4C6WhenStatusStaggered = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c6-stag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: echo
        withItems: [1, 2]
      - name: c
        template: echo
    - - name: b
        template: echo
        when: "{{steps.a.status}} == Succeeded"
  - name: echo
    container:
      image: alpine
      command: [echo]
`

// TestRegressionR4_C6_WhenStatusStaggered ports
// TestProbe_v1x23_WhenStatusStaggered (v1x23-1_test.go / C6): a's items
// finish a reconcile before their sibling c does; b must still run once the
// group has finished.
func TestRegressionR4_C6_WhenStatusStaggered(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C6WhenStatusStaggered)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = r4Operate(t, ctx, controller, woc.wf)
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(n.Name, ".a(") {
			return apiv1.PodSucceeded
		}
		return ""
	})
	woc = r4Operate(t, ctx, controller, woc.wf)
	woc = r4Operate(t, ctx, controller, woc.wf)
	woc = r4DriveToEnd(t, ctx, controller, woc, 6)
	b, err := woc.wf.GetNodeByName("r4-c6-stag[1].b")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeTypePod, b.Type, "b: phase=%s msg=%s", b.Phase, b.Message)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// r4SkipChain is a Steps template of n sequential steps switched off by a
// when clause, followed by one real step.
func r4SkipChain(name string, n int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: argoproj.io/v1alpha1\nkind: Workflow\nmetadata:\n  name: %s\n  namespace: default\nspec:\n  entrypoint: main\n  arguments:\n    parameters:\n    - name: run\n      value: \"false\"\n  templates:\n  - name: main\n    steps:\n", name)
	for i := range n {
		fmt.Fprintf(&b, "    - - name: s%d\n        template: pod\n        when: \"{{workflow.parameters.run}} == true\"\n", i)
	}
	b.WriteString("    - - name: last\n        template: pod\n  - name: pod\n    container:\n      image: busybox\n")
	return b.String()
}

// TestRegressionR4_C67_SkipChainFirstReconcile ports
// TestProbe_v3x10_SkipChainPodInFirstReconcile (v3x10-1_test.go / C67) with
// n reduced to 100: base walks the whole when-false chain in one short
// reconcile and creates the last step's pod straight away. The fixed-point
// loop needs a pass per skipped step, each re-evaluating and rebuilding
// scopes for every step before it, and runs out of operation time first.
func TestRegressionR4_C67_SkipChainFirstReconcile(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4SkipChain("r4-c67-skip", 100))
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf, func(c *WorkflowController) {
		c.maxOperationTime = r4C67OperationTime
		// Drop Kubernetes events: the fake recorder's 64-event buffer blocks
		// once more than 64 steps have been skipped.
		c.eventRecorderManager = &testEventRecorderManager{eventRecorder: &record.FakeRecorder{}}
	})
	defer cancel()

	start := time.Now()
	woc := r4Operate(t, ctx, controller, wf)
	t.Logf("first reconcile took %v", time.Since(start))
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1, "the last step's pod is created in the first reconcile")
	_, err = woc.wf.GetNodeByName("r4-c67-skip[100].last")
	assert.NoError(t, err)
}

// r4C20Limit bounds TestRegressionR4_C20_NestedReconcileLinearInDepth's
// reconcile: base takes about 2ms, the fixed-point loop about 50ms.
const r4C20Limit = 20 * time.Millisecond

// r4C67OperationTime is TestRegressionR4_C67_SkipChainFirstReconcile's
// operation deadline: base walks the chain in about 0.1s, the fixed-point
// loop needs about 9s.
const r4C67OperationTime = 5 * time.Second

// r4MainCommands maps each pod's node name to its main container's command
// line (command and args joined by spaces, without the emissary prefix).
func r4MainCommands(ctx context.Context, t *testing.T, woc *wfOperationCtx) map[string]string {
	t.Helper()
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	out := map[string]string{}
	for i := range pods.Items {
		p := &pods.Items[i]
		for _, c := range p.Spec.Containers {
			if c.Name != common.MainContainerName {
				continue
			}
			full := strings.Join(append(append([]string{}, c.Command...), c.Args...), " ")
			if _, after, ok := strings.Cut(full, " -- "); ok {
				full = after
			}
			out[p.Annotations[common.AnnotationKeyNodeName]] = full
		}
	}
	return out
}

// r4ErrorNodes lists the nodes in phase Error, as "name: message".
func r4ErrorNodes(woc *wfOperationCtx) []string {
	var out []string
	for _, n := range woc.wf.Status.Nodes {
		if n.Phase == wfv1.NodeError {
			out = append(out, n.Name+": "+n.Message)
		}
	}
	sort.Strings(out)
	return out
}

// r4SucceedPodsWith marks every unfinished pod Succeeded, giving the pod of
// a node whose display name is in outs those outputs.
func r4SucceedPodsWith(ctx context.Context, woc *wfOperationCtx, outs map[string]*wfv1.Outputs) {
	pods, err := listPods(ctx, woc)
	if err != nil {
		panic(err)
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == apiv1.PodSucceeded || pod.Status.Phase == apiv1.PodFailed {
			continue
		}
		nodeID := woc.nodeID(&pod)
		var w []with
		if o := outs[woc.wf.Status.Nodes[nodeID].DisplayName]; o != nil {
			w = append(w, withOutputs(ctx, *o))
		}
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.ID == nodeID {
				return apiv1.PodSucceeded
			}
			return ""
		}, w...)
	}
}

// r4GoTemplateScript is a leaf script template whose source holds kubectl
// go-template text, which is not Argo's to substitute.
const r4GoTemplateScript = `
  - name: list-pods
    script:
      image: bitnami/kubectl
      command: [sh]
      source: |
        kubectl get pods -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'
`

// TestRegressionR4_C16_LeafEntrypointGoTemplate ports
// TestProbe_v1x48_GoTemplateLeafEntrypoint (v1x48-1_test.go / C16). Main's
// SubstituteParams always lets unresolved tags through, so a leaf
// entrypoint whose script holds kubectl go-template text runs its pod.
// HEAD's reconcileTemplate substitutes a plain leaf template strictly and
// ends the workflow Error "failed to resolve {{range .items}}".
func TestRegressionR4_C16_LeafEntrypointGoTemplate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c16-gotmpl-entry
  namespace: default
spec:
  entrypoint: list-pods
  templates:` + r4GoTemplateScript)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, woc.wf.Status.Message)
	woc = r4DriveToEnd(t, ctx, controller, woc, 3)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
}

// TestRegressionR4_C16_ScheduledTimeLeafEntrypoint ports
// TestProbe_v1x48_ScheduledTimeLeafEntrypoint (v1x48-1_test.go / C16):
// {{workflow.scheduledTime}} is only set for CronWorkflow runs, and main
// passes it through unresolved otherwise.
func TestRegressionR4_C16_ScheduledTimeLeafEntrypoint(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c16-sched-entry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, "scheduled at {{workflow.scheduledTime}}"]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, woc.wf.Status.Message)
}

// TestRegressionR4_C16_DAGRunningHookGoTemplate ports
// TestProbe_v1x48_GoTemplateDAGRunningHook (v1x48-1_test.go / C16): a DAG
// task's running hook whose template holds go-template text runs its pod,
// instead of the hook node and task going Error.
func TestRegressionR4_C16_DAGRunningHookGoTemplate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c16-dag-hook
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: work
        hooks:
          running:
            expression: tasks.a.status == "Running"
            template: list-pods
  - name: work
    container:
      image: busybox
      command: [echo, hi]` + r4GoTemplateScript)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = r4Operate(t, ctx, controller, woc.wf)
	dumpNodes(t, "hook", woc.wf)
	_, ok := r4MainCommands(ctx, t, woc)["r4-c16-dag-hook.a.hooks.running"]
	assert.True(t, ok, "running hook pod not created")
	assert.Empty(t, r4ErrorNodes(woc))
	woc = r4DriveToEnd(t, ctx, controller, woc, 4)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
}

// TestRegressionR4_C16_DAGHookUnresolvedArgOnOmitted ports
// TestProbe_lead4_DAGTrueHookUnresolvedArgOnOmitted (lead4-1_test.go, lead
// 4 / C16). b depends on a, a fails and b is Omitted; b's running hook
// (expression "true") passes {{tasks.a.outputs.result}}, which never
// resolves. Main let the unresolved Argo tag through and the workflow ended
// Failed; HEAD's strict substitution of the hook makes it Error with
// "failed to resolve".
func TestRegressionR4_C16_DAGHookUnresolvedArgOnOmitted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c16-lead4-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: gen
      - name: b
        depends: a
        template: run
        hooks:
          running:
            expression: "true"
            template: notify
            arguments:
              parameters:
              - name: message
                value: "{{tasks.a.outputs.result}}"
  - name: gen
    script:
      image: busybox
      command: [sh]
      source: echo hi
  - name: run
    container:
      image: busybox
      command: [echo]
  - name: notify
    inputs:
      parameters:
      - name: message
    container:
      image: busybox
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{Submit: true}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	decide := func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.HasSuffix(n.Name, ".a") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	}
	woc := r4Operate(t, ctx, controller, wf)
	for i := 0; i < 12 && !woc.wf.Status.Phase.Completed(); i++ {
		setPodPhases(ctx, woc, decide)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, woc.wf.Status.Message)
	assert.NotContains(t, woc.wf.Status.Message, "failed to resolve")
}

// TestRegressionR4_C14_DAGExitHookInputShadow ports
// TestProbe_v1x47_DAGExitHookInputShadow (v1x47-1_test.go / C14). A task's
// exit hook template has an input named like an input of the enclosing DAG
// template. Main substitutes the hook with its own arguments only, so the
// hook pod runs "echo hook-value"; HEAD copies the task's scope into the
// hook's local parameters, which win, and the pod runs "echo parent-value".
func TestRegressionR4_C14_DAGExitHookInputShadow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c14-dag
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: message
      value: parent-value
  templates:
  - name: main
    inputs:
      parameters:
      - name: message
    dag:
      tasks:
      - name: a
        template: work
        hooks:
          exit:
            template: notify
            arguments:
              parameters:
              - name: message
                value: hook-value
  - name: work
    container:
      image: busybox
      command: [echo, hi]
  - name: notify
    inputs:
      parameters:
      - name: message
    container:
      image: busybox
      command: [echo, "{{inputs.parameters.message}}"]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	woc = r4DriveToEnd(t, ctx, controller, woc, 3)
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, "echo hook-value", r4MainCommands(ctx, t, woc)["r4-c14-dag.a.onExit"])
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
}

// TestRegressionR4_C14_ExitHookInnerStepsShadow ports
// TestProbe_v1x47_ExitHookInnerStepsShadow (v1x47-1_test.go / C14). The
// exit hook of step a is a Steps template with its own step named gen,
// like an outer step. Its inner {{steps.gen.outputs.parameters.p}} must
// read the inner gen ("inner"), not the outer one ("outer") that HEAD
// copies into the hook's local parameters.
func TestRegressionR4_C14_ExitHookInnerStepsShadow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c14-inner
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: gen
    - - name: a
        template: work
        hooks:
          exit:
            template: cleanup
  - name: cleanup
    steps:
    - - name: gen
        template: gen
    - - name: use
        template: echo
        arguments:
          parameters:
          - name: msg
            value: "{{steps.gen.outputs.parameters.p}}"
  - name: gen
    container:
      image: busybox
      command: [echo]
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
  - name: work
    container:
      image: busybox
      command: [echo, hi]
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: busybox
      command: [echo, "{{inputs.parameters.msg}}"]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	out := func(v string) map[string]*wfv1.Outputs {
		return map[string]*wfv1.Outputs{"gen": {Parameters: []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr(v)}}}}
	}
	woc := r4Operate(t, ctx, controller, wf)
	r4SucceedPodsWith(ctx, woc, out("outer"))
	woc = r4Operate(t, ctx, controller, woc.wf)
	// a succeeds, so its exit hook (the cleanup steps) starts its inner gen.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = r4Operate(t, ctx, controller, woc.wf)
	for range 3 {
		r4SucceedPodsWith(ctx, woc, out("inner"))
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, "echo inner", r4MainCommands(ctx, t, woc)["r4-c14-inner[1].a.onExit[1].use"])
}

// TestRegressionR4_C14_DAGExitHookRetryInputShadow ports
// TestProbe_v1x47_DAGExitHookRetryInputShadow (v1x47-1_test.go / C14): the
// C14 shadow with a retried hook template; every hook attempt must run
// "echo hook-value".
func TestRegressionR4_C14_DAGExitHookRetryInputShadow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c14-retry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    inputs:
      parameters:
      - name: message
        value: parent-value
    dag:
      tasks:
      - name: a
        template: work
        hooks:
          exit:
            template: notify
            arguments:
              parameters:
              - name: message
                value: hook-value
  - name: work
    container:
      image: busybox
      command: [echo, hi]
  - name: notify
    retryStrategy:
      limit: 2
    inputs:
      parameters:
      - name: message
    container:
      image: busybox
      command: [echo, "{{inputs.parameters.message}}"]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	woc = r4DriveToEnd(t, ctx, controller, woc, 3)
	dumpNodes(t, "final", woc.wf)
	var got []string
	for node, cmd := range r4MainCommands(ctx, t, woc) {
		if strings.Contains(node, "onExit") {
			got = append(got, node+"="+cmd)
		}
	}
	sort.Strings(got)
	require.NotEmpty(t, got)
	for _, g := range got {
		assert.Contains(t, g, "=echo hook-value")
	}
}

const r4C88WFT = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hooklib
  namespace: default
spec:
  templates:
  - name: exit-steps
    steps:
    - - name: e
        template: ok
  - name: ok
    container:
      image: busybox
  - name: flaky
    retryStrategy:
      limit: "1"
    container:
      image: busybox
`

const r4C88Wf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c88
  namespace: default
spec:
  entrypoint: main
  hooks:
    exit:
      templateRef:
        name: hooklib
        template: exit-steps
  templates:
  - name: main
    steps:
    - - name: s
        template: inner
    - - name: last
        template: ok
        hooks:
          exit:
            templateRef:
              name: hooklib
              template: flaky
  - name: inner
    dag:
      tasks:
      - name: a
        template: ok
        hooks:
          exit:
            templateRef:
              name: hooklib
              template: exit-steps
  - name: ok
    container:
      image: busybox
`

// TestRegressionR4_C88_HookTemplateRefScope ports
// TestProbe_v1x66_HookTemplateRefScope (v1x66-1_test.go / C88). A hook or
// onExit node that uses templateRef records its caller's templateScope, as
// the task it hooks does (main: the tmplCtx the hook is called from). HEAD
// records the referenced WorkflowTemplate's scope, "namespaced/hooklib".
func TestRegressionR4_C88_HookTemplateRefScope(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C88Wf)
	wft := wfv1.MustUnmarshalWorkflowTemplate(r4C88WFT)
	cancel, controller := newController(ctx, wf, wft, func(c *WorkflowController) {
		c.eventRecorderManager = &testEventRecorderManager{eventRecorder: record.NewFakeRecorder(1000000)}
	})
	defer cancel()
	require.NoError(t, validate.Workflow(ctx,
		templateresolution.WrapWorkflowTemplateInterface(controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(wf.Namespace)),
		templateresolution.WrapClusterWorkflowTemplateInterface(controller.wfclientset.ArgoprojV1alpha1().ClusterWorkflowTemplates()),
		wf.DeepCopy(), nil, validate.Opts{}))

	decide := func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.HasSuffix(n.Name, ".last.onExit(0)") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	}
	woc := r4Operate(t, ctx, controller, wf)
	for i := 0; i < 16 && !woc.wf.Status.Phase.Completed(); i++ {
		setPodPhases(ctx, woc, decide)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	dumpNodes(t, "final", woc.wf)
	require.True(t, woc.wf.Status.Phase.Completed())
	scope := func(name string) string {
		t.Helper()
		n, err := woc.wf.GetNodeByName(name)
		require.NoError(t, err)
		return n.TemplateScope
	}
	assert.Equal(t, scope("r4-c88[0].s.a"), scope("r4-c88[0].s.a.onExit"), "task exit hook scope")
	assert.Equal(t, scope("r4-c88"), scope("r4-c88.onExit"), "workflow exit hook scope")
	assert.Equal(t, scope("r4-c88"), scope("r4-c88[1].last.onExit"), "step exit hook (retry) scope")
}

// r4ArmDeadlineOnPodCreate makes the operate that creates a pod run past its
// per-operate deadline (as happens when a large fan-out is throttled by the
// client QPS limit), deterministically: the first pod create after arming
// moves the current woc's deadline into the past. Ported from r1x56-1_test.go
// armDeadlineOnPodCreate (C69), used only by
// TestRegressionR4_C69_WfLevelRunningHookFulfilledThenDeadline.
func r4ArmDeadlineOnPodCreate(controller *WorkflowController, cur **wfOperationCtx, armed *bool) {
	controller.kubeclientset.(*fake.Clientset).PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if *armed && *cur != nil {
			(*cur).deadline = time.Now().Add(-time.Minute)
		}
		return false, nil, nil
	})
}

// TestRegressionR4_C69_DeadlineBeforeEntryFulfilledRoot ports
// TestProbe_r1x56_DeadlineBeforeEntryFulfilledRoot (r1x56-1_test.go / C69).
// reconcileTemplate's early deadline gate returned ErrDeadlineExceeded before
// template resolution even for a node that is already fulfilled. Main
// checked the deadline only in checkConstraints, reached after
// handleNodeFulfilled, so a fulfilled entry node still completed the
// workflow in the same operate that ran past its deadline. HEAD leaves the
// workflow Running for one extra operate.
func TestRegressionR4_C69_DeadlineBeforeEntryFulfilledRoot(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c69-deadline
  namespace: default
spec:
  entrypoint: work
  templates:
  - name: work
    container:
      image: alpine
      command: [echo, hi]
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.deadline = time.Now().Add(-time.Minute)
	woc.operate(ctx)
	firstPhase := woc.wf.Status.Phase
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "converges")
	assert.Equal(t, wfv1.WorkflowSucceeded, firstPhase, "same operate as the fulfilled entry node")
}

// TestRegressionR4_C69_WfLevelRunningHookFulfilledThenDeadline ports
// TestProbe_r1x56_WfLevelRunningHookFulfilledThenDeadline (r1x56-1_test.go /
// C69). A workflow-level "running" lifecycle hook (documented in
// examples/life-cycle-hooks-wf-level.yaml) re-enters its already-fulfilled
// hook node through reconcileTemplate on every operate. When a later operate
// runs past MAX_OPERATION_TIME while creating fan-out pods, the early
// deadline gate turned that re-entry into ErrDeadlineExceeded, which
// bubbled up and marked the root node Error "Deadline exceeded" -- ending
// the workflow Error with the fan-out unfinished, instead of just leaving it
// Running for one more operate.
func TestRegressionR4_C69_WfLevelRunningHookFulfilledThenDeadline(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c69-hook-deadline
  namespace: default
spec:
  entrypoint: main
  hooks:
    running:
      expression: workflow.status == "Running"
      template: notify
  templates:
  - name: main
    steps:
    - - name: first
        template: work
    - - name: fanout
        template: work
        withItems: [1, 2, 3, 4]
  - name: work
    container:
      image: alpine
      command: [echo, hi]
  - name: notify
    container:
      image: alpine
      command: [echo, running]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	var cur *wfOperationCtx
	armed := false
	r4ArmDeadlineOnPodCreate(controller, &cur, &armed)

	// operate 1: first step pod + running hook pod
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	cur = woc
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// operate 2: hook + first fulfilled; fan-out starts and this operate runs past its deadline
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	cur = woc
	armed = true
	woc.operate(ctx)
	armed = false
	root := woc.wf.Status.Nodes.FindByDisplayName("r4-c69-hook-deadline")
	require.NotNil(t, root)
	assert.Equal(t, wfv1.NodeRunning, root.Phase, "root node after deadline-exceeded operate")

	// operate 3: normal operate, the workflow must still be running
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	cur = woc
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// run to completion
	for i := 0; i < 4 && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		cur = woc
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}
