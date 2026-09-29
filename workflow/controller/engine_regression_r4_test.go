package controller

// Round 4 regression red tests, per pr-16290-round4-fix-plan/task-*.md.
// Shared helpers live in engine_regression_r4_helpers_test.go. These files
// must compile against wt-base (4389bbf96) too, so `basecheck` can overlay
// them there: use only functions/types present in both trees.

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
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

// r4RunRounds unmarshals manifest, validates it, operates once, then
// succeeds every pod and operates again for each of the remaining rounds.
// Shared by the C80/C84 message tests.
func r4RunRounds(t *testing.T, manifest string, rounds int) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 1; i < rounds && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	return woc
}

// TestRegressionR4_C80_EntryPodSpecPatchMessage ports
// TestProbe_v1x2_EntryPodSpecPatchMessage (v1x2-1_test.go / C80). An
// entrypoint whose podSpecPatch fails to apply must report the cause alone,
// not wrapped as if the workflow itself were a task ("task <wf> errored:").
func TestRegressionR4_C80_EntryPodSpecPatchMessage(t *testing.T) {
	woc := r4RunRounds(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c80-entry-psp
  namespace: default
spec:
  entrypoint: work
  arguments:
    parameters:
    - name: patch
      value: not-a-pod-spec
  templates:
  - name: work
    podSpecPatch: "{{workflow.parameters.patch}}"
    container:
      image: alpine
      command: [echo, hi]
`, 2)
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "error in entry template execution: Error applying PodSpecPatch", woc.wf.Status.Message)
}

// TestRegressionR4_C80_WfOnExitPodSpecPatchMessage ports
// TestProbe_v1x2_WfOnExitPodSpecPatchMessage (v1x2-1_test.go / C80). Same
// for a workflow-level onExit template.
func TestRegressionR4_C80_WfOnExitPodSpecPatchMessage(t *testing.T) {
	woc := r4RunRounds(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c80-onexit-psp
  namespace: default
spec:
  entrypoint: ok
  onExit: bad
  templates:
  - name: ok
    container:
      image: alpine
      command: [echo, hi]
  - name: bad
    podSpecPatch: "this is not a patch"
    container:
      image: alpine
      command: [echo, hi]
`, 3)
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "error in exit template execution : Error applying PodSpecPatch", woc.wf.Status.Message)
}

// TestRegressionR4_C80_StepsOnExitPodSpecPatchMessage ports
// TestProbe_v1x2_StepsOnExitPodSpecPatchMessage (v1x2-1_test.go / C80). A
// step-level exit hook whose pod cannot be built must not gain the "task
// ... errored:" prefix on the workflow message either.
func TestRegressionR4_C80_StepsOnExitPodSpecPatchMessage(t *testing.T) {
	woc := r4RunRounds(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c80-steps-onexit-psp
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: ok
        onExit: bad
  - name: ok
    container:
      image: alpine
      command: [echo, hi]
  - name: bad
    podSpecPatch: "this is not a patch"
    container:
      image: alpine
      command: [echo, hi]
`, 4)
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "Error applying PodSpecPatch", woc.wf.Status.Message)
}

// TestRegressionR4_C84_PodRetryExprErrorMessage ports
// TestProbe_v3x2_PodRetryExprErrorMessage (v3x2-1_test.go / C84). A pod
// entry template whose retryStrategy.expression fails to evaluate must not
// gain the "task <wf> errored:" prefix on the workflow message.
func TestRegressionR4_C84_PodRetryExprErrorMessage(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c84-pod-retry
  namespace: default
spec:
  entrypoint: work
  templates:
  - name: work
    retryStrategy:
      limit: 2
      expression: 'asInt(lastRetry.message) >= 0'
    container:
      image: alpine
      command: [echo]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 10 && !woc.wf.Status.Phase.Completed(); i++ {
		for id, n := range woc.wf.Status.Nodes {
			if (n.Type == wfv1.NodeTypeSteps || n.Type == wfv1.NodeTypeDAG) && !n.StartedAt.IsZero() {
				n.StartedAt = metav1.NewTime(n.StartedAt.Add(time.Duration(-1500) * time.Millisecond))
				woc.wf.Status.Nodes[id] = n
			}
		}
		makePodsPhase(ctx, woc, apiv1.PodFailed)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "final", woc.wf)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.NotContains(t, woc.wf.Status.Message, "task r4-c84-pod-retry errored")
}

// r4WhenBadWfWithGen builds a workflow whose entrypoint has a "gen" step/task
// producing "heads" and a dependant with an invalid `when` clause containing
// a stray `@`, in DAG or Steps shape. Shared by the C83 tests.
func r4WhenBadWfWithGen(name string, dag bool) string {
	body := `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: tails
        depends: gen
        template: echo
        when: "{{tasks.gen.outputs.result}} == @tails"`
	if !dag {
		body = `
    steps:
    - - name: gen
        template: gen
    - - name: tails
        template: echo
        when: "{{steps.gen.outputs.result}} == @tails"`
	}
	return `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ` + name + `
spec:
  entrypoint: main
  templates:
  - name: main` + body + `
  - name: gen
    script:
      image: python:alpine3.6
      command: [python]
      source: print("heads")
  - name: echo
    container:
      image: alpine:3.7
      command: [echo, hi]
`
}

// TestRegressionR4_C83_DAGInvalidWhenHint ports
// TestProbe_v1x69_DAGInvalidWhenHint (v1x69-1_test.go / C83). An invalid
// `when` expression's error message lost the closing quote-hint text when
// shouldExecute moved into engine.go: base's hint ends
// `(hint: try wrapping the affected expression in quotes ("))`, HEAD's ends
// `(hint: try wrapping the affected expression in quotes)`.
func TestRegressionR4_C83_DAGInvalidWhenHint(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4WhenBadWfWithGen("r4-c83-dag-badwhen", true))
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 6 && !woc.wf.Status.Phase.Completed(); i++ {
		out := withOutputs(ctx, wfv1.Outputs{Result: new("heads")})
		onlyGen := func(pod *apiv1.Pod, woc *wfOperationCtx) {
			if n, ok := woc.wf.Status.Nodes[woc.nodeID(pod)]; ok && n.TemplateName == "gen" {
				out(pod, woc)
			}
		}
		makePodsPhase(ctx, woc, apiv1.PodSucceeded, onlyGen)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "final", woc.wf)
	n := woc.wf.Status.Nodes.FindByDisplayName("tails")
	require.NotNil(t, n)
	assert.Equal(t, wfv1.NodeError, n.Phase)
	assert.Equal(t, `Invalid 'when' expression 'heads == @tails': Invalid token: '@' (hint: try wrapping the affected expression in quotes ("))`, n.Message)
}

// TestRegressionR4_C83_StepsInvalidWhenHint ports
// TestProbe_v1x69_StepsInvalidWhenHint (v1x69-1_test.go / C83). Same hint,
// Steps shape; the node carrying the message differs by tree, so this only
// requires the full hint to appear somewhere.
func TestRegressionR4_C83_StepsInvalidWhenHint(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4WhenBadWfWithGen("r4-c83-steps-badwhen", false))
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 6 && !woc.wf.Status.Phase.Completed(); i++ {
		out := withOutputs(ctx, wfv1.Outputs{Result: new("heads")})
		onlyGen := func(pod *apiv1.Pod, woc *wfOperationCtx) {
			if n, ok := woc.wf.Status.Nodes[woc.nodeID(pod)]; ok && n.TemplateName == "gen" {
				out(pod, woc)
			}
		}
		makePodsPhase(ctx, woc, apiv1.PodSucceeded, onlyGen)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "final", woc.wf)
	all := woc.wf.Status.Message + "\n"
	for _, n := range woc.wf.Status.Nodes {
		all += n.Message + "\n"
	}
	assert.Contains(t, all, `(hint: try wrapping the affected expression in quotes ("))`)
}

// TestRegressionR4_C86_SeqNoCountNoEnd ports TestProbe_v1x40_DagSeqNoCountNoEnd
// (v1x40-1_test.go / C86). A withSequence with neither count nor end
// (validation accepts it) must error, as base's expandSequence did with
// "neither end nor count was specified in withSequence", instead of HEAD's
// silent zero-item "Skipped, empty params".
func TestRegressionR4_C86_SeqNoCountNoEnd(t *testing.T) {
	for _, rounds := range []int{1, 3} {
		t.Run(fmt.Sprintf("rounds=%d", rounds), func(t *testing.T) {
			woc := r4RunRounds(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c86-seq
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: fan
        template: echo
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withSequence: {start: "5"}
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: busybox
      args: ["{{inputs.parameters.msg}}"]
`, rounds)
			dumpNodes(t, "final", woc.wf)
			fan, err := woc.wf.GetNodeByName("r4-c86-seq.fan")
			require.NoError(t, err)
			assert.Equal(t, wfv1.NodeError, fan.Phase, fan.Message)
			assert.Contains(t, fan.Message, "neither end nor count")
			assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		})
	}
}

// r4ValidateWithTemplates validates wf against the controller's workflow
// templates, so a dynamic templateRef validates as it would on submit.
func r4ValidateWithTemplates(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) {
	t.Helper()
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(wf.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(controller.wfclientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	require.NoError(t, validate.Workflow(ctx, wftmplGetter, cwftmplGetter, wf.DeepCopy(), nil, validate.Opts{}))
}

// r4RunGen validates manifest, operates once, succeeds the pods of template
// "gen" with genOut (other pods are left alone), then operates rounds more
// times, reconciling from the in-memory status as the probes did.
func r4RunGen(t *testing.T, manifest string, genOut wfv1.Outputs, rounds int, objs ...any) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	cancel, controller := newController(ctx, append([]any{wf}, objs...)...)
	t.Cleanup(cancel)
	r4ValidateWithTemplates(ctx, t, controller, wf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	out := withOutputs(ctx, genOut)
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.TemplateName == "gen" {
			return apiv1.PodSucceeded
		}
		return ""
	}, out)
	for range rounds {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "after gen", woc.wf)
	return woc
}

// r4CommandsMatching lists the main-container command lines of the pods
// whose node name contains sub, sorted.
func r4CommandsMatching(ctx context.Context, t *testing.T, woc *wfOperationCtx, sub string) []string {
	t.Helper()
	var got []string
	for name, cmd := range r4MainCommands(ctx, t, woc) {
		if strings.Contains(name, sub) {
			got = append(got, cmd)
		}
	}
	sort.Strings(got)
	return got
}

const r4C5ItemTagTemplates = `
  - name: gen
    script:
      image: alpine
      command: [sh]
      source: echo hi
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

// TestRegressionR4_C5_StepsWithParamGitHubActionsExpr ports
// TestProbe_v3x9_StepsWithParamGitHubActionsExpr (v3x9-1_test.go / C5, lead
// 3). A withParam list whose items carry GitHub Actions `${{ steps.x }}` text
// runs both item pods with the literal text, as base did. HEAD re-parses the
// substituted item arguments with steps/tasks strict, requeues the whole
// batch and never creates an item.
func TestRegressionR4_C5_StepsWithParamGitHubActionsExpr(t *testing.T) {
	woc := r4RunGen(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c5-gha
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: gen
    - - name: consume
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withParam: "{{steps.gen.outputs.result}}"
`+r4C5ItemTagTemplates, wfv1.Outputs{Result: new(`["plain", "ref=${{ steps.checkout.outputs.ref }}"]`)}, 5)
	ctx := logging.TestContext(t.Context())
	assert.Equal(t, []string{"sh -c echo plain", "sh -c echo ref=${{ steps.checkout.outputs.ref }}"}, r4CommandsMatching(ctx, t, woc, "consume"))
	assert.NotEqual(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestRegressionR4_C5_DAGItemTagNotSilentSuccess ports
// TestProbe_v3x9_DAGItemTagNotSilentSuccess (v3x9-1_test.go / C5, lead 3):
// a DAG withParam item with `${{ tasks.x }}` text must not leave a childless
// TaskGroup that ends the workflow Succeeded without any item running.
func TestRegressionR4_C5_DAGItemTagNotSilentSuccess(t *testing.T) {
	woc := r4RunGen(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c5-dag-item-tag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: gen
        template: gen
      - name: consume
        depends: gen
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withParam: "{{tasks.gen.outputs.result}}"
`+r4C5ItemTagTemplates, wfv1.Outputs{Result: new(`["plain", "ref=${{ tasks.x.outputs.result }}"]`)}, 5)
	ctx := logging.TestContext(t.Context())
	if woc.wf.Status.Phase == wfv1.WorkflowSucceeded {
		assert.NotEmpty(t, r4CommandsMatching(ctx, t, woc, "consume"), "workflow Succeeded but no consume item ran")
	}
}

// r4C7Drive runs the workflow to completion (or maxRounds), succeeding every
// pod. Pods of template "gen" report result and parameter out = genOut.
func r4C7Drive(t *testing.T, manifest, genOut string, maxRounds int) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range maxRounds {
		woc.operate(ctx)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		setPodPhases(ctx, woc, func(node *wfv1.NodeStatus) apiv1.PodPhase {
			if node.Fulfilled() {
				return ""
			}
			return apiv1.PodSucceeded
		}, func(pod *apiv1.Pod, woc *wfOperationCtx) {
			node := woc.wf.Status.Nodes[woc.nodeID(pod)]
			out, res := genOut, genOut
			if node.TemplateName != "gen" {
				out = node.Name
				res = fmt.Sprintf("%q", node.Name)
			}
			withOutputs(ctx, wfv1.Outputs{Result: &res, Parameters: []wfv1.Parameter{{Name: "out", Value: wfv1.AnyStringPtr(out)}}})(pod, woc)
		})
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
	dumpNodes(t, "final", woc.wf)
	return woc
}

func r4C7Workflow(name, body string) string {
	return `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ` + name + `
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
` + body + `
  - name: gen
    outputs:
      parameters:
      - name: out
        valueFrom:
          path: /tmp/out
    container:
      image: busybox
  - name: echo
    inputs:
      parameters:
      - name: msg
    outputs:
      parameters:
      - name: out
        valueFrom:
          path: /tmp/out
    container:
      image: busybox
      args: ["{{inputs.parameters.msg}}"]
`
}

// TestRegressionR4_C7_WhenGuardedExpansion ports
// TestProbe_v1x24_WhenGuardedExpansion (v1x24-1_test.go / C7). A when that
// reads the same output as the withParam/withSequence guards the expansion:
// base resolved and evaluated the when first and skipped the task without
// parsing the list. HEAD evaluates the unresolved when during expansion and
// errors the task on the unparseable list.
func TestRegressionR4_C7_WhenGuardedExpansion(t *testing.T) {
	for _, tc := range []struct{ name, body, genOut, fanNode, afterNode string }{
		{"dag-withparam-empty-quoted-when", `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: fan
        depends: gen
        template: echo
        when: "'{{tasks.gen.outputs.parameters.out}}' != ''"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withParam: "{{tasks.gen.outputs.parameters.out}}"
      - name: after
        depends: fan
        template: echo
        arguments: {parameters: [{name: msg, value: "after"}]}
`, "", "wa.fan", "wa.after"},
		{"dag-withparam-none-unquoted-when", `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: fan
        depends: gen
        template: echo
        when: "{{tasks.gen.outputs.result}} != none"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withParam: "{{tasks.gen.outputs.result}}"
      - name: after
        depends: fan
        template: echo
        arguments: {parameters: [{name: msg, value: "after"}]}
`, "none", "wa.fan", "wa.after"},
		{"steps-withparam-empty-quoted-when", `
    steps:
    - - name: gen
        template: gen
    - - name: fan
        template: echo
        when: "'{{steps.gen.outputs.parameters.out}}' != ''"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withParam: "{{steps.gen.outputs.parameters.out}}"
    - - name: after
        template: echo
        arguments: {parameters: [{name: msg, value: "after"}]}
`, "", "wa[1].fan", "wa[2].after"},
		{"steps-withparam-none-unquoted-when", `
    steps:
    - - name: gen
        template: gen
    - - name: fan
        template: echo
        when: "{{steps.gen.outputs.result}} != none"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withParam: "{{steps.gen.outputs.result}}"
    - - name: after
        template: echo
        arguments: {parameters: [{name: msg, value: "after"}]}
`, "none", "wa[1].fan", "wa[2].after"},
		{"dag-withsequence-none", `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: fan
        depends: gen
        template: echo
        when: "{{tasks.gen.outputs.result}} != none"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withSequence: {count: "{{tasks.gen.outputs.result}}"}
      - name: after
        depends: fan
        template: echo
        arguments: {parameters: [{name: msg, value: "after"}]}
`, "none", "wa.fan", "wa.after"},
		{"steps-withsequence-none", `
    steps:
    - - name: gen
        template: gen
    - - name: fan
        template: echo
        when: "{{steps.gen.outputs.result}} != none"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withSequence: {count: "{{steps.gen.outputs.result}}"}
    - - name: after
        template: echo
        arguments: {parameters: [{name: msg, value: "after"}]}
`, "none", "wa[1].fan", "wa[2].after"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			woc := r4C7Drive(t, r4C7Workflow("wa", tc.body), tc.genOut, 12)
			fan, err := woc.wf.GetNodeByName(tc.fanNode)
			require.NoError(t, err)
			assert.Equal(t, wfv1.NodeSkipped, fan.Phase, "fan: %s", fan.Message)
			after, err := woc.wf.GetNodeByName(tc.afterNode)
			require.NoError(t, err)
			assert.Equal(t, wfv1.NodeSucceeded, after.Phase, "after: %s", after.Message)
			assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
		})
	}
}

// TestRegressionR4_C7_WhenFalseValidListSkipped ports
// TestProbe_v1x24_WhenFalseValidListSkipped (v1x24-1_test.go / C7, P7). A
// when-false task with a dynamic list (withParam) is one Skipped node, as on
// main, which left withParam unresolved on the when-false early return; so
// fan.Skipped dependants run and fan.Succeeded ones are Omitted. HEAD made
// a Succeeded TaskGroup of Skipped items.
func TestRegressionR4_C7_WhenFalseValidListSkipped(t *testing.T) {
	woc := r4C7Drive(t, r4C7Workflow("wv", `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: fan
        depends: gen
        template: echo
        when: "{{tasks.gen.outputs.parameters.out}} == nope"
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withParam: "{{tasks.gen.outputs.result}}"
      - name: onskip
        depends: fan.Skipped
        template: echo
        arguments: {parameters: [{name: msg, value: "onskip"}]}
      - name: onsucc
        depends: fan.Succeeded
        template: echo
        arguments: {parameters: [{name: msg, value: "onsucc"}]}
`), `["a","b"]`, 12)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
	fan, err := woc.wf.GetNodeByName("wv.fan")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSkipped, fan.Phase, "fan type=%s msg=%s", fan.Type, fan.Message)
	onskip, err := woc.wf.GetNodeByName("wv.onskip")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, onskip.Phase)
	onsucc, err := woc.wf.GetNodeByName("wv.onsucc")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeOmitted, onsucc.Phase)
}

// r4C8Run runs the workflow: the first pod succeeds with result, every
// later pod succeeds with no outputs. It returns the final woc and the pods'
// node names.
func r4C8Run(t *testing.T, manifest, result string) (*wfOperationCtx, []string) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withOutputs(ctx, wfv1.Outputs{Result: &result}))
	for i := 0; i < 6 && !woc.wf.Status.Fulfilled(); i++ {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	}
	dumpNodes(t, "final", woc.wf)
	return woc, r4PodNodeNames(ctx, t, woc)
}

// TestRegressionR4_C8_StepsExprWhenDoubleQuotes ports
// TestProbe_r1x25_StepsExprWhenDoubleQuotes (r1x25-1_test.go / C8). An
// expression when with double quotes (the bracket form a dashed step name
// needs) is substituted in its JSON form, as on base; HEAD substitutes the
// raw string and fails to unmarshal the expression.
func TestRegressionR4_C8_StepsExprWhenDoubleQuotes(t *testing.T) {
	woc, pods := r4C8Run(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c8-expr
  namespace: default
spec:
  entrypoint: coinflip
  templates:
  - name: coinflip
    steps:
    - - name: flip-coin
        template: flip-coin
    - - name: heads
        template: say
        when: '{{= steps["flip-coin"].outputs.result == "heads" }}'
      - name: tails
        template: say
        when: '{{= steps["flip-coin"].outputs.result == "tails" }}'
  - name: flip-coin
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        print("heads")
  - name: say
    container:
      image: alpine:3.23
      command: [sh, -c, "echo hi"]
`, "heads")
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Contains(t, pods, "r4-c8-expr[1].heads")
	tails := woc.wf.Status.Nodes.FindByDisplayName("tails")
	require.NotNil(t, tails)
	assert.Equal(t, wfv1.NodeSkipped, tails.Phase)
}

// TestRegressionR4_C8_DAGSimpleTagBackslashResult ports
// TestProbe_r1x25_DAGSimpleTagBackslashResult (r1x25-1_test.go / C8). A
// result with a backslash substituted into a simple when compares equal, as
// on base; HEAD leaves the JSON escape in the expression, compares unequal
// and silently skips the task.
func TestRegressionR4_C8_DAGSimpleTagBackslashResult(t *testing.T) {
	woc, pods := r4C8Run(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c8-bs
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: gen
        template: gen
      - name: work
        depends: gen
        template: say
        when: "'{{tasks.gen.outputs.result}}' == 'C:\\temp'"
  - name: gen
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        print(r"C:\temp")
  - name: say
    container:
      image: alpine:3.23
      command: [sh, -c, "echo hi"]
`, `C:\temp`)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Contains(t, pods, "r4-c8-bs.work")
}

// r4PodInputArtifact returns the named input artifact baked into the pod of
// the node nodeName.
func r4PodInputArtifact(ctx context.Context, t *testing.T, woc *wfOperationCtx, nodeName, artName string) *wfv1.Artifact {
	t.Helper()
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	for i := range pods.Items {
		p := &pods.Items[i]
		if p.Annotations[common.AnnotationKeyNodeName] != nodeName {
			continue
		}
		tmpl, err := getPodTemplate(p)
		require.NoError(t, err)
		return tmpl.Inputs.Artifacts.GetArtifactByName(artName)
	}
	require.Failf(t, "pod not created", "no pod for %s", nodeName)
	return nil
}

// TestRegressionR4_C13_RawAndHTTPArtDAG ports
// TestProbe_v1x26_RawAndHTTPArtDAG (v1x26-1_test.go / C13). An artifact
// argument's location fields (raw.data, http.url) that reference a task's
// output reach the pod substituted, as on base, which substituted the whole
// task; HEAD substitutes only the parameters.
func TestRegressionR4_C13_RawAndHTTPArtDAG(t *testing.T) {
	woc := r4RunGen(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c13-raw-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: gen
        template: gen
      - name: use
        depends: gen
        template: use
        arguments:
          artifacts:
          - name: data
            raw:
              data: "{{tasks.gen.outputs.result}}"
          - name: web
            http:
              url: "https://example.com/{{tasks.gen.outputs.result}}.txt"
  - name: gen
    script:
      image: alpine
      command: [sh]
      source: echo hello
  - name: use
    inputs:
      artifacts:
      - name: data
        path: /tmp/data
      - name: web
        path: /tmp/web
    container:
      image: alpine
      command: [cat, /tmp/data]
`, wfv1.Outputs{Result: new("hello")}, 3)
	ctx := logging.TestContext(t.Context())
	data := r4PodInputArtifact(ctx, t, woc, "r4-c13-raw-dag.use", "data")
	require.NotNil(t, data)
	require.NotNil(t, data.Raw)
	assert.Equal(t, "hello", data.Raw.Data)
	web := r4PodInputArtifact(ctx, t, woc, "r4-c13-raw-dag.use", "web")
	require.NotNil(t, web)
	require.NotNil(t, web.HTTP)
	assert.Equal(t, "https://example.com/hello.txt", web.HTTP.URL)
}

// TestRegressionR4_C13_S3KeyArtSteps ports TestProbe_v1x26_S3KeyArtSteps
// (v1x26-1_test.go / C13): an S3 key built from a step's output parameter.
func TestRegressionR4_C13_S3KeyArtSteps(t *testing.T) {
	woc := r4RunGen(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c13-s3-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: gen
    - - name: use
        template: use
        arguments:
          artifacts:
          - name: remote
            s3:
              key: "prefix/{{steps.gen.outputs.parameters.p}}.txt"
  - name: gen
    container:
      image: alpine
      command: [sh, -c, "echo hi > /tmp/p"]
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
  - name: use
    inputs:
      artifacts:
      - name: remote
        path: /tmp/remote
    container:
      image: alpine
      command: [cat, /tmp/remote]
`, wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("my-key")}}}, 3)
	ctx := logging.TestContext(t.Context())
	remote := r4PodInputArtifact(ctx, t, woc, "r4-c13-s3-steps[1].use", "remote")
	require.NotNil(t, remote)
	require.NotNil(t, remote.S3)
	assert.Equal(t, "prefix/my-key.txt", remote.S3.Key)
}

const r4C44WFT = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: r4-c44-wft
  namespace: default
spec:
  templates:
  - name: say
    container:
      image: busybox
      command: [echo, from-wft]
`

// TestRegressionR4_C44_DynTRefStepsParam ports
// TestProbe_v1x27_DynTRefStepsParam (v1x27-1_test.go / C44). A templateRef
// whose name comes from an earlier step's output is resolved after
// substitution, as on base; HEAD resolves the raw holder and fails
// "workflow template {{steps.pick.outputs.parameters.wft}} not found".
func TestRegressionR4_C44_DynTRefStepsParam(t *testing.T) {
	woc := r4RunGen(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c44-steps-param
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: pick
        template: gen
    - - name: run
        templateRef:
          name: "{{steps.pick.outputs.parameters.wft}}"
          template: say
  - name: gen
    container:
      image: busybox
      command: [echo]
    outputs:
      parameters:
      - name: wft
        valueFrom:
          path: /tmp/wft
`, wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "wft", Value: wfv1.AnyStringPtr("r4-c44-wft")}}}, 1,
		wfv1.MustUnmarshalWorkflowTemplate(r4C44WFT))
	ctx := logging.TestContext(t.Context())
	assert.Equal(t, "echo from-wft", r4MainCommands(ctx, t, woc)["r4-c44-steps-param[1].run"])
	for i := 0; i < 4 && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, woc.controller)
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
}

const r4C45Templates = `
  - name: gen
    script:
      image: busybox
      command: [sh]
      source: echo hello
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: busybox
      command: [echo, "{{inputs.parameters.msg}}"]
`

// r4C45Commands runs a withItems workflow whose gen step reports "hello"
// and returns the command line of every pod but gen's, by node name.
func r4C45Commands(t *testing.T, name, body string) (*wfOperationCtx, map[string]string) {
	t.Helper()
	woc := r4RunGen(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: `+name+`
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
`+body+r4C45Templates, wfv1.Outputs{Result: new("hello")}, 1)
	ctx := logging.TestContext(t.Context())
	cmds := r4MainCommands(ctx, t, woc)
	for n := range cmds {
		if strings.HasSuffix(n, ".gen") {
			delete(cmds, n)
		}
	}
	return woc, cmds
}

// TestRegressionR4_C45_PlainItemNames ports TestProbe_v1x38_PlainItemNames
// (v1x38-1_test.go / C45). withItems values that reference an earlier
// output are resolved before expansion, as base substituted the whole task:
// the item's node name and its pod carry "hello". HEAD expands the raw tag
// text.
func TestRegressionR4_C45_PlainItemNames(t *testing.T) {
	for _, tc := range []struct{ kind, body, prefix string }{
		{"dag", `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: fan
        depends: gen
        template: echo
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withItems: ["{{tasks.gen.outputs.result}}", "static"]
`, "p.fan"},
		{"steps", `
    steps:
    - - name: gen
        template: gen
    - - name: fan
        template: echo
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withItems: ["{{steps.gen.outputs.result}}", "static"]
`, "p[1].fan"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			woc, cmds := r4C45Commands(t, "p", tc.body)
			assert.Equal(t, map[string]string{
				tc.prefix + "(0:hello)":  "echo hello",
				tc.prefix + "(1:static)": "echo static",
			}, cmds)
			assert.NotNil(t, woc.wf.Status.Nodes.FindByDisplayName("fan(0:hello)"), "display name fan(0:hello) missing")
		})
	}
}

// TestRegressionR4_C45_ExprItem ports TestProbe_v1x38_ExprItem
// (v1x38-1_test.go / C45): an expression when and argument over an item
// that references an earlier output see the resolved value.
func TestRegressionR4_C45_ExprItem(t *testing.T) {
	for _, tc := range []struct{ kind, body, prefix string }{
		{"dag", `
    dag:
      tasks:
      - name: gen
        template: gen
      - name: fan
        depends: gen
        template: echo
        when: "{{=item == 'hello'}}"
        arguments: {parameters: [{name: msg, value: "{{=sprig.upper(item)}}"}]}
        withItems: ["{{tasks.gen.outputs.result}}"]
`, "e.fan"},
		{"steps", `
    steps:
    - - name: gen
        template: gen
    - - name: fan
        template: echo
        when: "{{=item == 'hello'}}"
        arguments: {parameters: [{name: msg, value: "{{=sprig.upper(item)}}"}]}
        withItems: ["{{steps.gen.outputs.result}}"]
`, "e[1].fan"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			_, cmds := r4C45Commands(t, "e", tc.body)
			assert.Equal(t, map[string]string{tc.prefix + "(0:hello)": "echo HELLO"}, cmds)
		})
	}
}

// r4C82Run drives a flip-coin workflow whose gen pod reports "heads" to
// completion and returns the final workflow.
func r4C82Run(t *testing.T, manifest string) *wfv1.Workflow {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 8 && !woc.wf.Status.Phase.Completed(); i++ {
		out := withOutputs(ctx, wfv1.Outputs{Result: new("heads")})
		onlyGen := func(pod *apiv1.Pod, woc *wfOperationCtx) {
			if n, ok := woc.wf.Status.Nodes[woc.nodeID(pod)]; ok && n.TemplateName == "gen" {
				out(pod, woc)
			}
		}
		makePodsPhase(ctx, woc, apiv1.PodSucceeded, onlyGen)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "final", woc.wf)
	return woc.wf
}

const r4C82Templates = `
  - name: gen
    script:
      image: python:alpine3.6
      command: [python]
      source: print("heads")
  - name: echo
    container:
      image: alpine:3.7
      command: [echo, hi]
`

// TestRegressionR4_C82_StepsWhenSkipMessage ports
// TestProbe_v1x55_StepsWhenSkipMessage (v1x55-1_test.go / C82). A step
// skipped by its when reports the substituted expression, as on base, not
// the raw template text.
func TestRegressionR4_C82_StepsWhenSkipMessage(t *testing.T) {
	wf := r4C82Run(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c82-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: flip-coin
        template: gen
    - - name: heads
        template: echo
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails
        template: echo
        when: "{{steps.flip-coin.outputs.result}} == tails"
`+r4C82Templates)
	assert.Equal(t, wfv1.WorkflowSucceeded, wf.Status.Phase)
	h := wf.Status.Nodes.FindByDisplayName("heads")
	require.NotNil(t, h)
	assert.Equal(t, wfv1.NodeSucceeded, h.Phase)
	n := wf.Status.Nodes.FindByDisplayName("tails")
	require.NotNil(t, n)
	assert.Equal(t, wfv1.NodeSkipped, n.Phase)
	assert.Equal(t, "when 'heads == tails' evaluated false", n.Message)
}

// TestRegressionR4_C82_DAGWhenSkipMessage ports
// TestProbe_v1x55_DAGWhenSkipMessage (v1x55-1_test.go / C82): the DAG shape.
func TestRegressionR4_C82_DAGWhenSkipMessage(t *testing.T) {
	wf := r4C82Run(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c82-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: flip-coin
        template: gen
      - name: heads
        depends: flip-coin
        template: echo
        when: "{{tasks.flip-coin.outputs.result}} == heads"
      - name: tails
        depends: flip-coin
        template: echo
        when: "{{tasks.flip-coin.outputs.result}} == tails"
`+r4C82Templates)
	assert.Equal(t, wfv1.WorkflowSucceeded, wf.Status.Phase)
	n := wf.Status.Nodes.FindByDisplayName("tails")
	require.NotNil(t, n)
	assert.Equal(t, wfv1.NodeSkipped, n.Phase)
	assert.Equal(t, "when 'heads == tails' evaluated false", n.Message)
}

// r4C43RunUntilDone operates woc until the workflow completes or rounds is
// exhausted, succeeding every unfulfilled pod each round. The first round in
// which the node named producerDisplayName exists, it reports producerOut for
// that node via a WorkflowTaskResult, as the executor would once its pod
// succeeds.
func r4C43RunUntilDone(ctx context.Context, t *testing.T, controller *WorkflowController, woc *wfOperationCtx, producerDisplayName string, producerOut wfv1.Outputs, rounds int) *wfOperationCtx {
	t.Helper()
	reported := false
	for i := 0; i < rounds && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		if !reported {
			if n := woc.wf.Status.Nodes.FindByDisplayName(producerDisplayName); n != nil {
				r4TaskResultOutputs(ctx, woc, n.Name, producerOut)
				reported = true
			}
		}
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "final", woc.wf)
	return woc
}

// TestRegressionR4_C43_StepsOutputsExprWorkflowOutputs ports
// TestProbe_v2x4_StepsOutputsExprWorkflowOutputs (v2x4-1_test.go / C43). The
// "mid" Steps template's own output param is declared as
// valueFrom.expression: "workflow.outputs.parameters.g", read from the
// workflow-level global that "a" exported. Base seeded executeSteps' scope
// with wf.Status.Outputs, so this resolves; HEAD's setDAGOutputs never adds
// workflow outputs to its scope, so "mid" fails ("unknown name workflow"),
// "c" is Omitted and the workflow Fails.
func TestRegressionR4_C43_StepsOutputsExprWorkflowOutputs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c43-expr
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    steps:
    - - name: mid
        template: mid
    - - name: c
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{steps.mid.outputs.parameters.out}}"
  - name: mid
    steps:
    - - name: a
        template: produce
    outputs:
      parameters:
      - name: out
        valueFrom:
          expression: "workflow.outputs.parameters.g"
  - name: produce
    container:
      image: alpine
      command: [echo]
    outputs:
      parameters:
      - name: p
        globalName: g
        valueFrom:
          path: /tmp/p
  - name: consume
    inputs:
      parameters:
      - name: x
    container:
      image: alpine
      command: [sh, -c]
      args: ["echo {{inputs.parameters.x}}"]
`)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	r4ValidateWithTemplates(ctx, t, controller, wf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	woc = r4C43RunUntilDone(ctx, t, controller, woc, "a", wfv1.Outputs{
		Parameters: []wfv1.Parameter{{Name: "p", GlobalName: "g", Value: wfv1.AnyStringPtr("A")}},
	}, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
	c := woc.wf.Status.Nodes.FindByDisplayName("c")
	require.NotNil(t, c)
	require.NotNil(t, c.Inputs)
	require.Len(t, c.Inputs.Parameters, 1)
	require.NotNil(t, c.Inputs.Parameters[0].Value)
	assert.Equal(t, "A", c.Inputs.Parameters[0].Value.String())
}

// TestRegressionR4_C43_StepsOutputsArtFromWorkflowOutputs ports
// TestProbe_v2x4_StepsOutputsArtFromWorkflowOutputs (v2x4-1_test.go / C43).
// Artifacts have no global-params fallback, so
// outputs.artifacts[].from: "{{workflow.outputs.artifacts.ga}}" is the only
// way for a Steps template to hand out a global artifact as its own output;
// it needs the same workflow-outputs scope seed as the expression form.
func TestRegressionR4_C43_StepsOutputsArtFromWorkflowOutputs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c43-art
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    steps:
    - - name: mid
        template: mid
    - - name: c
        template: consume-art
        arguments:
          artifacts:
          - name: a
            from: "{{steps.mid.outputs.artifacts.out}}"
  - name: mid
    steps:
    - - name: a
        template: produce-art
    outputs:
      artifacts:
      - name: out
        from: "{{workflow.outputs.artifacts.ga}}"
  - name: produce-art
    container:
      image: alpine
      command: [echo]
    outputs:
      artifacts:
      - name: art
        globalName: ga
        path: /tmp/a
  - name: consume-art
    inputs:
      artifacts:
      - name: a
        path: /tmp/a
    container:
      image: alpine
      command: [sh, -c]
      args: ["cat /tmp/a"]
`)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	r4ValidateWithTemplates(ctx, t, controller, wf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	woc = r4C43RunUntilDone(ctx, t, controller, woc, "a", wfv1.Outputs{
		Artifacts: []wfv1.Artifact{{Name: "art", GlobalName: "ga", ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: "key-A"}}}},
	}, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
	c := woc.wf.Status.Nodes.FindByDisplayName("c")
	require.NotNil(t, c)
	require.NotNil(t, c.Inputs)
	require.Len(t, c.Inputs.Artifacts, 1)
	require.NotNil(t, c.Inputs.Artifacts[0].S3)
	assert.Equal(t, "key-A", c.Inputs.Artifacts[0].S3.Key)
}

const r4C30EmptyGroup = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c30-empty
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: first
        template: ok
    - []
    - - name: second
        template: ok
  - name: ok
    container:
      image: busybox
`

// TestRegressionR4_C30_EmptyStepGroupKeepsOrder ports
// TestProbe_v1x65_EmptyStepGroupKeepsOrderTop (v1x65-1_test.go / C30). An
// empty group (`- []`, the shape of test/e2e/smoke/empty-template-steps.yaml)
// must not reset the order: the group after it waits for the last group that
// had steps, so only first runs in round 0 and while it runs, and the Steps
// node's outbound node is second alone.
func TestRegressionR4_C30_EmptyStepGroupKeepsOrder(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C30EmptyGroup)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	assert.Equal(t, []string{"r4-c30-empty[0].first"}, r4PodNodeNames(ctx, t, woc), "round 0: only the first group's step may run")
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = r4Operate(t, ctx, controller, woc.wf)
	assert.Equal(t, []string{"r4-c30-empty[0].first"}, r4PodNodeNames(ctx, t, woc), "second must wait while first is running")

	woc = r4DriveToEnd(t, ctx, controller, woc, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	root, err := woc.wf.GetNodeByName("r4-c30-empty")
	require.NoError(t, err)
	second, err := woc.wf.GetNodeByName("r4-c30-empty[2].second")
	require.NoError(t, err)
	assert.Equal(t, []string{second.ID}, root.OutboundNodes)
}

// TestRegressionR4_C30_FailedFirstStepDoesNotRunLater ports
// TestProbe_r1x65_EmptyGroupFailedFirstStepDoesNotRunLater (r1x65-1_test.go
// / C30): when first fails, the step after the empty group never runs. It
// may have an Omitted node (the decided R4/R10 shape), but no pod.
func TestRegressionR4_C30_FailedFirstStepDoesNotRunLater(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C30EmptyGroup)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	for i := 0; i < 8 && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodFailed)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, []string{"r4-c30-empty[0].first"}, r4PodNodeNames(ctx, t, woc), "second must never run after first failed")
	if second, err := woc.wf.GetNodeByName("r4-c30-empty[2].second"); err == nil {
		assert.Equal(t, wfv1.NodeOmitted, second.Phase)
	}
}

// r4RetryStored applies `argo retry` (FormulateRetryWorkflow) to wf, stores
// the retried workflow and deletes the pods retry asks to delete, as the
// server does.
//
//nolint:revive // matches the r4 harness convention (t before ctx)
func r4RetryStored(t *testing.T, ctx context.Context, controller *WorkflowController, wf *wfv1.Workflow) *wfv1.Workflow {
	t.Helper()
	retried, podsToDelete, err := wfutil.FormulateRetryWorkflow(ctx, wf.DeepCopy(), false, "", nil)
	require.NoError(t, err, "argo retry")
	for _, p := range podsToDelete {
		_ = controller.kubeclientset.CoreV1().Pods(wf.Namespace).Delete(ctx, p, metav1.DeleteOptions{})
	}
	retried, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, retried, metav1.UpdateOptions{})
	require.NoError(t, err)
	return retried
}

// r4BackdateNodes moves the StartedAt of every node of the stored workflow d
// into the past, so a later node's start time is told apart from the Steps
// node's without sleeping between reconciles.
//
//nolint:revive // matches the r4 harness convention (t before ctx)
func r4BackdateNodes(t *testing.T, ctx context.Context, controller *WorkflowController, wf *wfv1.Workflow, d time.Duration) {
	t.Helper()
	wfs := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	stored, err := wfs.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	for id, n := range stored.Status.Nodes {
		n.StartedAt = metav1.NewTime(n.StartedAt.Add(-d))
		stored.Status.Nodes[id] = n
	}
	_, err = wfs.Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)
}

// r4UnfulfilledTyped lists the nodes that are not fulfilled, as
// "name(type)=phase".
func r4UnfulfilledTyped(wf *wfv1.Workflow) []string {
	var out []string
	for _, n := range wf.Status.Nodes {
		if !n.Fulfilled() {
			out = append(out, fmt.Sprintf("%s(%s)=%s", n.Name, n.Type, n.Phase))
		}
	}
	sort.Strings(out)
	return out
}

const r4C52FailFastGroups = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c52-ff
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    failFast: true
    steps:
    - - name: a
        template: work
    - - name: b
        template: work
    - - name: c
        template: work
  - name: work
    container:
      image: alpine
      command: [sh, -c, "exit 0"]
`

// TestRegressionR4_C52_TwoGroupsRetryCompletes ports
// TestProbe_r3x0_TwoGroupsRetryCompletes (r3x0-1_test.go / C52). A failFast
// Steps template whose first step fails ends with every node fulfilled, and
// `argo retry` then runs it to Succeeded. HEAD created every StepGroup up
// front; the failFast branch closed only one, so the later groups stayed
// Running and unlinked and retry failed with "couldn't find parent node".
func TestRegressionR4_C52_TwoGroupsRetryCompletes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C52FailFastGroups)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	for i := 0; i < 15 && !woc.wf.Status.Fulfilled(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodFailed)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Empty(t, r4UnfulfilledTyped(woc.wf), "nodes left non-terminal in a finished workflow")

	retried := r4RetryStored(t, ctx, controller, woc.wf)
	woc = r4Operate(t, ctx, controller, retried)
	for i := 0; i < 15 && !woc.wf.Status.Fulfilled(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Empty(t, r4UnfulfilledTyped(woc.wf), "nodes left non-terminal in a finished workflow")
}

// TestRegressionR4_C75_StepsDeadlineMessage ports
// TestProbe_v1x8_StepsDeadlineMessage (v1x8-1_test.go / C75).
// activeDeadlineSeconds expires while the first step of a two-group Steps
// template runs, and the killed step's pod lingers. The workflow message
// names the failed step, and group [1], which never ran, is not marked with
// the deadline message. HEAD pre-created [1] and linked it under the killed
// step, so execution control failed it "Step exceeded its deadline" and that
// replaced the Steps and workflow message.
func TestRegressionR4_C75_StepsDeadlineMessage(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c75-deadline
  namespace: default
spec:
  entrypoint: main
  activeDeadlineSeconds: 30
  onExit: echo
  templates:
  - name: main
    steps:
    - - name: a
        template: echo
        hooks:
          exit:
            template: echo
    - - name: b
        template: echo
  - name: echo
    container: {image: alpine, command: [echo]}
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for round := 1; round <= 4; round++ {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			switch {
			case round == 1:
				return apiv1.PodRunning
			case n.DisplayName == "a", n.Phase.Fulfilled(nil):
				return "" // the killed pod has not gone away yet
			}
			return apiv1.PodSucceeded
		})
		next := woc.wf
		if round == 1 {
			next.Status.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
		}
		woc = newWorkflowOperationCtx(ctx, next, controller)
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	a, err := woc.wf.GetNodeByName("r4-c75-deadline[0].a")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("child '%s' failed", a.ID), woc.wf.Status.Message)
	steps, err := woc.wf.GetNodeByName("r4-c75-deadline")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("child '%s' failed", a.ID), steps.Message)
	if sg1, err := woc.wf.GetNodeByName("r4-c75-deadline[1]"); err == nil {
		// Absent at base; on the branch the Omitted b gives it an Omitted
		// group (R4/R10). It never ran, so it is never Failed.
		assert.Equal(t, wfv1.NodeOmitted, sg1.Phase, "a StepGroup that never ran")
		assert.NotEqual(t, "Step exceeded its deadline", sg1.Message, "a StepGroup that never ran is not a deadline-killed step")
	}
}

// TestRegressionR4_C85_StepGroupStartedAt ports
// TestProbe_v1x35_StepGroupStartedAt (v1x35-1_test.go / C85), keeping only
// its "[1] not before [0].FinishedAt" check. The nodes are back-dated after
// the first reconcile instead of sleeping. A StepGroup starts when the group
// before it has finished; HEAD created every group with the Steps node, so
// [1]'s StartedAt (and its UI duration) included group [0]'s run.
func TestRegressionR4_C85_StepGroupStartedAt(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c85-sg
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: work
    - - name: B
        template: work
  - name: work
    container:
      image: alpine
      command: [echo, hi]
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	r4BackdateNodes(t, ctx, controller, woc.wf, time.Hour)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = r4Operate(t, ctx, controller, woc.wf)
	sg0, err := woc.wf.GetNodeByName("r4-c85-sg[0]")
	require.NoError(t, err)
	sg1, err := woc.wf.GetNodeByName("r4-c85-sg[1]")
	require.NoError(t, err)
	assert.False(t, sg1.StartedAt.Before(&sg0.FinishedAt), "mid: StepGroup [1] started %s, before StepGroup [0] finished %s", sg1.StartedAt, sg0.FinishedAt)

	woc = r4DriveToEnd(t, ctx, controller, woc, 4)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	sg0, err = woc.wf.GetNodeByName("r4-c85-sg[0]")
	require.NoError(t, err)
	sg1, err = woc.wf.GetNodeByName("r4-c85-sg[1]")
	require.NoError(t, err)
	assert.False(t, sg1.StartedAt.Before(&sg0.FinishedAt), "end: StepGroup [1] started %s, before StepGroup [0] finished %s", sg1.StartedAt, sg0.FinishedAt)
}

// TestRegressionR4_C85_ExpandedStartedAtNoSleep is the back-dated variant of
// TestProbe_v1x35_ExpandedStartedAt (wp7.md §4.2 / C85). {{steps.a.startedAt}}
// of an expanded step reads its StepGroup, so it is when group [1] started,
// like its non-expanded sibling plain; HEAD created the group with the Steps
// node, an hour earlier once back-dated.
func TestRegressionR4_C85_ExpandedStartedAtNoSleep(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c85-items
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: first
        template: echo
    - - name: a
        template: echo
        withItems: [one, two]
      - name: plain
        template: echo
    - - name: b
        template: echo-p
        arguments:
          parameters:
          - name: p
            value: "{{steps.a.startedAt}}"
          - name: q
            value: "{{steps.plain.startedAt}}"
  - name: echo
    container:
      image: alpine
      command: [echo]
  - name: echo-p
    inputs:
      parameters:
      - name: p
      - name: q
    container:
      image: alpine
      command: [echo]
      args: ["{{inputs.parameters.p}}", "{{inputs.parameters.q}}"]
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	r4BackdateNodes(t, ctx, controller, woc.wf, time.Hour)
	woc = r4DriveToEnd(t, ctx, controller, woc, 6)
	b, err := woc.wf.GetNodeByName("r4-c85-items[2].b")
	require.NoError(t, err)
	require.NotNil(t, b.Inputs)
	params := map[string]time.Time{}
	for _, prm := range b.Inputs.Parameters {
		at, err := time.Parse(time.RFC3339, prm.Value.String())
		require.NoError(t, err, prm.Name)
		params[prm.Name] = at
	}
	require.Len(t, params, 2)
	assert.WithinDuration(t, params["q"], params["p"], time.Minute, "steps.a.startedAt must be when group [1] started, as steps.plain.startedAt is")
}

// TestRegressionR4_C30_EmptyGroupRetry ports
// TestProbe_wp7_EmptyGroupRetryAfterFailure
// (plan-inputs/wp7-empty-group-retry_test.go / C30, P12). An empty group
// after a failed group ends Omitted: HEAD marked it Succeeded in the first
// reconcile, so the failed first step had a Succeeded descendant and `argo
// retry` would not reset it; the retried workflow then never completed.
func TestRegressionR4_C30_EmptyGroupRetry(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C30EmptyGroup)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = r4Operate(t, ctx, controller, woc.wf)
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	retried := r4RetryStored(t, ctx, controller, woc.wf)
	woc = r4Operate(t, ctx, controller, retried)
	woc = r4DriveToEnd(t, ctx, controller, woc, 6)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Empty(t, r4UnfulfilledTyped(woc.wf))
}

// TestRegressionR4_C85_EmptyGroupClosedBeforeNext extends C85 to an empty
// group: when second starts, the empty group [1] before it has already
// finished, so [2] does not start before [1] finishes (in-memory node times,
// as the reconcile recorded them).
func TestRegressionR4_C85_EmptyGroupClosedBeforeNext(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C30EmptyGroup)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = r4Operate(t, ctx, controller, woc.wf)
	sg1, err := woc.wf.GetNodeByName("r4-c30-empty[1]")
	require.NoError(t, err)
	sg2, err := woc.wf.GetNodeByName("r4-c30-empty[2]")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, sg1.Phase)
	require.False(t, sg1.FinishedAt.IsZero(), "[1] must have finished")
	assert.False(t, sg2.StartedAt.Before(&sg1.FinishedAt), "StepGroup [2] started %s, before StepGroup [1] finished %s", sg2.StartedAt, sg1.FinishedAt)
}

// r4C3SkippedHandler is a DAG with a failing task A and a when-false notify handler
// that runs on A.Failed: the notify-on-failure pattern (C3).
const r4C3SkippedHandler = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c3-notify
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: notify
      value: "no"
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: bad
      - name: notify
        template: ok
        depends: A.Failed
        when: "{{workflow.parameters.notify}} == yes"
  - name: ok
    container: {image: alpine, command: [sh, -c, "exit 0"]}
  - name: bad
    container: {image: alpine, command: [sh, -c, "exit 1"]}
`

// TestRegressionR4_C3_SkippedOnFailureHandler ports
// TestProbe_v1x11_SkippedOnFailureHandler (v1x11-1_test.go / C3). A leaf
// that did not run takes the phase of the branch above it, as main's
// assessDAGPhase did: the Skipped notify leaf inherits A's failure, so the
// DAG fails. HEAD let only an Omitted leaf inherit, so the workflow ended
// Succeeded although nothing handled the failure.
func TestRegressionR4_C3_SkippedOnFailureHandler(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C3SkippedHandler)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode("r4-c3-notify.A"))
	for range 2 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	notify, err := woc.wf.GetNodeByName("r4-c3-notify.notify")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSkipped, notify.Phase)
	dagNode, err := woc.wf.GetNodeByName("r4-c3-notify")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, dagNode.Phase)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestRegressionR4_C3_DaemonLeafAfterContinueOn ports
// TestProbe_v1x11_DaemonLeafAfterContinueOn (v1x11-1_test.go / C3). A
// running daemon leaf has not completed, so it takes the phase of the branch
// above it: A failed (continueOn only lets D start), so the DAG fails. HEAD
// read the daemon as Succeeded and the workflow Succeeded.
func TestRegressionR4_C3_DaemonLeafAfterContinueOn(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c3-daemon
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: bad
        continueOn:
          failed: true
      - name: D
        template: daemon
        dependencies: [A]
  - name: daemon
    daemon: true
    container: {image: alpine, command: [sleep, "1000"]}
  - name: bad
    container: {image: alpine, command: [sh, -c, "exit 1"]}
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode("r4-c3-daemon.A"))
	woc = r4Operate(t, ctx, controller, woc.wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode("r4-c3-daemon.D"), r4WithReady)
	for range 2 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestRegressionR4_C10_LeafAfterDependencyPodLost ports
// TestProbe_v3x5_LeafAfterDependencyPodLost (v3x5-1_test.go / C10). The
// running pod of leaf a is deleted: its node goes Error "pod deleted" with
// its task result unsynced, so it has not finished, and the pod is
// recreated. The DAG waits for it, as main's did. HEAD read the transient
// Error as final and ended the workflow Error while a's new pod ran and
// Succeeded.
func TestRegressionR4_C10_LeafAfterDependencyPodLost(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c10-chain
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: gen
        template: ok
      - name: a
        template: ok
        depends: gen
  - name: ok
    container: {image: alpine, command: [sh, -c, "exit 0"]}
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	const gen, a = "r4-c10-chain.gen", "r4-c10-chain.a"
	woc := r4Operate(t, ctx, controller, wf)
	op := func() {
		r4MoveNewPodsPending(ctx, woc)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(gen), withExitCode(0))
	op()
	op()
	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(a), r4IncompleteTaskResult(ctx))
	op()
	r4DeletePod(ctx, t, woc, a)
	for range 3 {
		op()
		dagNode, err := woc.wf.GetNodeByName("r4-c10-chain")
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeRunning, dagNode.Phase, "the DAG while a's pod is recreated")
	}
	require.Contains(t, r4PodNodeNames(ctx, t, woc), a, "the deleted pod was not recreated")

	r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, r4PodForNode(a))
	op()
	r4CompleteTaskResult(ctx, t, woc, a)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodSucceeded, r4PodForNode(a), withExitCode(0))
	for range 3 {
		op()
	}
	n, err := woc.wf.GetNodeByName(a)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "message %q", woc.wf.Status.Message)
}

// TestRegressionR4_C71_ContinueOnMissingOutputWithFailedSibling ports
// TestProbe_v1x15_ContinueOnMissingOutputWithFailedSibling (v1x15-1_test.go
// / C71). B references an output its continueOn-failed dependency A never
// wrote, so B can never be created. With failFast (the default) main's
// target loop passed over a target with no node and ended the DAG Failed on
// its failed sibling C; HEAD read B as Pending and hung Running.
func TestRegressionR4_C71_ContinueOnMissingOutputWithFailedSibling(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c71
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: producer
        continueOn:
          failed: true
      - name: B
        template: consumer
        dependencies: [A]
        arguments:
          parameters:
          - name: p
            value: "{{tasks.A.outputs.parameters.p}}"
      - name: C
        template: bad
  - name: producer
    container: {image: alpine, command: [sh, -c, "exit 3"]}
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
  - name: consumer
    inputs:
      parameters:
      - name: p
    container: {image: alpine, command: [echo, "{{inputs.parameters.p}}"]}
  - name: bad
    container: {image: alpine, command: [sh, -c, "exit 1"]}
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, func(pod *apiv1.Pod) bool { return true })
	for range 6 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	_, err := woc.wf.GetNodeByName("r4-c71.B")
	require.Error(t, err, "B can never be created")
	c, err := woc.wf.GetNodeByName("r4-c71.C")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, c.Phase)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// r4C72Run runs a DAG with dag.target "C A" (and failFast as given, "" for
// the default) where A ends Error (its pod is Unknown) and C Failed, and
// returns the DAG node.
func r4C72Run(t *testing.T, failFast string) (*wfOperationCtx, *wfv1.NodeStatus) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	ff := ""
	if failFast != "" {
		ff = "\n      failFast: " + failFast
	}
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c72
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      target: "C A"` + ff + `
      tasks:
      - name: A
        template: ok
      - name: C
        template: ok
  - name: ok
    container: {image: alpine, command: [sh, -c, "exit 0"]}
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	woc := r4Operate(t, ctx, controller, wf)
	r4SetPodsPhase(t, ctx, woc, apiv1.PodUnknown, r4PodForNode("r4-c72.A"))
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, r4PodForNode("r4-c72.C"))
	for range 4 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	for name, want := range map[string]wfv1.NodePhase{"r4-c72.A": wfv1.NodeError, "r4-c72.C": wfv1.NodeFailed} {
		n, err := woc.wf.GetNodeByName(name)
		require.NoError(t, err)
		require.Equal(t, want, n.Phase, name)
	}
	dagNode, err := woc.wf.GetNodeByName("r4-c72")
	require.NoError(t, err)
	return woc, dagNode
}

// TestRegressionR4_C72_TargetOrderFailFast ports
// TestProbe_v1x19_TargetOrderFailFast (v1x19-1_test.go / C72). With
// failFast, the first failed target in the order dag.target is written
// decides the DAG's phase, as on main: C (Failed) before A (Error). HEAD
// sorted the targets, so A decided and the workflow ended Error.
func TestRegressionR4_C72_TargetOrderFailFast(t *testing.T) {
	woc, dagNode := r4C72Run(t, "")
	assert.Equal(t, wfv1.NodeFailed, dagNode.Phase)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestRegressionR4_C72_TargetOrderNoFailFast ports
// TestProbe_v1x19_TargetOrderNoFailFast (v1x19-1_test.go / C72). Without
// failFast every target is looked at in written order and the last failed
// one decides, as on main: A (Error). HEAD's sort made it C (Failed).
func TestRegressionR4_C72_TargetOrderNoFailFast(t *testing.T) {
	woc, dagNode := r4C72Run(t, "false")
	assert.Equal(t, wfv1.NodeError, dagNode.Phase)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}

// r4Run is a workflow under test, from r4Start.
type r4Run struct {
	t          *testing.T
	controller *WorkflowController
	woc        *wfOperationCtx
}

// r4Start validates manifest, creates a controller for it and operates once.
func r4Start(t *testing.T, manifest string) (context.Context, *r4Run) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	return ctx, &r4Run{t: t, controller: controller, woc: r4Operate(t, ctx, controller, wf)}
}

// op moves newly created pods to Pending, as a kubelet would, and reconciles
// again from the stored status.
func (r *r4Run) op(ctx context.Context) {
	r.t.Helper()
	r4MoveNewPodsPending(ctx, r.woc)
	r.woc = r4Operate(r.t, ctx, r.controller, r.woc.wf)
}

// r4PodForNodePrefix matches the pods of the named node and of its items.
func r4PodForNodePrefix(prefix string) func(*apiv1.Pod) bool {
	return func(pod *apiv1.Pod) bool {
		return strings.HasPrefix(pod.Annotations[common.AnnotationKeyNodeName], prefix)
	}
}

// r4NodePhase is the phase of the named node, or "" if it does not exist.
func r4NodePhase(woc *wfOperationCtx, name string) wfv1.NodePhase {
	if n, err := woc.wf.GetNodeByName(name); err == nil {
		return n.Phase
	}
	return ""
}

// TestRegressionR4_C9_SingleStepForceDeletedRerunSucceeds ports
// TestProbe_v2x9_SingleStepForceDeletedRerunSucceeds (v2x9-1_test.go / C9).
// a's running pod vanishes before its task result is complete: a is Error
// "pod deleted" but has not finished, and its pod is recreated. Its
// StepGroup waits for the re-run, as main's did; HEAD recorded the group
// Failed on the transient Error and the workflow ended Failed although the
// re-run Succeeded.
func TestRegressionR4_C9_SingleStepForceDeletedRerunSucceeds(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c9-single
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
    - - name: b
        template: work
  - name: work
    container: {image: alpine, command: [sh, -c, "exit 0"]}
`)
	const a, b, sg0 = "r4-c9-single[0].a", "r4-c9-single[1].b", "r4-c9-single[0]"
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodRunning, r4PodForNode(a), r4IncompleteTaskResult(ctx))
	r.op(ctx)
	r4DeletePod(ctx, t, r.woc, a)
	for range 2 {
		r.op(ctx)
		assert.Equal(t, wfv1.NodeRunning, r4NodePhase(r.woc, sg0), "step group [0] while a's pod is recreated")
	}
	require.Contains(t, r4PodNodeNames(ctx, t, r.woc), a, "the deleted pod was not recreated")

	r4CompleteTaskResult(ctx, t, r.woc, a)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode(a), withExitCode(0))
	for range 2 {
		r.op(ctx)
	}
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode(b), withExitCode(0))
	for range 3 {
		r.op(ctx)
	}
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(r.woc, sg0), "step group [0]")
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, "message %q", r.woc.wf.Status.Message)
}

// TestRegressionR4_C9_ForceDeletedPodContinueOnErrorRerunFails ports
// TestProbe_v2x9_ForceDeletedPodContinueOnErrorRerunFails (v2x9-1_test.go /
// C9). As above, with continueOn.error on a, whose re-run then Fails, which
// continueOn.error does not cover. HEAD recorded the group Succeeded on the
// transient Error and the workflow Succeeded although a Failed.
func TestRegressionR4_C9_ForceDeletedPodContinueOnErrorRerunFails(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c9-coe
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
        continueOn:
          error: true
    - - name: b
        template: work
  - name: work
    container: {image: alpine, command: [sh, -c, "exit 0"]}
`)
	const a, sg0 = "r4-c9-coe[0].a", "r4-c9-coe[0]"
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodRunning, r4PodForNode(a), r4IncompleteTaskResult(ctx))
	r.op(ctx)
	r4DeletePod(ctx, t, r.woc, a)
	for range 3 {
		r.op(ctx)
	}
	require.Contains(t, r4PodNodeNames(ctx, t, r.woc), a, "the deleted pod was not recreated")

	r4CompleteTaskResult(ctx, t, r.woc, a)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodFailed, r4PodForNode(a), withExitCode(1))
	for range 4 {
		r.op(ctx)
	}
	assert.Equal(t, wfv1.NodeFailed, r4NodePhase(r.woc, a))
	assert.Equal(t, wfv1.NodeFailed, r4NodePhase(r.woc, sg0), "step group [0]")
	assert.Equal(t, wfv1.WorkflowFailed, r.woc.wf.Status.Phase)
}

const r4C23DaemonSteps = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c23
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: server
        template: daemon
    - - name: client
        template: ok
        arguments:
          parameters:
          - name: ip
            value: "{{steps.server.ip}}"
  - name: daemon
    daemon: true
    container: {image: alpine, command: [sh, -c, "sleep 9999"]}
  - name: ok
    inputs:
      parameters:
      - name: ip
    container: {image: alpine, command: [sh, -c, "echo {{inputs.parameters.ip}}"]}
`

// r4C23Drive brings server up as a ready daemon at 10.0.0.7 and ends client
// with clientPhase, until the workflow completes. It checks what C23 broke:
// every StepGroup has finished and every node hangs off the root.
func r4C23Drive(t *testing.T, clientPhase apiv1.PodPhase) *wfOperationCtx {
	t.Helper()
	ctx, r := r4Start(t, r4C23DaemonSteps)
	withIP := func(pod *apiv1.Pod, _ *wfOperationCtx) { pod.Status.PodIP = "10.0.0.7" }
	for i := 0; i < 8 && !r.woc.wf.Status.Phase.Completed(); i++ {
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodRunning, r4PodForNode("r4-c23[0].server"), r4WithReady, withIP)
		r4SetPodsPhase(t, ctx, r.woc, clientPhase, r4PodForNode("r4-c23[1].client"))
		r.op(ctx)
	}
	for _, n := range r.woc.wf.Status.Nodes {
		if n.Type == wfv1.NodeTypeStepGroup {
			assert.True(t, n.Fulfilled(), "StepGroup %s left %s after the workflow %s", n.Name, n.Phase, r.woc.wf.Status.Phase)
		}
		assert.True(t, r4Reachable(r.woc, n.Name), "%s is not reachable from the root", n.Name)
	}
	return r.woc
}

// TestRegressionR4_C23_DaemonStepsSucceeded ports
// TestProbe_v1x1_DaemonStepsSucceeded (v1x1-1_test.go / C23). A StepGroup
// whose daemon step is up has finished, as a running daemon has for its
// dependants: main recorded [0] Succeeded and hung [1] off the daemon. HEAD
// kept [0] Running for the daemon's life, so [1] was never linked and [0]
// stayed Running in the finished workflow.
func TestRegressionR4_C23_DaemonStepsSucceeded(t *testing.T) {
	woc := r4C23Drive(t, apiv1.PodSucceeded)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	client, err := woc.wf.GetNodeByName("r4-c23[1].client")
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.7", client.Inputs.Parameters[0].Value.String())
}

// TestRegressionR4_C23_DaemonStepsClientFailedArgoRetry ports
// TestProbe_v1x1_DaemonStepsClientFailedArgoRetry (v1x1-1_test.go / C23).
// With [1] unlinked, `argo retry` of the failed workflow could not find
// [1]'s parent.
func TestRegressionR4_C23_DaemonStepsClientFailedArgoRetry(t *testing.T) {
	woc := r4C23Drive(t, apiv1.PodFailed)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	_, _, err := wfutil.FormulateRetryWorkflow(logging.TestContext(t.Context()), woc.wf, false, "", nil)
	require.NoError(t, err, "argo retry of the failed workflow")
}

// TestRegressionR4_C50_StepsDaemonDiesTrajectory ports
// TestProbe_v1x67_StepsDaemonDiesTrajectory (v1x67-1_test.go / C49, the
// fresh-run form of C50's mechanism). db's StepGroup was recorded
// Succeeded once db was up; db then dies while test runs. Each group's
// phase is derived from its steps on every reconcile, as main re-assessed
// it, so the Steps node fails straight away, naming db, and report never
// starts. [0] itself keeps the Succeeded it was recorded with (P16). HEAD
// never looked at a finished group again: report ran and the workflow
// failed only afterwards, or not at all.
func TestRegressionR4_C50_StepsDaemonDiesTrajectory(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c50
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: db
        template: daemon
    - - name: test
        template: work
    - - name: report
        template: work
  - name: daemon
    daemon: true
    container: {image: alpine, command: [sleep, infinity]}
  - name: work
    container: {image: alpine, command: [echo]}
`)
	const db, test, report = "r4-c50[0].db", "r4-c50[1].test", "r4-c50[2].report"
	reportStarted := false
	stage := func(phase apiv1.PodPhase, name string, rounds int, with ...with) {
		r4SetPodsPhase(t, ctx, r.woc, phase, r4PodForNode(name), with...)
		for range rounds {
			r.op(ctx)
			reportStarted = reportStarted || r4NodePhase(r.woc, report) != ""
		}
	}
	stage(apiv1.PodRunning, db, 2, r4WithReady)
	stage(apiv1.PodRunning, test, 2)
	stage(apiv1.PodFailed, db, 3)
	assert.Equal(t, wfv1.WorkflowFailed, r.woc.wf.Status.Phase, "the workflow once db has died")
	dbNode, err := r.woc.wf.GetNodeByName(db)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("child '%s' failed", dbNode.ID), r.woc.wf.Status.Message)
	stage(apiv1.PodSucceeded, test, 3)
	stage(apiv1.PodSucceeded, report, 3)
	assert.False(t, reportStarted, "report must never start")
	assert.Equal(t, wfv1.WorkflowFailed, r.woc.wf.Status.Phase)
}

// r4C89Crash runs [[A (daemon)]], [[B]], where A is expanded or not: A comes
// up, B starts, A's pods crash while B runs, then B succeeds.
func r4C89Crash(t *testing.T, name, a string) *wfOperationCtx {
	t.Helper()
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: `+name+`
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: daemon`+a+`
    - - name: B
        template: ok
  - name: ok
    container: {image: alpine, command: [sh, -c, "exit 0"]}
  - name: daemon
    daemon: true
    container: {image: alpine, command: [sleep, "1000"]}
`)
	aPods, b := r4PodForNodePrefix(name+"[0].A"), name+"[1].B"
	r.op(ctx)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodRunning, aPods, r4WithReady)
	r.op(ctx)
	r.op(ctx)
	require.NotEmpty(t, r4NodePhase(r.woc, b), "B must start once A is up")
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodRunning, r4PodForNode(b))
	r.op(ctx)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodFailed, aPods)
	r.op(ctx)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode(b))
	for range 3 {
		r.op(ctx)
	}
	return r.woc
}

// TestRegressionR4_C89_StepsExpandedDaemonCrashNextGroup ports
// TestProbe_lead1_StepsExpandedDaemonCrashNextGroup (lead1-1_test.go /
// C89). A's daemon items crash after [0] was recorded Succeeded: the
// TaskGroup's outcome fails with them, so [0]'s derived phase does, and so
// does the workflow, as on main. HEAD never looked at [0] again and the
// workflow Succeeded.
func TestRegressionR4_C89_StepsExpandedDaemonCrashNextGroup(t *testing.T) {
	woc := r4C89Crash(t, "r4-c89", "\n        withItems: [p, q]")
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "daemon items crashed; the workflow must not succeed")
}

// TestRegressionR4_C89_StepsPlainDaemonCrashControl ports
// TestProbe_lead1_StepsPlainDaemonCrashControl (lead1-1_test.go). Control:
// the same crash of a plain daemon step fails the workflow at base and
// HEAD. Fixing C23 without deriving a recorded group's phase again would
// break it: [0] would be Succeeded for good once the daemon was up.
func TestRegressionR4_C89_StepsPlainDaemonCrashControl(t *testing.T) {
	woc := r4C89Crash(t, "r4-c89c", "")
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestRegressionR4_C74_StepGroupWaitsForOnExit ports
// TestProbe_v1x6_StepGroupWaitsForOnExit (v1x6-1_test.go / C74). A's step
// group waits for A's exit handler, as main's did: it stays Running, with
// no FinishedAt, while the handler runs, and does not finish before it.
// HEAD recorded the group Succeeded as soon as A's pod did.
func TestRegressionR4_C74_StepGroupWaitsForOnExit(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c74
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: work
        onExit: exit-handler
    - - name: B
        template: work
  - name: work
    container: {image: alpine, command: [echo, hi]}
  - name: exit-handler
    container: {image: alpine, command: [echo, bye]}
`)
	const a, exit, sg0, b = "r4-c74[0].A", "r4-c74[0].A.onExit", "r4-c74[0]", "r4-c74[1].B"
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode(a))
	r.op(ctx)
	for _, phase := range []apiv1.PodPhase{apiv1.PodRunning, apiv1.PodSucceeded} {
		exitNode, err := r.woc.wf.GetNodeByName(exit)
		require.NoError(t, err)
		require.False(t, exitNode.Fulfilled(), "precondition: the exit handler runs")
		sg, err := r.woc.wf.GetNodeByName(sg0)
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeRunning, sg.Phase, "step group [0] while A's exit handler runs")
		assert.True(t, sg.FinishedAt.IsZero(), "step group [0] FinishedAt while A's exit handler runs")
		assert.Empty(t, r4NodePhase(r.woc, b), "B must not start while A's exit handler runs")
		time.Sleep(20 * time.Millisecond)
		r4SetPodsPhase(t, ctx, r.woc, phase, r4PodForNode(exit))
		r.op(ctx)
	}
	for i := 0; i < 4 && !r.woc.wf.Status.Phase.Completed(); i++ {
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode(b))
		r.op(ctx)
	}
	require.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase)
	sg, err := r.woc.wf.GetNodeByName(sg0)
	require.NoError(t, err)
	exitNode, err := r.woc.wf.GetNodeByName(exit)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, sg.Phase)
	assert.False(t, sg.FinishedAt.Before(&exitNode.FinishedAt), "step group [0] finished before A's exit handler")
}

// r4RunDecided reconciles manifest, deciding each pod's phase from its node,
// until the workflow completes (at most rounds), then three more times.
// It returns the node names of the workflow's pods.
func r4RunDecided(t *testing.T, manifest string, rounds int, decide func(*wfv1.NodeStatus) apiv1.PodPhase) (*wfOperationCtx, []string) {
	t.Helper()
	ctx, r := r4Start(t, manifest)
	for i := 0; i < rounds && !r.woc.wf.Status.Phase.Completed(); i++ {
		setPodPhases(ctx, r.woc, decide)
		r.op(ctx)
	}
	for range 3 {
		setPodPhases(ctx, r.woc, decide)
		r.op(ctx)
	}
	return r.woc, r4PodNodeNames(ctx, t, r.woc)
}

const r4C74SpecRetryHooked = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c74-sr
  namespace: default
spec:
  entrypoint: main
  retryStrategy:
    limit: 1
  templates:
  - name: main
    steps:
    - - name: a
        template: work
        hooks:
          exit:
            template: hook
    - - name: b
        template: work
  - name: work
    container: {image: alpine, command: [echo]}
  - name: hook
    container: {image: alpine, command: [echo]}
`

// TestRegressionR4_C74_HooksExitAllSucceed ports
// TestProbe_lead10_StepsHooksExitAllSucceed (lead10-1_test.go / C74, lead
// 10). Under spec.retryStrategy a's exit hook runs once. HEAD recorded [0]
// before the hook finished and linked [1] under the hook's Retry node,
// which then started a second hook attempt.
func TestRegressionR4_C74_HooksExitAllSucceed(t *testing.T) {
	woc, pods := r4RunDecided(t, r4C74SpecRetryHooked, 20, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	hooks := slices.DeleteFunc(pods, func(name string) bool { return !strings.Contains(name, "onExit") })
	assert.Len(t, hooks, 1, "exit hook pods %v", hooks)
}

// TestRegressionR4_C74_SpecRetryStepsHookedFail ports
// TestProbe_lead7_SpecRetryStepsHookedFail (lead7-1_test.go / C74, lead 7).
// a always fails: the workflow makes exactly spec.retryStrategy's two
// attempts, each running a twice, and leaves nothing running. HEAD's early
// link started an attempt beyond the limit and left nodes Running in the
// completed workflow.
func TestRegressionR4_C74_SpecRetryStepsHookedFail(t *testing.T) {
	isA := func(name string) bool {
		return strings.Contains(name, ".a(") && !strings.Contains(name, "onExit") && !strings.Contains(name, "hooks")
	}
	woc, pods := r4RunDecided(t, r4C74SpecRetryHooked, 16, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if isA(n.Name) || strings.HasSuffix(n.Name, ".a") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	for i := range 2 {
		assert.NotEmpty(t, r4NodePhase(woc, fmt.Sprintf("r4-c74-sr(%d)", i)), "attempt %d", i)
	}
	assert.Empty(t, r4NodePhase(woc, "r4-c74-sr(2)"), "no attempt beyond the limit")
	assert.Len(t, slices.DeleteFunc(pods, func(name string) bool { return !isA(name) }), 4, "2 workflow attempts x 2 attempts of a")
	assert.Empty(t, r4Unfulfilled(woc), "nodes left unfulfilled")
}

// TestRegressionR4_C53_StepsFailFastItems ports TestProbe_v1x46_StepsFailFastItems
// (v1x46-1_test.go / C53). A single-group Steps template with an expanded
// step (withItems) and failFast+parallelism ends the workflow Failed while
// one item is still running. The TaskGroup node between the StepGroup and
// its items is never assessed by the failFast branch (it only walked to a
// StepGroup leaf), so it is left Running forever inside a Failed workflow.
func TestRegressionR4_C53_StepsFailFastItems(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c53-items
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    parallelism: 2
    failFast: true
    steps:
    - - name: s
        template: work
        withItems: [p1, p2, p3, p4, p5]
  - name: work
    container: {image: alpine, command: [echo]}
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	first := true
	setPodPhases(ctx, woc, func(*wfv1.NodeStatus) apiv1.PodPhase {
		if first {
			first = false
			return apiv1.PodFailed
		}
		return apiv1.PodRunning
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.Fulfilled() {
			return ""
		}
		return apiv1.PodSucceeded
	})
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Empty(t, r4UnfulfilledTyped(woc.wf), "nodes left non-terminal in a finished workflow")
}

// r4C81LoopWf is a single expanded step (withItems: [a, b]).
const r4C81LoopWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c81-loop
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: print
        template: echo
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [a, b]
  - name: echo
    inputs:
      parameters:
      - name: msg
    container: {image: alpine, command: [echo, "{{inputs.parameters.msg}}"]}
`

// TestRegressionR4_C81_ExpandedStepFailureMessageNamesFailedItem ports
// TestProbe_v1x54_ExpandedStepFailureMessageNamesFailedItem (v1x54-1_test.go
// / C81). When a withItems step fails, the StepGroup, Steps and workflow
// message must name the failed item's own node (the one with the pod), as
// executeStepGroup did, not the TaskGroup HEAD inserts between the StepGroup
// and its items: the TaskGroup has no pod and an empty message, and the same
// ID would appear whichever item failed.
func TestRegressionR4_C81_ExpandedStepFailureMessageNamesFailedItem(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C81LoopWf)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 3 && woc.wf.Status.Phase == wfv1.WorkflowRunning; i++ {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Fulfilled() {
				return ""
			}
			if strings.Contains(n.DisplayName, "(0:a)") {
				return apiv1.PodFailed
			}
			return apiv1.PodSucceeded
		})
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	item := woc.wf.Status.Nodes.FindByDisplayName("print(0:a)")
	require.NotNil(t, item)
	assert.Equal(t, wfv1.NodeFailed, item.Phase)
	want := fmt.Sprintf("child '%s' failed", item.ID)
	sg, err := woc.wf.GetNodeByName("r4-c81-loop[0]")
	require.NoError(t, err)
	steps, err := woc.wf.GetNodeByName("r4-c81-loop")
	require.NoError(t, err)
	assert.Equal(t, want, sg.Message, "StepGroup message")
	assert.Equal(t, want, steps.Message, "Steps message")
	assert.Equal(t, want, woc.wf.Status.Message, "workflow message")
}

// TestRegressionR4_C81_ExpandedRetriedStepFailureMessage ports
// TestProbe_v1x54_ExpandedRetriedStepFailureMessage (v1x54-1_test.go / C81).
// With a retryStrategy on the expanded template, the message names the
// failed item's Retry node, as base did, not the TaskGroup.
func TestRegressionR4_C81_ExpandedRetriedStepFailureMessage(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c81-loop-retry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: print
        template: echo
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [a, b]
  - name: echo
    retryStrategy:
      limit: 1
    inputs:
      parameters:
      - name: msg
    container: {image: alpine, command: [echo, "{{inputs.parameters.msg}}"]}
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 6 && woc.wf.Status.Phase == wfv1.WorkflowRunning; i++ {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Fulfilled() {
				return ""
			}
			if strings.Contains(n.DisplayName, "(0:a)") {
				return apiv1.PodFailed
			}
			return apiv1.PodSucceeded
		})
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	item := woc.wf.Status.Nodes.FindByDisplayName("print(0:a)")
	require.NotNil(t, item)
	assert.Equal(t, wfv1.NodeTypeRetry, item.Type)
	want := fmt.Sprintf("child '%s' failed", item.ID)
	assert.Equal(t, want, woc.wf.Status.Message, "workflow message")
}

// TestRegressionR4_C81_DAGExpandedTaskFailureMessageNamesFailedItem is the
// DAG counterpart of TestRegressionR4_C81_ExpandedStepFailureMessageNamesFailedItem
// (round 1 fix, controller ruling): boundaryFailureMessage named the
// TaskGroup HEAD inserts between a DAG task and its items, using the same
// bare node.ID that stepGroupOutcome used before failedNodeID. Base DAG
// never produced a "child '<id>' failed" message in the first place (it had
// no such format for a withItems task), so this is a decided extension that
// shares failedNodeID's rule with Steps rather than a base behaviour: it
// need not pass at base, only fail on the branch before the fix.
func TestRegressionR4_C81_DAGExpandedTaskFailureMessageNamesFailedItem(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c81-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: print
        template: echo
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [a, b]
  - name: echo
    inputs:
      parameters:
      - name: msg
    container: {image: alpine, command: [echo, "{{inputs.parameters.msg}}"]}
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 3 && woc.wf.Status.Phase == wfv1.WorkflowRunning; i++ {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Fulfilled() {
				return ""
			}
			if strings.Contains(n.DisplayName, "(0:a)") {
				return apiv1.PodFailed
			}
			return apiv1.PodSucceeded
		})
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	item := woc.wf.Status.Nodes.FindByDisplayName("print(0:a)")
	require.NotNil(t, item)
	assert.Equal(t, wfv1.NodeFailed, item.Phase)
	want := fmt.Sprintf("child '%s' failed", item.ID)
	dagNode, err := woc.wf.GetNodeByName("r4-c81-dag")
	require.NoError(t, err)
	assert.Equal(t, want, dagNode.Message, "DAG message")
	assert.Equal(t, want, woc.wf.Status.Message, "workflow message")
}

// r4C12Fanout is v1x29-1's workflow: a withItems step A in group [0] and a
// plain step B in group [1], started by a pre-Engine controller.
const r4C12Fanout = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c12-fanout
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: c
        withItems: [x, z]
    - - name: B
        template: c
  - name: c
    container: {image: busybox, command: [echo]}
`

// r4C12CreatePod creates the pod an older controller made for the legacy
// item node nodeName (template tmplName), in phase, as it would be found by
// the new controller on its first reconcile after the upgrade.
func r4C12CreatePod(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, nodeName, tmplName string, phase apiv1.PodPhase) {
	t.Helper()
	nodeID := wf.NodeID(nodeName)
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        wfutil.GeneratePodName(wf.Name, nodeName, tmplName, nodeID, wfutil.GetWorkflowPodNameVersion(wf)),
			Namespace:   wf.Namespace,
			Labels:      map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyCompleted: "false"},
			Annotations: map[string]string{common.AnnotationKeyNodeID: nodeID, common.AnnotationKeyNodeName: nodeName},
		},
		Spec:   apiv1.PodSpec{Containers: []apiv1.Container{{Name: "main", Image: "busybox"}}},
		Status: apiv1.PodStatus{Phase: phase},
	}
	created, err := controller.kubeclientset.CoreV1().Pods(wf.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, controller.PodController.TestingPodInformer(), created, func(any) bool { return true })
}

// r4C12Drive operates the upgraded workflow until it completes, finishing
// every pod that has not finished yet with Succeeded (and with) after each
// reconcile. It returns the last operation context.
func r4C12Drive(ctx context.Context, controller *WorkflowController, wf *wfv1.Workflow, with ...with) *wfOperationCtx {
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 10 {
		woc.operate(ctx)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Fulfilled() {
				return ""
			}
			return apiv1.PodSucceeded
		}, with...)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
	return woc
}

// r4C12AssertItemsUnderOneParent checks that every legacy item of step A
// hangs off exactly one node: the TaskGroup [0].A when there is one (the
// new controller adopted the items, moving their StepGroup edge to it,
// P14), else the StepGroup [0] (base, which had no TaskGroup). A TaskGroup
// that exists next to the items, rather than above them, fails this.
func r4C12AssertItemsUnderOneParent(t *testing.T, woc *wfOperationCtx, items ...string) {
	t.Helper()
	parent := woc.wf.Name + "[0]"
	if _, err := woc.wf.GetNodeByName(parent + ".A"); err == nil {
		parent += ".A"
	}
	for _, item := range items {
		assert.Equal(t, []string{parent}, r4Parents(woc, woc.wf.Name+"[0]."+item), "parents of %s", item)
	}
}

// r4C12AssertFailedBeforeB checks the outcome both C12 fan-out cases share:
// the failed legacy item fails the workflow, B never runs (it has no pod,
// and is Omitted if it has a node at all, R4/R10), and both legacy items
// hang off one parent (r4C12AssertItemsUnderOneParent).
func r4C12AssertFailedBeforeB(ctx context.Context, t *testing.T, woc *wfOperationCtx) {
	t.Helper()
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	b := woc.wf.Name + "[1].B"
	if node, err := woc.wf.GetNodeByName(b); err == nil {
		assert.Equal(t, wfv1.NodeOmitted, node.Phase, "B must not run after a failed step group")
	}
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	for _, p := range pods.Items {
		assert.NotEqual(t, b, p.Annotations[common.AnnotationKeyNodeName], "B must have no pod")
	}
	r4C12AssertItemsUnderOneParent(t, woc, "A(0:x)", "A(1:z)")
}

// TestRegressionR4_C12_LegacyItemFailedSiblingRunning ports
// TestProbe_v1x29_LegacyShapeItemFailedSiblingRunning (v1x29-1_test.go /
// C12), with "B absent" relaxed to "B has no pod / is Omitted" (R4/R10).
// An older controller left A's items directly under StepGroup [0]: A(0:x)
// had Failed and A(1:z) was still Running. The new controller must adopt
// both items under the TaskGroup it creates, so the failure fails the step
// group, instead of assessing an empty TaskGroup that holds only the items
// it dispatches itself.
func TestRegressionR4_C12_LegacyItemFailedSiblingRunning(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := r4LegacyStepsStatus(r4C12Fanout, 0, "A", "c", []r4LegacyStepItem{{Name: "0:x", Phase: wfv1.NodeFailed}, {Name: "1:z", Phase: wfv1.NodeRunning}})
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	r4C12CreatePod(ctx, t, controller, wf, wf.Name+"[0].A(1:z)", "c", apiv1.PodRunning)

	woc := r4C12Drive(ctx, controller, wf)
	r4C12AssertFailedBeforeB(ctx, t, woc)
}

// TestRegressionR4_C12_LegacyBothDoneWhileDownOneFailed ports
// TestProbe_v1x29_LegacyShapeBothDoneWhileDownOneFailed (v1x29-1_test.go /
// C12), with "B absent" relaxed to "B has no pod / is Omitted" (R4/R10).
// Both legacy items were Running in the old controller's status and both
// pods finished while the controller was being upgraded, A(0:x) Failed.
func TestRegressionR4_C12_LegacyBothDoneWhileDownOneFailed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := r4LegacyStepsStatus(r4C12Fanout, 0, "A", "c", []r4LegacyStepItem{{Name: "0:x", Phase: wfv1.NodeRunning}, {Name: "1:z", Phase: wfv1.NodeRunning}})
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	r4C12CreatePod(ctx, t, controller, wf, wf.Name+"[0].A(0:x)", "c", apiv1.PodFailed)
	r4C12CreatePod(ctx, t, controller, wf, wf.Name+"[0].A(1:z)", "c", apiv1.PodSucceeded)
	wf.Status.MarkTaskResultComplete(ctx, wf.NodeID(wf.Name+"[0].A(1:z)"))

	woc := r4C12Drive(ctx, controller, wf)
	r4C12AssertFailedBeforeB(ctx, t, woc)
}

// TestRegressionR4_C12_LegacyAggregateMidFanout is the self-contained
// equivalent of TestProbe_r1x29_ParamAggUpgradeMidFanout (r1x29-1_test.go /
// C12). An older controller left A's items directly under StepGroup [0]:
// A(0:x) had Succeeded with its output, A(1:z) was still Running. After the
// upgrade A(1:z) finishes; the aggregate {{steps.A.outputs.parameters.out}}
// that C receives must hold both items' outputs, not only the one the new
// controller saw finish.
func TestRegressionR4_C12_LegacyAggregateMidFanout(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := r4LegacyStepsStatus(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c12-agg
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: gen
        withItems: [x, z]
    - - name: C
        template: use
        arguments:
          parameters:
          - name: res
            value: "{{steps.A.outputs.parameters.out}}"
  - name: gen
    outputs:
      parameters:
      - name: out
        valueFrom: {path: /tmp/out}
    container: {image: busybox, command: [echo]}
  - name: use
    inputs:
      parameters:
      - name: res
    container: {image: busybox, command: [echo, "{{inputs.parameters.res}}"]}
`, 0, "A", "gen", []r4LegacyStepItem{{Name: "0:x", Phase: wfv1.NodeSucceeded}, {Name: "1:z", Phase: wfv1.NodeRunning}})
	a0 := wf.NodeID(wf.Name + "[0].A(0:x)")
	done := wf.Status.Nodes[a0]
	done.Outputs = &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "out", Value: wfv1.AnyStringPtr("v-x")}}}
	wf.Status.Nodes[a0] = done
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	r4C12CreatePod(ctx, t, controller, wf, wf.Name+"[0].A(1:z)", "gen", apiv1.PodRunning)

	woc := r4C12Drive(ctx, controller, wf, withOutputs(ctx, wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "out", Value: wfv1.AnyStringPtr("v-z")}}}))
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	c, err := woc.wf.GetNodeByName(woc.wf.Name + "[1].C")
	require.NoError(t, err)
	require.NotNil(t, c.Inputs)
	res := c.Inputs.GetParameterByName("res")
	require.NotNil(t, res)
	assert.Contains(t, res.Value.String(), "v-x", "res holds the item that finished under the old controller")
	assert.Contains(t, res.Value.String(), "v-z", "res holds the item that finished after the upgrade")
	r4C12AssertItemsUnderOneParent(t, woc, "A(0:x)", "A(1:z)")
}

// C35: `argo retry` of a Steps workflow whose failing step is a non-execution
// node (Skipped/Error from an unevaluatable `when` or a malformed withParam,
// or a Suspend node failed by `argo stop --node-field-selector`) followed by
// a later StepGroup. planReset's leaf rule counted a failure only on an
// execution node or a node with no children at all; the on-demand next
// StepGroup, created Omitted under the failed step, gave it a child, so it
// was never treated as the actually-failed node and retry reset nothing.
// Ported from _pr-16290-round4/probes/v2x2-1_test.go.

const r4C35WhenErr = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c35-whenerr
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - {name: x, value: "("}
  templates:
  - name: main
    steps:
    - - {name: a, template: run}
    - - {name: b, template: run, when: "{{workflow.parameters.x}} == ok"}
    - - {name: c, template: run}
  - name: run
    container:
      image: busybox
`

const r4C35WithParamErr = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c35-paramerr
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - {name: list, value: "[a, b"}
  templates:
  - name: main
    steps:
    - - {name: a, template: run}
    - - {name: fan, template: run, withParam: "{{workflow.parameters.list}}"}
    - - {name: c, template: run}
  - name: run
    container:
      image: busybox
`

const r4C35Suspend = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c35-approval
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - {name: a, template: run}
    - - {name: approve, template: wait}
    - - {name: c, template: run}
  - name: wait
    suspend: {}
  - name: run
    container:
      image: busybox
`

func r4C35Run(t *testing.T, yaml string, param string, wantPods int) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(yaml)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	woc := runToCompletion(ctx, t, controller, wf, allSucceed)
	cancel()
	dumpNodes(t, "original run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	retried, _, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", []string{param})
	require.NoError(t, err)
	dumpNodes(t, "after FormulateRetryWorkflow", retried)
	root, err := retried.GetNodeByName(retried.Name)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, root.Phase, "retry must reset the failed Steps node")

	cancel2, controller2 := newController(ctx, retried)
	defer cancel2()
	woc = runToCompletion(ctx, t, controller2, retried, allSucceed)
	dumpNodes(t, "retried run finished", woc.wf)
	assert.Len(t, podNames(ctx, woc), wantPods, "the failed step and the steps after it must run once the retry fixed the parameter")
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestRegressionR4_C35_RetryAfterStepWhenError(t *testing.T) {
	r4C35Run(t, r4C35WhenErr, "x=ok", 2) // b, c
}

func TestRegressionR4_C35_RetryAfterStepWithParamError(t *testing.T) {
	r4C35Run(t, r4C35WithParamErr, `list=["a","b"]`, 3) // fan(0:a), fan(1:b), c
}

// A suspended approval step rejected with `argo stop
// --node-field-selector displayName=approve` (which marks the suspend node
// Failed), then `argo retry`.
func TestRegressionR4_C35_RetryAfterRejectedSuspendStep(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C35Suspend)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 4 {
		woc.operate(ctx)
		setPodPhases(ctx, woc, allSucceed)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
	dumpNodes(t, "suspended", woc.wf)
	// Emulate util.updateSuspendedNode for `argo stop --node-field-selector`.
	found := false
	for id, n := range woc.wf.Status.Nodes {
		if n.IsActiveSuspendNode() && n.DisplayName == "approve" {
			n.Phase = wfv1.NodeFailed
			n.FinishedAt = metav1.Time{Time: time.Now().UTC()}
			n.Message = "rejected"
			woc.wf.Status.Nodes[id] = n
			found = true
		}
	}
	require.True(t, found, "approve must be suspended")
	for i := 0; i < 4 && !woc.wf.Status.Phase.Completed(); i++ {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	cancel()
	dumpNodes(t, "rejected run finished", woc.wf)
	require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	retried, _, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err)
	dumpNodes(t, "after FormulateRetryWorkflow", retried)
	root, err := retried.GetNodeByName(retried.Name)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, root.Phase, "retry must reset the failed Steps node")

	cancel2, controller2 := newController(ctx, retried)
	defer cancel2()
	woc = newWorkflowOperationCtx(ctx, retried, controller2)
	for i := 0; i < 4 && !woc.wf.Status.Phase.Completed(); i++ {
		woc.operate(ctx)
		setPodPhases(ctx, woc, allSucceed)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller2)
	}
	dumpNodes(t, "retried run", woc.wf)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "the retried workflow must be waiting on approval again")
	approve, err := woc.wf.GetNodeByName("r4-c35-approval[1].approve")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, approve.Phase, "approve must be suspended again")
}

// D3: a ContainerSet pod whose containers all finished successfully (exit 0)
// but whose pod is deleted before the wait container finishes uploading ends
// Error "pod deleted", while its Container children keep their true
// Succeeded phase (the node-phase state machine refuses Succeeded->Error, by
// design: D3 decision). isDescendantNodeSucceeded counted those Succeeded
// Container children as a succeeded descendant, so planReset never reset the
// pod node and `argo retry` silently did nothing. Ported from
// _pr-16290-round4/probes/v1x68-1_test.go
// TestProbe_v1x68_CtrSetAllDoneDeletedThenRetry.
const r4D3CtrSetAllDone = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-d3-ctrset-alldone
spec:
  entrypoint: init
  templates:
    - name: init
      dag:
        tasks:
          - name: A
            template: run
    - name: run
      containerSet:
        containers:
          - name: first
            image: alpine:3.23
            command: [echo]
          - name: main
            image: alpine:3.23
            command: [echo]
            dependencies: [first]
`

func TestRegressionR4_D3_CtrSetAllDoneDeletedThenRetry(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(r4D3CtrSetAllDone)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf, nil, validate.Opts{}))
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning, func(pod *apiv1.Pod, _ *wfOperationCtx) {
		pod.Status.ContainerStatuses = []apiv1.ContainerStatus{
			{Name: "first", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0}}},
			{Name: "main", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0}}},
			{Name: "wait", State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
		}
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	dumpNodes(t, "before delete", woc.wf)
	deletePods(ctx, woc)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	dumpNodes(t, "after delete", woc.wf)
	require.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)

	retried, podsToDelete, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), false, "", nil)
	require.NoError(t, err)
	t.Logf("podsToDelete=%v", podsToDelete)
	dumpNodes(t, "after retry formulate", retried)
	podNode := retried.Status.Nodes.FindByDisplayName("A")
	assert.Nil(t, podNode, "pod node A should have been deleted by retry so it re-runs")

	retried, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("").Update(ctx, retried, metav1.UpdateOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, retried, controller)
	woc.operate(ctx)
	dumpNodes(t, "after retry operate", woc.wf)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	t.Logf("pods after retry: %d", len(pods.Items))
	assert.Len(t, pods.Items, 1, "retry should create a new pod for A")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

const r4C28DAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c28-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: work
  - name: work
    container:
      image: alpine
      command: [sh, -c, "exit 1"]
`

// TestRegressionR4_C28_PodDeletedWhileFailedUnsynced ports
// TestProbe_v1x51_DAG (v1x51-1_test.go / C28). A's pod fails before its
// WorkflowTaskResult is complete, so A is Failed but unsynced (not
// fulfilled). The pod is then deleted, and pod reconciliation marks A
// Error "pod deleted". The branch's state machine refused Failed -> Error
// and returned before storing the node, which also dropped the synced
// fix-up, so A never became fulfilled and the workflow stayed Running for
// good. Base moved A to Error, synced, and the workflow ended.
func TestRegressionR4_C28_PodDeletedWhileFailedUnsynced(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
	t.Setenv("RECENTLY_DELETED_POD_DURATION", "0s")
	t.Setenv("TASK_RESULT_TIMEOUT_DURATION", "0s")
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C28DAG)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodRunning, r4IncompleteTaskResult(ctx))
	woc = r4Operate(t, ctx, controller, woc.wf)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = r4Operate(t, ctx, controller, woc.wf)
	a := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, a)
	require.Equal(t, wfv1.NodeFailed, a.Phase)
	require.False(t, a.Fulfilled(), "A must be unsynced: its task result is incomplete")

	deletePods(ctx, woc)
	for range 3 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	dumpNodes(t, "after pod deleted", woc.wf)
	a = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, a)
	assert.Equal(t, wfv1.NodeError, a.Phase)
	assert.Equal(t, "pod deleted", a.Message)
	assert.True(t, woc.wf.Status.Phase.Completed(), "workflow must complete, got %s", woc.wf.Status.Phase)
}

const r4C58CsAutoRestart = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c58-cs-restart
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: cs
        template: cs
  - name: cs
    containerSet:
      containers:
      - name: a
        image: argoproj/argosay:v2
      - name: b
        image: argoproj/argosay:v2
        dependencies: [a]
`

// TestRegressionR4_C58_CsAutoRestartOriginal ports
// TestProbe_v2x11_CsAutoRestartOriginal (v2x11-1_test.go / C58). With
// failedPodRestart enabled, a containerSet pod evicted before its
// containers start is recreated (the pod node goes back to Pending). The
// branch marked each container node Failed "Pod Failed whilst container
// running" on the evicted pod, and the state machine then refused every
// transition out of Failed, so the container nodes stayed Failed inside a
// Succeeded pod node and workflow. Base's container nodes ended Succeeded.
func TestRegressionR4_C58_CsAutoRestartOriginal(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C58CsAutoRestart)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{Enabled: true}

	woc := r4Operate(t, ctx, controller, wf)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)

	// Evicted before any container started: every container is waiting.
	makePodsPhase(ctx, woc, apiv1.PodFailed, func(pod *apiv1.Pod, _ *wfOperationCtx) {
		pod.UID = "r4-c58-evicted-pod-uid"
		pod.Status.Reason = "Evicted"
		pod.Status.Message = "The node was low on resource: memory."
		pod.Status.ContainerStatuses = nil
		for _, c := range pod.Spec.Containers {
			pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, apiv1.ContainerStatus{
				Name:  c.Name,
				State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: "ContainerCreating"}},
			})
		}
	})
	woc = r4Operate(t, ctx, controller, woc.wf)
	dumpNodes(t, "after eviction", woc.wf)
	podNode, err := woc.wf.GetNodeByName("r4-c58-cs-restart[0].cs")
	require.NoError(t, err)
	require.Equal(t, wfv1.NodePending, podNode.Phase, "pod node auto-restarting")

	deletePods(ctx, woc)
	woc = r4Operate(t, ctx, controller, woc.wf)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1, "pod recreated")

	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withExitCode(0))
	for range 4 {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	dumpNodes(t, "after success", woc.wf)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	for _, name := range []string{"r4-c58-cs-restart[0].cs.a", "r4-c58-cs-restart[0].cs.b"} {
		n, err := woc.wf.GetNodeByName(name)
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeSucceeded, n.Phase, "container node %s", name)
	}
}

// C55 (hook side): a hook node that errors while it is created is still
// linked under the node it hooks, so `argo retry` and `argo resubmit
// --memoized` find its parent. The exit-hook cases pass at base (runOnExitNode
// linked unconditionally); the lifecycle-hook and workflow-level-hook cases
// are the decided always-link rule (T4.1) and fail at base, which returned
// before linking for those too.

// r4C55HookLinked runs wf to completion and asserts that hookName is an Error
// node linked under parentName and that `argo retry` (restartSuccessful when
// asked, so every node is reset) accepts the result.
func r4C55HookLinked(t *testing.T, manifest, parentName, hookName string, restartSuccessful bool) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := runToCompletion(ctx, t, controller, wf, allSucceed)
	dumpNodes(t, "finished", woc.wf)
	hook, err := woc.wf.GetNodeByName(hookName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, hook.Phase)
	parent, err := woc.wf.GetNodeByName(parentName)
	require.NoError(t, err)
	assert.True(t, slices.Contains(parent.Children, hook.ID), "hook node %s is not linked under %s", hookName, parentName)
	selector := ""
	if restartSuccessful {
		selector = "phase=Succeeded"
	}
	_, _, err = wfutil.FormulateRetryWorkflow(ctx, woc.wf.DeepCopy(), restartSuccessful, selector, nil)
	require.NoError(t, err, "argo retry")
}

// r4C55HookMissingCM is a hook template whose input comes from a ConfigMap
// that does not exist: its node errors on creation, with no pod.
const r4C55HookMissingCM = `
  - name: hook
    inputs:
      parameters:
      - name: msg
        valueFrom:
          configMapKeyRef:
            name: missing-cm
            key: msg
    container:
      image: busybox
      command: [echo, "{{inputs.parameters.msg}}"]
`

// The exit hook's image has no command, so the controller cannot look up
// its entrypoint and the hook node errors on creation.
func TestRegressionR4_C55_RetryAfterExitHookNodeErrorDag(t *testing.T) {
	r4C55HookLinked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: g2dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: run
      - name: b
        template: run
        depends: a
        hooks:
          exit:
            template: hook
  - name: run
    container:
      image: busybox
  - name: hook
    container:
      image: hookimg
`, "g2dag.b", "g2dag.b.onExit", true)
}

func TestRegressionR4_C55_RetryAfterExitHookArgError(t *testing.T) {
	r4C55HookLinked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: g3steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: run
        hooks:
          exit:
            template: hook
    - - name: b
        template: run
  - name: run
    container:
      image: busybox
      command: [echo]
`+r4C55HookMissingCM, "g3steps[0].a", "g3steps[0].a.onExit", false)
}

// Decided behaviour (always-link rule): fails at base.
func TestRegressionR4_C55_LifecycleHookArgErrorLinked(t *testing.T) {
	r4C55HookLinked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: g4dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: run
        hooks:
          running:
            expression: "true"
            template: hook
  - name: run
    container:
      image: busybox
      command: [echo]
`+r4C55HookMissingCM, "g4dag.a", "g4dag.a.hooks.running", true)
}

// Decided behaviour (always-link rule): fails at base.
func TestRegressionR4_C55_WorkflowHookArgErrorLinked(t *testing.T) {
	r4C55HookLinked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: g5wf
  namespace: default
spec:
  entrypoint: run
  hooks:
    running:
      expression: "true"
      template: hook
  templates:
  - name: run
    container:
      image: busybox
      command: [echo]
`+r4C55HookMissingCM, "g5wf", "g5wf.hooks.running", true)
}

// C29: `argo terminate` while a task's Steps exit hook runs. Base did not
// re-enter the exit-hook node under Terminate (runOnExitNode returned early,
// exit_handler.go:23), but its DAG waited only on its target tasks' exit
// hooks, and build is not a target: assessDAGPhase failed the DAG under the
// shutdown and the workflow ended Failed. The branch waits on every task's
// pending hook and did not re-enter the hook, so the boundary waited on it
// forever; it now re-enters an existing exit-hook node under Terminate too,
// so its Steps node is assessed and finishes.

const r4C29DAGWithStepsHook = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-steps-hook
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: build
        template: work
        hooks:
          exit:
            template: notify
      - name: deploy
        depends: build
        template: work
  - name: notify
    steps:
    - - name: slack
        template: work
    - - name: email
        template: work
  - name: work
    container: {image: alpine, command: [echo]}
`

// r4C29KillActivePods fails every pod that has not finished, as the pod
// controller's container termination would.
func r4C29KillActivePods(ctx context.Context, t *testing.T, woc *wfOperationCtx) {
	t.Helper()
	r4SetPodsPhase(t, ctx, woc, apiv1.PodFailed, func(pod *apiv1.Pod) bool {
		return pod.Status.Phase != apiv1.PodSucceeded && pod.Status.Phase != apiv1.PodFailed
	})
}

// r4C29StartHook runs the DAG until build's exit hook is Running.
func r4C29StartHook(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) *wfOperationCtx {
	t.Helper()
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hook, err := woc.wf.GetNodeByName("dag-steps-hook.build.onExit")
	require.NoError(t, err, "exit hook node should exist")
	require.Equal(t, wfv1.NodeRunning, hook.Phase)
	return woc
}

// The hook's pods are killed after terminate.
func TestRegressionR4_C29_TerminateDAGStepsHook(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C29DAGWithStepsHook)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4C29StartHook(ctx, t, controller, wf)

	woc.wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	for range 8 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		r4C29KillActivePods(ctx, t, woc)
	}
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(woc))
}

// The hook's pod is really Pending (not the fake clientset's empty phase)
// and never changes phase: terminate's execution control fails its node.
func TestRegressionR4_C29_TerminateDAGStepsHookPendingPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C29DAGWithStepsHook)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4C29StartHook(ctx, t, controller, wf)
	r4MoveNewPodsPending(ctx, woc)

	woc.wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	for range 6 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		r4MoveNewPodsPending(ctx, woc)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(woc))
}

// `argo stop` while the task runs (its exit hook starts, as Stop allows),
// then `argo terminate` because the hook is slow.
func TestRegressionR4_C29_StopThenTerminateDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C29DAGWithStepsHook)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	woc.wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	r4C29KillActivePods(ctx, t, woc)
	for range 2 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	hook, err := woc.wf.GetNodeByName("dag-steps-hook.build.onExit")
	require.NoError(t, err, "exit hook should have started under Stop")
	require.False(t, hook.Fulfilled())

	woc.wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	for range 8 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		r4C29KillActivePods(ctx, t, woc)
	}
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(woc))
}

// C65: a task or step exit hook that finishes between reconciles (its pod
// completed, an HTTP call answered, a suspend resumed) must still be
// re-entered once, so its template metrics are emitted and its lock is
// released. The branch re-entered it only while it was unfulfilled.

// r4C65Counter reads a counter metric by name and labels; -1 when it was
// never emitted.
func r4C65Counter(t *testing.T, name string, kv ...string) float64 {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	var attrs []attribute.KeyValue
	for i := 0; i+1 < len(kv); i += 2 {
		attrs = append(attrs, attribute.String(kv[i], kv[i+1]))
	}
	set := attribute.NewSet(attrs...)
	v, err := testExporter.GetFloat64CounterValue(ctx, name, &set)
	if err != nil {
		t.Logf("counter %s %v: %v", name, kv, err)
		return -1
	}
	return v
}

// r4C65HookTemplate is a pod hook template with a completion counter.
func r4C65HookTemplate(metric string) string {
	return `
  - name: hook
    metrics:
      prometheus:
      - name: ` + metric + `
        help: "hook counter"
        labels:
        - key: status
          value: "{{status}}"
        counter:
          value: "1"
    container:
      image: busybox
      command: [echo, hook]
`
}

func TestRegressionR4_C65_HookTemplateMetrics(t *testing.T) {
	const work = `
  - name: work
    container:
      image: busybox
      command: [echo, work]
`
	cases := []struct {
		name, metric, body string
		hooks              int
	}{
		{"dag-task-exit-hook", "r4_c65_dag_task_exit_hook", `
  - name: main
    dag:
      tasks:
      - name: A
        template: work
        hooks:
          exit:
            template: hook
      - name: B
        depends: A
        template: work
`, 1},
		{"steps-step-exit-hook", "r4_c65_steps_step_exit_hook", `
  - name: main
    steps:
    - - name: A
        template: work
        hooks:
          exit:
            template: hook
    - - name: B
        template: work
`, 1},
		{"dag-item-exit-hook", "r4_c65_dag_item_exit_hook", `
  - name: main
    dag:
      tasks:
      - name: A
        template: work
        withItems: [a, b]
        hooks:
          exit:
            template: hook
`, 2},
		// Control: the workflow's own onExit was always re-entered.
		{"wf-onexit", "r4_c65_wf_onexit", `
  - name: main
    dag:
      tasks:
      - name: A
        template: work
`, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			onExit := ""
			if c.name == "wf-onexit" {
				onExit = "\n  onExit: hook"
			}
			wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c65-` + c.name + `
  namespace: default
spec:
  entrypoint: main` + onExit + `
  templates:` + c.body + work + r4C65HookTemplate(c.metric))
			require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			for i := 0; i < 8 && !woc.wf.Status.Phase.Completed(); i++ {
				makePodsPhase(ctx, woc, apiv1.PodSucceeded)
				woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
				woc.operate(ctx)
			}
			require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
			hooks := 0
			for _, n := range woc.wf.Status.Nodes {
				if n.TemplateName == "hook" {
					hooks++
					assert.Equal(t, wfv1.NodeSucceeded, n.Phase, n.Name)
				}
			}
			assert.Equal(t, c.hooks, hooks, "hook nodes")
			assert.InDelta(t, float64(c.hooks), r4C65Counter(t, c.metric, "status", "Succeeded"), 0.001)
			// Further reconciles re-enter the finished hooks but emit nothing more.
			for range 3 {
				woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
				woc.operate(ctx)
			}
			assert.InDelta(t, float64(c.hooks), r4C65Counter(t, c.metric, "status", "Succeeded"), 0.001, "after extra reconciles")
		})
	}
}

// r4C65RunHTTP drives wf: pods succeed, the agent pod stays Running, and
// every task the task set hands to the agent is reported Succeeded.
func r4C65RunHTTP(t *testing.T, manifest string, rounds int) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	t.Cleanup(cancel)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < rounds && !woc.wf.Status.Phase.Completed(); i++ {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Type != wfv1.NodeTypePod {
				return ""
			}
			return apiv1.PodSucceeded
		})
		podcs := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace)
		if agent, err := podcs.Get(ctx, woc.getAgentPodName(), metav1.GetOptions{}); err == nil && agent.Status.Phase != apiv1.PodRunning {
			agent.Status.Phase = apiv1.PodRunning
			updated, err := podcs.Update(ctx, agent, metav1.UpdateOptions{})
			require.NoError(t, err)
			waitForInformer(ctx, controller.PodController.TestingPodInformer(), updated, func(obj any) bool {
				return obj.(*apiv1.Pod).Status.Phase == apiv1.PodRunning
			})
		}
		results := map[string]wfv1.NodeResult{}
		if ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Get(ctx, woc.wf.Name, metav1.GetOptions{}); err == nil {
			for id := range ts.Spec.Tasks {
				if _, done := ts.Status.Nodes[id]; !done {
					results[id] = wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{}}
				}
			}
		}
		r4ReportTaskSet(ctx, t, woc, results)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	return woc
}

// r4ReportTaskSet writes results into the WorkflowTaskSet status, as the
// agent would, and waits for the task set informer to see them.
func r4ReportTaskSet(ctx context.Context, t *testing.T, woc *wfOperationCtx, results map[string]wfv1.NodeResult) {
	t.Helper()
	if len(results) == 0 {
		return
	}
	patch, err := json.Marshal(map[string]any{"status": wfv1.WorkflowTaskSetStatus{Nodes: results}})
	require.NoError(t, err)
	_, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Patch(ctx, woc.wf.Name, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
	require.NoError(t, err)
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, 20*time.Millisecond, 5*time.Second, true, func(context.Context) (bool, error) {
		obj, exists, err := woc.controller.wfTaskSetInformer.Informer().GetIndexer().GetByKey(woc.wf.Namespace + "/" + woc.wf.Name)
		if err != nil || !exists {
			return false, err
		}
		ts := obj.(*wfv1.WorkflowTaskSet)
		for id, r := range results {
			if got, found := ts.Status.Nodes[id]; !found || got.Phase != r.Phase {
				return false, nil
			}
		}
		return true, nil
	}), "task set informer never saw the results")
}

// Two step exit hooks, each an HTTP call holding the same mutex: the first
// must release it when it completes, so the second can take it.
func TestRegressionR4_C65_HTTPExitHookMutexReleased(t *testing.T) {
	woc := r4C65RunHTTP(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-mutex-hooks
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: A
            template: work
            hooks:
              exit:
                template: notify
        - - name: B
            template: work
            hooks:
              exit:
                template: notify
    - name: work
      container:
        image: busybox
        command: [echo, work]
    - name: notify
      synchronization:
        mutexes:
          - name: r4-c65-notify
      http:
        url: http://example.com/notify
`, 10)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(woc))
	hooks := 0
	for _, n := range woc.wf.Status.Nodes {
		if n.TemplateName == "notify" && n.Phase == wfv1.NodeSucceeded {
			hooks++
		}
	}
	assert.Equal(t, 2, hooks, "Succeeded notify hooks")
}

// A step's exit hook is a suspend holding a mutex that the next step also
// needs. Resuming the hook finishes it outside the controller; its mutex
// must still be released so the next step can run.
func TestRegressionR4_C65_ExitHookSuspendMutexReleasedSteps(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: susp-mutex-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: work
        hooks:
          exit:
            template: approve
    - - name: B
        template: locked
  - name: approve
    suspend: {}
  - name: locked
    container: {image: alpine, command: [echo]}
  - name: work
    container: {image: alpine, command: [echo]}
`)
	for i := range wf.Spec.Templates {
		if name := wf.Spec.Templates[i].Name; name == "approve" || name == "locked" {
			wf.Spec.Templates[i].Synchronization = r4NamespacedMutex("r4-c65-m1")
		}
	}
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	advance := func(woc *wfOperationCtx, phase apiv1.PodPhase) *wfOperationCtx {
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Fulfilled() {
				return ""
			}
			return phase
		})
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		return woc
	}
	const hookName, bName = "susp-mutex-steps[0].A.onExit", "susp-mutex-steps[1].B"

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	woc = advance(woc, apiv1.PodSucceeded) // A succeeds; its exit hook suspends holding the mutex
	require.Equal(t, wfv1.NodeRunning, r4NodePhase(woc, hookName))

	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	require.NoError(t, wfutil.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.Name, "displayName=A.onExit"))
	stored, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, stored, controller)
	woc.operate(ctx)
	woc = advance(woc, apiv1.PodPending)
	woc = advance(woc, apiv1.PodPending)

	require.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, hookName))
	b, err := woc.wf.GetNodeByName(bName)
	require.NoError(t, err)
	require.Nil(t, b.SynchronizationStatus, "B still waiting for the lock: %q", b.Message)

	woc = advance(woc, apiv1.PodSucceeded)
	woc = advance(woc, apiv1.PodSucceeded)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(woc))
}

// A task's exit hook takes the workflow's templateDefaults semaphore (limit
// 1); it must release it when it finishes, or the next task waits forever.
func TestRegressionR4_C65_TDSyncHookDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: tdsync-hook-dag
  namespace: default
spec:
  entrypoint: main
  onExit: cleanup
  templateDefaults:
    synchronization:
      semaphores:
        - configMapKeyRef:
            name: my-config
            key: workflow
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: work
        hooks:
          exit:
            template: hook
      - name: b
        depends: a
        template: work
      - name: c
        depends: b
        template: work
  - name: work
    container: {image: alpine, command: [echo]}
  - name: hook
    container: {image: alpine, command: [echo]}
  - name: cleanup
    container: {image: alpine, command: [echo]}
`)
	cancel, controller := newController(ctx)
	defer cancel()
	var err error
	controller.syncManager, err = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(string) {}, workflowExistenceFunc, false)
	require.NoError(t, err)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for i := 0; i < 12 && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(woc))
}

// C51: `argo stop` while a non-target DAG task with an HTTP exit hook runs.
// Operate skips task-set reconciliation while shutting down, so the exit
// driver must hand the hook to the agent itself, as the workflow's onExit
// does; otherwise the hook stays Pending and the workflow Running.
func TestRegressionR4_C51_StopCtrTaskHTTPExitHook(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: stop-hook
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        hooks:
          exit:
            template: http
      - name: B
        depends: A
        template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
  - name: http
    http:
      url: http://example.com
`)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	// setPods gives every workflow pod phase and keeps the agent pod Running.
	setPods := func(woc *wfOperationCtx, phase apiv1.PodPhase) {
		agent := woc.getAgentPodName()
		r4SetPodsPhase(t, ctx, woc, apiv1.PodRunning, func(pod *apiv1.Pod) bool { return pod.Name == agent })
		r4SetPodsPhase(t, ctx, woc, phase, func(pod *apiv1.Pod) bool { return pod.Name != agent })
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	setPods(woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	woc.wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	for range 8 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		setPods(woc, apiv1.PodFailed)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		// The agent completes whatever it was handed.
		results := map[string]wfv1.NodeResult{}
		if ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Get(ctx, woc.wf.Name, metav1.GetOptions{}); err == nil {
			for id := range ts.Spec.Tasks {
				if n, err := woc.wf.Status.Nodes.Get(id); err == nil && !n.Fulfilled() {
					results[id] = wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{}}
				}
			}
		}
		r4ReportTaskSet(ctx, t, woc, results)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase, "stopped workflow must finish; unfulfilled: %v", r4Unfulfilled(woc))
}

// r4HookRun validates manifest, then reconciles it from its stored status,
// deciding each pod's phase with decide between reconciles, until it
// completes or rounds run out, and reconciles twice more so that a hook that
// fires late still shows.
func r4HookRun(t *testing.T, manifest string, decide func(*wfv1.NodeStatus) apiv1.PodPhase, rounds int) (context.Context, *wfOperationCtx) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	woc := r4Operate(t, ctx, controller, wf)
	extra := 2
	for i := 0; i < rounds && extra > 0; i++ {
		if woc.wf.Status.Phase.Completed() {
			extra--
		}
		setPodPhases(ctx, woc, decide)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	dumpNodes(t, "final", woc.wf)
	return ctx, woc
}

// r4HookInput is the value of the named node's first input parameter, or ""
// when the node or the parameter does not exist.
func r4HookInput(woc *wfOperationCtx, nodeName string) string {
	n, err := woc.wf.GetNodeByName(nodeName)
	if err != nil || n.Inputs == nil || len(n.Inputs.Parameters) == 0 || n.Inputs.Parameters[0].Value == nil {
		return ""
	}
	return n.Inputs.Parameters[0].Value.String()
}

// r4HookNodes lists the workflow's hook nodes (lifecycle and exit), sorted.
func r4HookNodes(woc *wfOperationCtx) []string {
	var out []string
	for _, n := range woc.wf.Status.Nodes {
		if n.NodeFlag != nil && n.NodeFlag.Hooked {
			out = append(out, n.Name)
		}
	}
	sort.Strings(out)
	return out
}

// r4FailStepA fails the pod of the step or task named a.
func r4FailStepA(n *wfv1.NodeStatus) apiv1.PodPhase {
	if strings.HasSuffix(n.Name, ".a") {
		return apiv1.PodFailed
	}
	return apiv1.PodSucceeded
}

// C15 (v1x4): an expanded step's exit hook is its item's, with {{item}}
// substituted, as executeStepGroup drove it.
func TestRegressionR4_C15_ExpandedStepExitHookItemArg(t *testing.T) {
	_, woc := r4HookRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: items-exit
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: echo
        withItems: [i1, i2]
        hooks:
          exit:
            template: hook
            arguments:
              parameters:
              - name: v
                value: "{{item}}"
  - name: echo
    container: {image: alpine, command: [echo]}
  - name: hook
    inputs: {parameters: [{name: v}]}
    container: {image: alpine, command: [echo, "{{inputs.parameters.v}}"]}
`, allSucceed, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, "i1", r4HookInput(woc, "items-exit[0].a(0:i1).onExit"))
	assert.Equal(t, "i2", r4HookInput(woc, "items-exit[0].a(1:i2).onExit"))
}

// C15 (v1x4): an exit-hook expression that selects an item runs the hook for
// that item only.
func TestRegressionR4_C15_ExpandedStepExitHookItemExpression(t *testing.T) {
	_, woc := r4HookRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: items-expr
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: echo
        withItems: [i1, i2]
        hooks:
          exit:
            expression: "'{{item}}' == 'i1'"
            template: echo
  - name: echo
    container: {image: alpine, command: [echo]}
`, allSucceed, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, []string{"items-expr[0].a(0:i1).onExit"}, r4HookNodes(woc))
}

// C48 (v1x5): an expanded step's lifecycle hook runs once per item, on each
// item node, as executeStepGroup ran it.
func TestRegressionR4_C48_StepsExpandedRunningHookPerItem(t *testing.T) {
	ctx, woc := r4HookRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: v1x5a
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: echo
        withItems: [i1, i2, i3]
        hooks:
          running:
            expression: workflow.status == "Running"
            template: hook
  - name: echo
    container: {image: alpine, command: [echo]}
  - name: hook
    container: {image: alpine, command: [echo, hook]}
`, allSucceed, 12)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	hooks := []string{"v1x5a[0].a(0:i1).hooks.running", "v1x5a[0].a(1:i2).hooks.running", "v1x5a[0].a(2:i3).hooks.running"}
	assert.Equal(t, hooks, r4HookNodes(woc))
	var hookPods []string
	for _, n := range r4PodNodeNames(ctx, t, woc) {
		if strings.Contains(n, ".hooks.") {
			hookPods = append(hookPods, n)
		}
	}
	assert.Equal(t, hooks, hookPods, "one hook pod per item")
	for _, hook := range hooks {
		assert.Equal(t, []string{strings.TrimSuffix(hook, ".hooks.running")}, r4Parents(woc, hook), "hook %s hangs off its item", hook)
	}
}

const r4C46Templates = `
  - name: ok
    container:
      image: alpine
      command: [sh, -c, "exit 0"]
  - name: notify
    inputs:
      parameters:
      - name: msg
        value: "none"
    container: {image: alpine, command: [echo, "{{inputs.parameters.msg}}"]}
`

// C46 (v3x4): a step's lifecycle hook sees the status of an earlier step in
// its own group, as executeStepGroup's group scope gave it.
func TestRegressionR4_C46_LifecycleExprSibling(t *testing.T) {
	ctx, woc := r4HookRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: lhs
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: ok
      - name: B
        template: ok
        hooks:
          afterA:
            expression: steps.A.status == "Succeeded"
            template: notify
`+r4C46Templates, allSucceed, 8)
	assert.Contains(t, r4PodNodeNames(ctx, t, woc), "lhs[0].B.hooks.afterA")
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, "lhs[0].B"))
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// C46 (v3x4): a step's exit hook argument can name an earlier step in its
// own group.
func TestRegressionR4_C46_ExitArgSibling(t *testing.T) {
	_, woc := r4HookRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: eas
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: ok
      - name: B
        template: ok
        hooks:
          exit:
            template: notify
            arguments:
              parameters:
              - name: msg
                value: "{{steps.A.status}}"
`+r4C46Templates, allSucceed, 8)
	assert.Equal(t, "Succeeded", r4HookInput(woc, "eas[0].B.onExit"))
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// C47 (v1x7): a step skipped by its when clause never ran, so its hooks do
// not run.
func TestRegressionR4_C47_StepsWhenSkippedStepHookNotRun(t *testing.T) {
	ctx, woc := r4HookRun(t, `
metadata:
  name: v1x7-skip
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: run
        when: "false"
        hooks:
          notify:
            expression: workflow.status == "Running"
            template: run
    - - name: c
        template: run
  - name: run
    container:
      image: busybox
      command: [echo]
`, allSucceed, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Empty(t, r4HookNodes(woc), "no hook node for a when-skipped step")
	assert.Equal(t, []string{"v1x7-skip[1].c"}, r4PodNodeNames(ctx, t, woc), "only c runs")
}

// C47 (v1x7): a step after a failed group never ran, so its hook, which
// reads the failed step's missing output, is not evaluated.
func TestRegressionR4_C47_StepsHookOnPrevOutputAfterFailure(t *testing.T) {
	ctx, woc := r4HookRun(t, `
metadata:
  name: v1x7-prevout
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: produce
    - - name: b
        template: run
        hooks:
          notify:
            expression: steps.a.outputs.parameters.p == "go"
            template: run
  - name: produce
    container:
      image: busybox
      command: [echo]
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
  - name: run
    container:
      image: busybox
      command: [echo]
`, r4FailStepA, 8)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.NotContains(t, woc.wf.Status.Message, "unable to evaluate expression")
	assert.Empty(t, r4HookNodes(woc))
	assert.Equal(t, []string{"v1x7-prevout[0].a"}, r4PodNodeNames(ctx, t, woc))
}

// C47 / lead 4 (lead4-1): a true-expression hook on a step after a failed
// group, whose argument names the failed step's result, fires nothing and
// leaves the workflow Failed, not Error.
func TestRegressionR4_C47_TrueHookUnresolvedArgOnOmitted(t *testing.T) {
	_, woc := r4HookRun(t, `
metadata:
  name: lead4-true
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: gen
    - - name: b
        template: run
        hooks:
          running:
            expression: "true"
            template: notify
            arguments:
              parameters:
              - name: message
                value: "{{steps.a.outputs.result}}"
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
`, r4FailStepA, 8)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.NotContains(t, woc.wf.Status.Message, "failed to resolve")
	assert.Empty(t, r4HookNodes(woc))
}

// Lead 11: a TaskGroup is completed only once its items' exit hooks have
// finished, as executeDAGTask returned before marking the group while an
// item's onExit was unfinished. The branch recorded the group Succeeded in
// the reconcile that created the item hooks.
func TestRegressionR4_Lead11_GroupWaitsForItemExitHooks(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: lead11
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: work
        withItems: [p, q]
        hooks:
          exit:
            template: notify
      - name: b
        depends: a
        template: work
  - name: work
    container: {image: alpine, command: [echo]}
  - name: notify
    container: {image: alpine, command: [echo, hook]}
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = r4Operate(t, ctx, controller, woc.wf)
	hook, err := woc.wf.GetNodeByName("lead11.a(0:p).onExit")
	require.NoError(t, err, "the first item's exit hook exists")
	require.False(t, hook.Fulfilled())
	assert.Equal(t, wfv1.NodeRunning, r4NodePhase(woc, "lead11.a"), "the group waits for its items' exit hooks")
	assert.Empty(t, r4NodePhase(woc, "lead11.b"))

	woc = r4DriveToEnd(t, ctx, controller, woc, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, "lead11.a"))
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, "lead11.a(0:p).onExit"))
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, "lead11.a(1:q).onExit"))
}

// Review Focus 1 (upgrade): main ran a DAG task's lifecycle hooks on its
// TaskGroup node. A workflow it started can hold such a hook node, and when
// that node is not a pod (here a suspend with a duration) it only advances
// when re-entered. The group, its dependant and the workflow wait for it.
func TestRegressionR4_Lead11_UpgradedGroupHookReentered(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: upgroup
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: work
        withItems: [p, q]
        hooks:
          running:
            expression: "false"
            template: wait
      - name: b
        depends: a
        template: work
  - name: work
    container: {image: alpine, command: [echo]}
  - name: wait
    suspend: {duration: "1s"}
`)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)

	// The hook node main created under the TaskGroup, running since long
	// before the upgrade.
	stored, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	group, err := stored.GetNodeByName("upgroup.a")
	require.NoError(t, err)
	root, err := stored.GetNodeByName("upgroup")
	require.NoError(t, err)
	hookName := "upgroup.a.hooks.running"
	hookID := stored.NodeID(hookName)
	stored.Status.Nodes[hookID] = wfv1.NodeStatus{
		ID:            hookID,
		Name:          hookName,
		DisplayName:   "a.hooks.running",
		Type:          wfv1.NodeTypeSuspend,
		TemplateName:  "wait",
		TemplateScope: group.TemplateScope,
		BoundaryID:    root.ID,
		Phase:         wfv1.NodeRunning,
		StartedAt:     metav1.NewTime(time.Now().Add(-time.Hour)),
		NodeFlag:      &wfv1.NodeFlag{Hooked: true},
	}
	group.Children = append(group.Children, hookID)
	stored.Status.Nodes[group.ID] = *group
	stored, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)
	woc.wf = stored

	woc = r4DriveToEnd(t, ctx, controller, woc, 8)
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, hookName), "unfulfilled: %v", r4Unfulfilled(woc))
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, "upgroup.b"))
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// P18, decided behaviour (fails at base, which passed the DAG item exit hook
// the literal {{item}}): an expanded DAG task's exit hook is each item's own,
// with {{item}} substituted, as for Steps.
func TestRegressionR4_P18_ExpandedDAGTaskExitHookItemArg(t *testing.T) {
	_, woc := r4HookRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-items-exit
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: echo
        withItems: [i1, i2]
        hooks:
          exit:
            template: hook
            arguments:
              parameters:
              - name: v
                value: "{{item}}"
  - name: echo
    container: {image: alpine, command: [echo]}
  - name: hook
    inputs: {parameters: [{name: v}]}
    container: {image: alpine, command: [echo, "{{inputs.parameters.v}}"]}
`, allSucceed, 8)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, "i1", r4HookInput(woc, "dag-items-exit.a(0:i1).onExit"))
	assert.Equal(t, "i2", r4HookInput(woc, "dag-items-exit.a(1:i2).onExit"))
	assert.Equal(t, []string{"dag-items-exit.a(0:i1).onExit", "dag-items-exit.a(1:i2).onExit"}, r4HookNodes(woc))
}

// r4HookDenied rejects the creation of every hook pod whose node name
// contains ".onExit", as an admission webhook denying the pod would.
func r4HookDenied(controller *WorkflowController) {
	r4RejectPodCreate(controller, func(pod *apiv1.Pod) bool {
		return strings.Contains(pod.Annotations[common.AnnotationKeyNodeName], ".onExit")
	}, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, "hook", fmt.Errorf("admission webhook denied the request")))
}

// r4C33Run starts manifest with every exit hook pod denied, then succeeds
// every pod and reconciles until the workflow completes (at most rounds),
// then three more times.
func r4C33Run(t *testing.T, manifest string, rounds int) (context.Context, *r4Run) {
	t.Helper()
	ctx, r := r4Start(t, manifest)
	r4HookDenied(r.controller)
	for i := 0; i < rounds+3; i++ {
		setPodPhases(ctx, r.woc, allSucceed)
		r.op(ctx)
	}
	return ctx, r
}

const r4C33Steps = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c33-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: run
        hooks:
          exit:
            template: cleanup
    - - name: b
        template: run
  - name: run
    container: {image: busybox, command: [echo, hi]}
  - name: cleanup
    container: {image: busybox, command: [echo, cleanup]}
`

// C33 (r1x58 _StepsHookDenied): a step's exit hook pod is denied after the
// step Succeeded. As at base, the Steps template and the workflow end Error,
// and the next group's step b is never created.
func TestRegressionR4_C33_StepsHookDenied(t *testing.T) {
	ctx, r := r4C33Run(t, r4C33Steps, 6)
	assert.Equal(t, wfv1.NodeError, r4NodePhase(r.woc, "c33-steps[0].a.onExit"))
	assert.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
	assert.Empty(t, r4Unfulfilled(r.woc))
	assert.Equal(t, []string{"c33-steps[0].a"}, r4PodNodeNames(ctx, t, r.woc), "b's pod was created although the workflow errored in group [0]")
}

// C33 (r1x58 _StepsHookExprErrorNextGroup): the #14031 fixture with a second
// group. The exit hook's expression fails after a Succeeded: the workflow
// ends Error and b never starts.
func TestRegressionR4_C33_StepsHookExprErrorNextGroup(t *testing.T) {
	woc, pods := r4RunDecided(t, strings.Replace(r4C33Steps, "            template: cleanup", "            expression: steps[\"a\"].outputs !\n            template: cleanup", 1), 6, allSucceed)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase, woc.wf.Status.Message)
	assert.Empty(t, r4Unfulfilled(woc))
	assert.Equal(t, []string{"c33-steps[0].a"}, pods)
}

const r4C33DAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c33-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: run
        hooks:
          exit:
            template: cleanup
      - {name: b, template: run, depends: a}
  - name: run
    container: {image: busybox, command: [echo, hi]}
  - name: cleanup
    container: {image: busybox, command: [echo, cleanup]}
`

// C33, decided deviation (base DAG ignored the errored hook, ran b and
// Succeeded; fails at base by design): a DAG task's exit hook pod is denied
// after the task Succeeded. The DAG ends Error, as Steps does, and nothing
// more is dispatched: b gets neither a node nor a pod.
func TestRegressionR4_C33_DAGHookDeniedIsErrorWithoutStrayPod(t *testing.T) {
	ctx, r := r4C33Run(t, r4C33DAG, 6)
	assert.Equal(t, wfv1.NodeError, r4NodePhase(r.woc, "c33-dag.a.onExit"))
	assert.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
	assert.Contains(t, r.woc.wf.Status.Message, "admission webhook denied the request")
	assert.Empty(t, r4Unfulfilled(r.woc))
	assert.Equal(t, wfv1.NodePhase(""), r4NodePhase(r.woc, "c33-dag.b"), "b has a node")
	assert.Equal(t, []string{"c33-dag.a"}, r4PodNodeNames(ctx, t, r.woc))
}

// C33 with P20, decided deviation (base DAG ignored the errored hook and
// Succeeded; fails at base by design), from v1x58 H: a's exit hook pod is
// denied while its sibling b still runs. The DAG waits for b, keeps b's pod
// and dispatches nothing new (c, which depends on a, gets no node) on every
// reconcile, then ends Error once b has finished.
func TestRegressionR4_C33_DAGHookErrorWaitsForRunningSibling(t *testing.T) {
	ctx, r := r4Start(t, strings.Replace(r4C33DAG, "depends: a}", "}\n      - {name: c, template: run, depends: a}", 1))
	r4HookDenied(r.controller)
	running := func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.HasSuffix(n.Name, ".b") {
			return apiv1.PodRunning
		}
		return apiv1.PodSucceeded
	}
	for range 4 {
		setPodPhases(ctx, r.woc, running)
		r.op(ctx)
		require.Equal(t, wfv1.WorkflowRunning, r.woc.wf.Status.Phase, "the workflow ended while b runs: %s", r.woc.wf.Status.Message)
		assert.Equal(t, wfv1.NodeRunning, r4NodePhase(r.woc, "c33-dag"))
		assert.Equal(t, wfv1.NodePhase(""), r4NodePhase(r.woc, "c33-dag.c"), "c was dispatched after a's hook errored")
	}
	assert.Equal(t, wfv1.NodeError, r4NodePhase(r.woc, "c33-dag.a.onExit"))
	for range 3 {
		setPodPhases(ctx, r.woc, allSucceed)
		r.op(ctx)
	}
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(r.woc, "c33-dag.b"))
	assert.Equal(t, wfv1.NodePhase(""), r4NodePhase(r.woc, "c33-dag.c"))
	assert.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
	assert.Contains(t, r.woc.wf.Status.Message, "admission webhook denied the request")
	assert.Equal(t, []string{"c33-dag.a", "c33-dag.b"}, r4PodNodeNames(ctx, t, r.woc))
}

// C33, decided deviation (base DAG ignored the errored hook and Succeeded;
// fails at base by design): continueOn.error does not cover an exit hook
// that errors after its task Succeeded. The DAG ends Error rather than
// Succeeded, and does not hang.
func TestRegressionR4_C33_DAGHookErrorWithContinueOn(t *testing.T) {
	manifest := strings.Replace(r4C33DAG, "depends: a}", "dependencies: [a]}", 1)
	manifest = strings.Replace(manifest, "            template: cleanup", "            expression: tasks[\"a\"].outputs !\n            template: cleanup\n        continueOn: {error: true}", 1)
	woc, _ := r4RunDecided(t, manifest, 6, allSucceed)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase, woc.wf.Status.Message)
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "c33-dag.a.onExit"), "the hook error is recorded on the hook node")
	assert.Equal(t, wfv1.NodePhase(""), r4NodePhase(woc, "c33-dag.b"))
	assert.Empty(t, r4Unfulfilled(woc))
}

// C33, decided deviation (base DAG ignored the errored hook, so the nested
// DAG Succeeded and was memoized; fails at base by design): a nested DAG
// whose task's exit hook errors ends Error, gets no outputs and is not
// written to its memoization cache.
func TestRegressionR4_C33_HookErroredBoundaryNotMemoized(t *testing.T) {
	woc, _ := r4RunDecided(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c33-memo
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - {name: inner, template: inner}
  - name: inner
    memoize:
      key: c33
      cache:
        configMap: {name: r4-c33-cache}
    outputs:
      parameters:
      - name: s
        valueFrom: {parameter: "{{tasks.a.status}}"}
    dag:
      tasks:
      - name: a
        template: run
        hooks:
          exit:
            expression: tasks["a"].outputs !
            template: run
  - name: run
    container: {image: busybox, command: [echo, hi]}
`, 6, allSucceed)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase, woc.wf.Status.Message)
	inner, err := woc.wf.GetNodeByName("c33-memo.inner")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, inner.Phase)
	assert.Nil(t, inner.Outputs, "a hook-errored boundary gets no outputs")
	cm, err := woc.controller.kubeclientset.CoreV1().ConfigMaps("default").Get(logging.TestContext(t.Context()), "r4-c33-cache", metav1.GetOptions{})
	if err == nil {
		assert.Empty(t, cm.Data, "a hook-errored boundary is not memoized")
	} else {
		assert.True(t, apierr.IsNotFound(err), err)
	}
}

// v1x58 D and E, controls: a hook pod that runs and Fails is still ignored,
// in DAG and Steps, as at base.
func TestRegressionR4_C33_FailedHookPodIgnored(t *testing.T) {
	failHook := func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.HasSuffix(n.Name, ".onExit") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	}
	for _, manifest := range []string{r4C33DAG, r4C33Steps} {
		woc, _ := r4RunDecided(t, manifest, 8, failHook)
		assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
		assert.Empty(t, r4Unfulfilled(woc))
	}
}

// r4C41Run runs a template (steps or dag) whose task a has an exit or
// running hook with pendingTimeout: 1s, keeps the hook's pod Pending past
// it, then lets a finish (as v3x7Run does), deletes the timed-out pod as the
// pod controller would, and reconciles to the end.
func r4C41Run(t *testing.T, kind, hookKind string) *wfv1.Workflow {
	t.Helper()
	ref, body := "steps", "    steps:\n    - - name: a\n        template: c\n        hooks:\n"
	if kind == "dag" {
		ref, body = "tasks", "    dag:\n      tasks:\n      - name: a\n        template: c\n        hooks:\n"
	}
	hookSuffix, mainPhase := ".onExit", apiv1.PodSucceeded
	if hookKind == "running" {
		body += "          running:\n            expression: " + ref + ".a.status == \"Running\"\n            template: hook\n"
		hookSuffix, mainPhase = ".hooks.running", apiv1.PodRunning
	} else {
		body += "          exit:\n            template: hook\n"
	}
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c41-`+kind+`-`+hookKind+`
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
`+body+`  - name: c
    container: {image: busybox, command: [echo]}
  - name: hook
    pendingTimeout: 1s
    container: {image: busybox, command: [echo]}
`)
	makePodsPhase(ctx, r.woc, mainPhase)
	r.op(ctx)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodPending, r4PodForNodePrefix(r.woc.wf.Name+"[0].a"+hookSuffix))
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodPending, r4PodForNodePrefix(r.woc.wf.Name+".a"+hookSuffix))
	r.op(ctx)
	time.Sleep(1500 * time.Millisecond)
	r.op(ctx)
	if hookKind == "running" {
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, func(pod *apiv1.Pod) bool {
			return strings.HasSuffix(pod.Annotations[common.AnnotationKeyNodeName], ".a")
		})
	}
	for range 3 {
		r.op(ctx)
	}
	for _, n := range r.woc.wf.Status.Nodes {
		if n.Message == "timeout" && n.Type == wfv1.NodeTypePod {
			pods, err := listPods(ctx, r.woc)
			require.NoError(t, err)
			for _, pod := range pods.Items {
				if pod.Annotations[common.AnnotationKeyNodeName] == n.Name {
					r4DeletePod(ctx, t, r.woc, n.Name)
				}
			}
		}
	}
	for range 5 {
		r.op(ctx)
	}
	dumpNodes(t, "final", r.woc.wf)
	return r.woc.wf
}

// C41 (v3x7): a step's exit hook times out (pendingTimeout) after the step
// Succeeded. As at base, the workflow ends Error "timeout".
func TestRegressionR4_C41_StepsExitHookPendingTimeout(t *testing.T) {
	wf := r4C41Run(t, "steps", "exit")
	assert.Equal(t, wfv1.WorkflowError, wf.Status.Phase)
	assert.Equal(t, "timeout", wf.Status.Message)
}

// C41 (v3x7): a step's running hook times out while the step runs. As at
// base, the workflow ends Failed.
func TestRegressionR4_C41_StepsLifecycleHookPendingTimeout(t *testing.T) {
	wf := r4C41Run(t, "steps", "running")
	assert.Equal(t, wfv1.WorkflowFailed, wf.Status.Phase, wf.Status.Message)
}

// C41 (v3x7): a DAG task's running hook times out while the task runs. As at
// base, the workflow ends Error.
func TestRegressionR4_C41_DAGLifecycleHookPendingTimeout(t *testing.T) {
	wf := r4C41Run(t, "dag", "running")
	assert.Equal(t, wfv1.WorkflowError, wf.Status.Phase, wf.Status.Message)
}

// C41 with P21, decided deviation (base DAG ignored a timed-out exit hook and
// Succeeded; fails at base by design): a DAG task's exit hook times out
// after the task Succeeded. The DAG ends Error "timeout", as Steps does.
func TestRegressionR4_C41_DAGExitHookPendingTimeoutIsError(t *testing.T) {
	wf := r4C41Run(t, "dag", "exit")
	assert.Equal(t, wfv1.WorkflowError, wf.Status.Phase)
	assert.Equal(t, "timeout", wf.Status.Message)
}

// C33 with Review Focus 4: `argo retry` of a DAG that a denied exit hook
// ended Error removes the Error hook node, so the retried DAG runs the hook
// again and, once it is admitted, runs b and Succeeds. (Base DAG Succeeded
// in the first place, so there was nothing to retry.)
func TestRegressionR4_C33_DAGHookErrorRetried(t *testing.T) {
	ctx, r := r4Start(t, r4C33DAG)
	denied := 0
	r4RejectPodCreate(r.controller, func(pod *apiv1.Pod) bool {
		if strings.Contains(pod.Annotations[common.AnnotationKeyNodeName], ".onExit") && denied == 0 {
			denied++
			return true
		}
		return false
	}, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, "hook", fmt.Errorf("admission webhook denied the request")))
	for i := 0; i < 6 && !r.woc.wf.Status.Phase.Completed(); i++ {
		setPodPhases(ctx, r.woc, allSucceed)
		r.op(ctx)
	}
	require.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)

	r4RetryStored(t, ctx, r.controller, r.woc.wf)
	r.woc = r4Operate(t, ctx, r.controller, r.woc.wf)
	for i := 0; i < 8 && !r.woc.wf.Status.Phase.Completed(); i++ {
		setPodPhases(ctx, r.woc, allSucceed)
		r.op(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(r.woc, "c33-dag.a.onExit"))
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(r.woc, "c33-dag.b"))
}

// C33, decided deviation (base marked a Error, continueOn let b run and the
// workflow Succeeded; fails at base by design): a's running hook pod is
// denied while a runs. The hook error ends the DAG Error, and b, made ready
// in that same reconcile by a's Error under continueOn, gets no node.
func TestRegressionR4_C33_DAGLifecycleHookErrorStopsDispatchInSamePass(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c33-lc
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: run
        continueOn: {error: true}
        hooks:
          running:
            expression: tasks.a.status == "Running"
            template: run
      - {name: b, template: run, dependencies: [a]}
  - name: run
    container: {image: busybox, command: [echo, hi]}
`)
	r4RejectPodCreate(r.controller, func(pod *apiv1.Pod) bool {
		return strings.Contains(pod.Annotations[common.AnnotationKeyNodeName], ".hooks.running")
	}, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, "hook", fmt.Errorf("admission webhook denied the request")))
	makePodsPhase(ctx, r.woc, apiv1.PodRunning)
	r.op(ctx)
	assert.Equal(t, wfv1.NodeError, r4NodePhase(r.woc, "c33-lc.a.hooks.running"))
	assert.Equal(t, wfv1.NodePhase(""), r4NodePhase(r.woc, "c33-lc.b"), "b was dispatched in the reconcile a's hook errored")
	for range 3 {
		r.op(ctx)
	}
	assert.Equal(t, wfv1.NodePhase(""), r4NodePhase(r.woc, "c33-lc.b"))
	assert.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
	assert.Equal(t, []string{"c33-lc.a"}, r4PodNodeNames(ctx, t, r.woc))
}

const r4C92WorkflowHookExprErr = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c92-wfhook
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: threshold
      value: "abc"
  hooks:
    notify:
      expression: workflow.status == "Succeeded" && workflow.parameters.threshold > 5
      template: notify
  templates:
  - name: main
    container: {image: busybox, command: [echo, hi]}
  - name: notify
    container: {image: busybox, command: [echo, notify]}
`

// TestRegressionR4_C92_WorkflowHookExprErrorOnHookNode (C92, P17) is a
// decided deviation: it encodes the chosen fix, not main's behaviour, so it
// is expected to fail both at base (4389bbf96, where markNodeError flips
// the entry node Succeeded -> Error with the hook error instead) and on
// this branch before the fix (where the strict node-phase state machine
// refuses that transition and the error is dropped entirely, logging
// "refusing invalid node phase transition" every reconcile). See C92 in
// pr-16290-round4-regressions.md and P17's answer.
//
// The workflow-level hook (spec.hooks) here has an expression that only
// errors once the workflow has Succeeded: comparing the string parameter
// "abc" against 5 is a runtime type error, mirroring lead9's
// lead9ExprSteps probe. The fix records that error on its own Error hook
// node "<wf>.hooks.notify" instead, leaving the entry node and the
// workflow itself Succeeded.
func TestRegressionR4_C92_WorkflowHookExprErrorOnHookNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C92WorkflowHookExprErr)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := r4Operate(t, ctx, controller, wf)
	for range 3 {
		makePodsPhase(ctx, woc, apiv1.PodRunning)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	for i := 0; i < 6 && !woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	require.True(t, woc.wf.Status.Phase.Completed(), "workflow did not complete: phase=%s", woc.wf.Status.Phase)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, "c92-wfhook"), "entry node")
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "c92-wfhook.hooks.notify"), "hook node")
	hookNode, err := woc.wf.GetNodeByName("c92-wfhook.hooks.notify")
	require.NoError(t, err)
	assert.NotEmpty(t, hookNode.Message, "hook node message")
}

// r4C2Workflow is the C2 shape: task A (a DAG task or a step, per kind)
// runs a failing template whose retryStrategy (retry, or the workflow's
// templateDefaults) is only known once the template is processed. The
// strategy has a one-hour backoff, so A's second attempt never starts here.
func r4C2Workflow(kind, defaults, retry string) string {
	body := "dag:\n      tasks:\n      - name: A"
	if kind == "steps" {
		body = "steps:\n    - - name: A"
	}
	return fmt.Sprintf(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c2-%s
  namespace: default
spec:
  entrypoint: main
%s  templates:
  - name: main
    %s
        template: fail
        arguments:
          parameters:
          - name: p
            value: "2"
  - name: fail
    inputs:
      parameters:
      - name: p
%s    container:
      image: alpine
      command: [sh, -c, exit 1]
`, kind, defaults, body, retry)
}

// r4C2BackingOff fails A's first attempt as a kubelet would (a terminated
// container with a finish time, so the backoff window is open), reconciles
// a few more times, and checks that A is still backing off and nothing
// around it has finished: the second attempt is still to come.
func r4C2BackingOff(t *testing.T, manifest, boundary string) {
	ctx, r := r4Start(t, manifest)
	require.Equal(t, wfv1.WorkflowRunning, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodFailed, func(*apiv1.Pod) bool { return true }, func(pod *apiv1.Pod, _ *wfOperationCtx) {
		now := metav1.Now()
		pod.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: common.MainContainerName, State: apiv1.ContainerState{
			Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1, StartedAt: metav1.NewTime(now.Add(-2 * time.Second)), FinishedAt: now},
		}}}
	})
	for range 3 {
		r.op(ctx)
	}
	a := r.woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, a)
	assert.Equal(t, wfv1.NodeRunning, a.Phase, "A is backing off")
	assert.True(t, strings.HasPrefix(a.Message, "Backoff for"), "A: %q", a.Message)
	assert.Len(t, podNames(ctx, r.woc), 1, "no second attempt yet")
	assert.Equal(t, wfv1.NodeRunning, r4NodePhase(r.woc, boundary), "the boundary waits for A's retries")
	assert.Equal(t, wfv1.WorkflowRunning, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
}

// C2: A's retry limit comes from its inputs. Ports TestProbe_v1x0_ParamLimitDAG.
func TestRegressionR4_C2_ParamLimitDAG(t *testing.T) {
	r4C2BackingOff(t, r4C2Workflow("dag", "", `    retryStrategy:
      limit: "{{inputs.parameters.p}}"
      backoff: {duration: 1h}
`), "c2-dag")
}

// C2 for Steps, where HEAD reported the workflow Succeeded. Ports
// TestProbe_v1x0_ParamLimitSteps.
func TestRegressionR4_C2_ParamLimitSteps(t *testing.T) {
	r4C2BackingOff(t, r4C2Workflow("steps", "", `    retryStrategy:
      limit: "{{inputs.parameters.p}}"
      backoff: {duration: 1h}
`), "c2-steps")
}

// C2: A's retryStrategy comes from spec.templateDefaults, which also retries
// the DAG itself; the DAG's first attempt must keep running. Ports
// TestProbe_v1x0_TemplateDefaultsDAG.
func TestRegressionR4_C2_TemplateDefaultsDAG(t *testing.T) {
	r4C2BackingOff(t, r4C2Workflow("dag", `  templateDefaults:
    retryStrategy:
      limit: "2"
      backoff: {duration: 1h}
`, ""), "c2-dag(0)")
}

// C2: A's retry expression refers to its inputs. Ports
// TestProbe_v1x0_ParamExprDAG.
func TestRegressionR4_C2_ParamExprDAG(t *testing.T) {
	r4C2BackingOff(t, r4C2Workflow("dag", "", `    retryStrategy:
      limit: "2"
      expression: "'{{inputs.parameters.p}}' == '2'"
      backoff: {duration: 1h}
`), "c2-dag")
}

// r4C27Workflow is a memoized template with a retryStrategy, run as task
// or step a: a cache miss gives a Retry node whose first attempt is Pending.
func r4C27Workflow(kind string) string {
	body := "dag:\n      tasks:\n      - {name: a, template: echo}"
	if kind == "steps" {
		body = "steps:\n    - - {name: a, template: echo}"
	}
	return fmt.Sprintf(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c27-%s
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    %s
  - name: echo
    retryStrategy: {limit: 1}
    memoize:
      key: my-key
      cache:
        configMap: {name: my-cache}
    container:
      image: alpine
      command: [echo, hi]
`, kind, body)
}

// r4C27Succeeds lets the recreated pod succeed and checks the workflow ends
// Succeeded.
func r4C27Succeeds(ctx context.Context, t *testing.T, r *r4Run) {
	t.Helper()
	require.Len(t, podNames(ctx, r.woc), 1, "the Pending attempt's pod is created again")
	for i := 0; i < 4 && !r.woc.wf.Status.Phase.Completed(); i++ {
		makePodsPhase(ctx, r.woc, apiv1.PodSucceeded)
		r.op(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, r.woc.wf.Status.Message)
}

// C27: the Pending pod of a memoized retried DAG task is deleted; the next
// reconciles must create it again, as base did. Ports
// TestProbe_v3x12_DagMemoRetryPendingPodDeleted.
func TestRegressionR4_C27_MemoRetryPendingPodDeleted(t *testing.T) {
	ctx, r := r4Start(t, r4C27Workflow("dag"))
	r.op(ctx)
	require.Len(t, podNames(ctx, r.woc), 1)
	deletePods(ctx, r.woc)
	for range 5 {
		r4BackdateNodes(t, ctx, r.controller, r.woc.wf, time.Hour)
		r.op(ctx)
	}
	r4C27Succeeds(ctx, t, r)
}

// C27: the first pod creation of a memoized retried step is refused by a
// ResourceQuota (a transient error that leaves the attempt Pending); once
// the quota frees up the pod must be created. Ports
// TestProbe_v3x12_StepsMemoRetryQuotaExceeded.
func TestRegressionR4_C27_StepsMemoRetryQuotaExceeded(t *testing.T) {
	quotaFull := true
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(r4C27Workflow("steps"))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	r4RejectPodCreate(controller, func(*apiv1.Pod) bool { return quotaFull },
		apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, "x", fmt.Errorf("exceeded quota: compute-resources, requested: pods=1, used: pods=10, limited: pods=10")))
	r := &r4Run{t: t, controller: controller, woc: r4Operate(t, ctx, controller, wf)}
	require.Empty(t, podNames(ctx, r.woc), "the first pod creation is refused")
	quotaFull = false
	for range 5 {
		r.op(ctx)
	}
	r4C27Succeeds(ctx, t, r)
}

// C34: after `argo resubmit --memoized`, a retried DAG task that had
// Succeeded keeps its Succeeded Retry node over a Skipped placeholder
// attempt; it must count as Succeeded, not as a failed retry. Ports
// TestProbe_v2x1_MemoizedRetryLeafDAG.
func TestRegressionR4_C34_MemoizedRetryLeafDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c34-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - {name: A, template: run}
      - {name: C, template: run}
  - name: run
    retryStrategy: {limit: 1}
    container:
      image: busybox
`)
	cancel, controller, newWf := memoizedResubmit(ctx, t, wf, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(node.Name, ".A(") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	defer cancel()
	woc := runToCompletion(ctx, t, controller, newWf, allSucceed)
	assert.Len(t, podNames(ctx, woc), 1, "only A runs again")
	for _, n := range woc.wf.Status.Nodes {
		assert.NotEqual(t, wfv1.NodeFailed, n.Phase, "node %s", n.Name)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
}

// C34 for Steps: the memoized C of the first group must not stop the second
// group, which never ran. Ports TestProbe_v2x1_MemoizedRetryStepsNextGroup.
func TestRegressionR4_C34_MemoizedRetryStepsNextGroup(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c34-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - {name: A, template: run}
      - {name: C, template: run}
    - - {name: B, template: run}
  - name: run
    retryStrategy: {limit: 1}
    container:
      image: busybox
`)
	cancel, controller, newWf := memoizedResubmit(ctx, t, wf, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(node.Name, "].A(") {
			return apiv1.PodFailed
		}
		return apiv1.PodSucceeded
	})
	defer cancel()
	woc := runToCompletion(ctx, t, controller, newWf, allSucceed)
	assert.Len(t, podNames(ctx, woc), 2, "A and B run")
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(woc, newWf.Name+"[1].B"), "B runs once A succeeds")
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, woc.wf.Status.Message)
}

// r4C78DaemonDAG is a daemon server with a retryStrategy and a client that
// depends on it and sorts before it.
const r4C78DaemonDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: c78
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - {name: server, template: daemon}
      - {name: client, template: ok, depends: server}
  - name: daemon
    daemon: true
    retryStrategy: {limit: 2}
    container:
      image: alpine
      command: [sh, -c, sleep 9999]
  - name: ok
    container:
      image: alpine
      command: [sh, -c, exit 0]
`

// r4C78Start runs the daemon until it is Running and daemoned and the client
// has started.
func r4C78Start(t *testing.T) (context.Context, *r4Run) {
	t.Helper()
	ctx, r := r4Start(t, r4C78DaemonDAG)
	for range 2 {
		makePodsPhase(ctx, r.woc, apiv1.PodRunning)
		r.op(ctx)
	}
	attempt := r.woc.wf.Status.Nodes.FindByDisplayName("server(0)")
	require.NotNil(t, attempt)
	require.True(t, attempt.IsDaemoned())
	require.NotNil(t, r.woc.wf.Status.Nodes.FindByDisplayName("client"))
	return ctx, r
}

// C78: the Retry node of a running daemon is itself daemoned, reconcile
// after reconcile. Ports TestProbe_v1x3_DaemonRetryNodeDaemoned.
func TestRegressionR4_C78_DaemonRetryNodeDaemoned(t *testing.T) {
	ctx, r := r4C78Start(t)
	for i := range 4 {
		server := r.woc.wf.Status.Nodes.FindByDisplayName("server")
		require.NotNil(t, server)
		assert.True(t, server.IsDaemoned(), "reconcile %d: the Retry node of a running daemon is daemoned", i)
		r.op(ctx)
	}
}

// C78: `argo stop` while the daemon and client run ends the daemon's Retry
// node Succeeded, like its attempt, not Failed "Stopped with strategy".
// Ports TestProbe_v1x3_DaemonStopRetryNodePhase.
func TestRegressionR4_C78_DaemonStopRetryNodePhase(t *testing.T) {
	ctx, r := r4C78Start(t)
	stored, err := r.controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Get(ctx, "c78", metav1.GetOptions{})
	require.NoError(t, err)
	stored.Spec.Shutdown = wfv1.ShutdownStrategyStop
	_, err = r.controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)
	r.op(ctx)
	for range 3 {
		makePodsPhase(ctx, r.woc, apiv1.PodFailed)
		r.op(ctx)
	}
	server := r.woc.wf.Status.Nodes.FindByDisplayName("server")
	require.NotNil(t, server)
	attempt := r.woc.wf.Status.Nodes.FindByDisplayName("server(0)")
	require.NotNil(t, attempt)
	assert.Equal(t, attempt.Phase, server.Phase, "the Retry node takes its only attempt's phase")
	assert.Equal(t, wfv1.NodeSucceeded, server.Phase, server.Message)
}

const r4C24SuspendMutexSteps = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c24-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: approve
        template: gate
    - - name: deploy
        template: locked
  - name: gate
    synchronization:
      mutexes:
      - name: r4-c24-m
    suspend: {}
  - name: locked
    synchronization:
      mutexes:
      - name: r4-c24-m
    container:
      image: busybox
`

const r4C24SuspendMutexDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c24-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: approve
        template: gate
      - name: deploy
        template: locked
        dependencies: [approve]
  - name: gate
    synchronization:
      mutexes:
      - name: r4-c24-m
    suspend: {}
  - name: locked
    synchronization:
      mutexes:
      - name: r4-c24-m
    container:
      image: busybox
`

// r4C24ResumedSuspend resumes an approval gate holding a mutex that the next
// task needs: the gate, finished outside the controller, must release it.
func r4C24ResumedSuspend(t *testing.T, manifest string) {
	ctx, r := r4StartLocked(t, manifest)
	require.Equal(t, wfv1.NodeRunning, r.r4Phase("approve"))
	r.r4Resume(ctx)
	for range 4 {
		r.r4SetPods(ctx, nil)
	}
	deploy := r.woc.wf.Status.Nodes.FindByDisplayName("deploy")
	require.NotNil(t, deploy)
	assert.Contains(t, r4PodNodeNames(ctx, t, r.woc), deploy.Name, "deploy takes the mutex once approve was resumed")
	r.r4SetPods(ctx, map[string]apiv1.PodPhase{"deploy": apiv1.PodSucceeded})
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(r.woc))
}

// C24: a resumed suspend step releases its mutex. Ports
// TestProbe_v1x43_ResumedSuspendMutexSteps.
func TestRegressionR4_C24_ResumedSuspendMutexSteps(t *testing.T) {
	r4C24ResumedSuspend(t, r4C24SuspendMutexSteps)
}

// C24: a resumed suspend task releases its mutex. Ports
// TestProbe_v1x43_ResumedSuspendMutexDAG.
func TestRegressionR4_C24_ResumedSuspendMutexDAG(t *testing.T) {
	r4C24ResumedSuspend(t, r4C24SuspendMutexDAG)
}

// C24: an HTTP task, completed by task-set reconciliation after the Engine
// has run, releases its mutex so its dependant can take it. Ports
// TestProbe_v1x43_HTTPTaskReleasesMutex.
func TestRegressionR4_C24_HTTPTaskReleasesMutex(t *testing.T) {
	ctx, r := r4StartLocked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c24-http
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: call
        template: http
      - name: after
        template: locked
        dependencies: [call]
  - name: http
    synchronization:
      mutexes:
      - name: r4-c24-http-m
    http:
      url: "http://localhost"
  - name: locked
    synchronization:
      mutexes:
      - name: r4-c24-http-m
    container:
      image: busybox
`, defaultServiceAccount)
	call := r.woc.wf.Status.Nodes.FindByDisplayName("call")
	require.NotNil(t, call)
	r4ReportTaskSet(ctx, t, r.woc, map[string]wfv1.NodeResult{call.ID: {Phase: wfv1.NodeSucceeded}})
	for range 8 {
		r.r4SetPods(ctx, nil)
	}
	after := r.woc.wf.Status.Nodes.FindByDisplayName("after")
	require.NotNil(t, after)
	assert.Nil(t, after.SynchronizationStatus, "after holds the mutex: %s", after.Message)
	assert.Contains(t, r4PodNodeNames(ctx, t, r.woc), "r4-c24-http.after")
}

const r4C25FailFastInner = `
  - name: locked-inner
    synchronization:
      mutexes:
      - name: r4-c25-m
    failFast: true
    dag:
      tasks:
      - name: bad
        template: bad
  - name: uses-m
    synchronization:
      mutexes:
      - name: r4-c25-m
    container:
      image: busybox
      command: [sh, -c, "exit 0"]
  - name: bad
    container:
      image: busybox
      command: [sh, -c, "exit 1"]
`

// C25: a nested DAG holding a mutex that its own failFast check marks Failed
// releases it, so a sibling waiting on the mutex runs. Ports
// TestProbe_v2x8_FailFastSimpleMutexReleased.
func TestRegressionR4_C25_FailFastSimpleMutexReleased(t *testing.T) {
	ctx, r := r4StartLocked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c25-ffa
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: inner
        template: locked-inner
      - name: waiter
        template: uses-m
`+r4C25FailFastInner)
	r.r4SetPods(ctx, map[string]apiv1.PodPhase{"bad": apiv1.PodFailed})
	for range 10 {
		r.r4SetPods(ctx, nil)
	}
	assert.Equal(t, wfv1.NodeFailed, r.r4Phase("inner"))
	assert.Contains(t, r4PodNodeNames(ctx, t, r.woc), "r4-c25-ffa.waiter")
	for range 3 {
		r.r4SetPods(ctx, map[string]apiv1.PodPhase{"waiter": apiv1.PodSucceeded})
	}
	assert.Equal(t, wfv1.WorkflowFailed, r.woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(r.woc))
}

// C25: the failFast nested DAG's dependant (depends: inner.Failed) takes the
// mutex it released. Ports TestProbe_v2x8_FailFastDependantAcquiresMutex.
func TestRegressionR4_C25_FailFastDependantAcquiresMutex(t *testing.T) {
	ctx, r := r4StartLocked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c25-ffd
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: inner
        template: locked-inner
      - name: cleanup
        template: uses-m
        depends: inner.Failed
`+r4C25FailFastInner)
	r.r4SetPods(ctx, map[string]apiv1.PodPhase{"bad": apiv1.PodFailed})
	for range 6 {
		r.r4SetPods(ctx, nil)
	}
	assert.Equal(t, wfv1.NodeFailed, r.r4Phase("inner"))
	assert.Contains(t, r4PodNodeNames(ctx, t, r.woc), "r4-c25-ffd.cleanup")
}

// C25: a pod holding a mutex whose pendingTimeout expires while it is
// Pending releases the mutex, so its sibling runs and the workflow ends.
// Ports TestProbe_r2syncmemo_PendingTimeoutHolderReleasesDAG.
func TestRegressionR4_C25_PendingTimeoutHolderReleasesDAG(t *testing.T) {
	ctx, r := r4StartLocked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c25-pth
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: uses-m-timeout
      - name: b
        template: uses-m
  - name: uses-m
    synchronization:
      mutexes:
      - name: r4-c25-pt-m
    container:
      image: busybox
      command: [sh, -c, "exit 0"]
  - name: uses-m-timeout
    pendingTimeout: 30s
    synchronization:
      mutexes:
      - name: r4-c25-pt-m
    container:
      image: busybox
      command: [sh, -c, "exit 0"]
`)
	// a's pod stays Pending (unschedulable) until its pendingTimeout passes.
	r.op(ctx)
	r.r4AgeNode(ctx, "a", time.Hour)
	r.op(ctx)
	require.Equal(t, wfv1.NodeFailed, r.r4Phase("a"))
	// The pod controller deletes the timed-out pod.
	r4DeletePod(ctx, t, r.woc, "r4-c25-pth.a")
	for range 3 {
		r.op(ctx)
	}
	assert.Contains(t, r4PodNodeNames(ctx, t, r.woc), "r4-c25-pth.b")
	for range 3 {
		r.r4SetPods(ctx, nil)
		r.r4SetPods(ctx, map[string]apiv1.PodPhase{"b": apiv1.PodSucceeded})
	}
	assert.True(t, r.woc.wf.Status.Phase.Completed(), "workflow phase %s, unfulfilled: %v", r.woc.wf.Status.Phase, r4Unfulfilled(r.woc))
}

// C25: a template that takes a mutex and then fails its memoization setup
// (an invalid maxAge) releases the mutex. Ports
// TestProbe_r2syncmemo_MemoizeErrorHolderReleases.
func TestRegressionR4_C25_MemoizeErrorHolderReleases(t *testing.T) {
	ctx, r := r4StartLocked(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c25-mma
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: age
      value: "one-hour"
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: memo-m
      - name: b
        template: uses-m
  - name: uses-m
    synchronization:
      mutexes:
      - name: r4-c25-mm
    container:
      image: busybox
      command: [sh, -c, "exit 0"]
  - name: memo-m
    synchronization:
      mutexes:
      - name: r4-c25-mm
    memoize:
      key: "k"
      maxAge: "{{workflow.parameters.age}}"
      cache:
        configMap:
          name: r4-c25-cache
    outputs:
      parameters:
      - name: out
        valueFrom:
          path: /tmp/out
    container:
      image: busybox
      command: [sh, -c, "exit 0"]
`)
	for range 3 {
		r.r4SetPods(ctx, nil)
	}
	assert.Equal(t, wfv1.NodeError, r.r4Phase("a"))
	assert.Contains(t, r4PodNodeNames(ctx, t, r.woc), "r4-c25-mma.b")
}

// r4C64TDSync is a workflow whose only lock comes from
// spec.templateDefaults.synchronization (my-config/workflow, limit 2).
func r4C64TDSync(name, body string) string {
	return `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ` + name + `
  namespace: default
spec:
  entrypoint: main
  templateDefaults:
    synchronization:
      semaphores:
        - configMapKeyRef:
            name: my-config
            key: workflow
  templates:
` + body
}

// C64 (section 7): items holding a semaphore from templateDefaults release
// it, so the next item and the workflow finish. Ports
// TestProbe_r3wfdefaults_TDSyncDAGItems.
func TestRegressionR4_C64_TDSyncDAGItems(t *testing.T) {
	ctx, r := r4StartLocked(t, r4C64TDSync("r4-c64-td-dag", `
  - name: main
    dag:
      tasks:
      - name: a
        template: work
        withItems: [1, 2]
  - name: work
    container:
      image: alpine
      command: [echo]
`))
	r.woc = r4DriveToEnd(t, ctx, r.controller, r.woc, 6)
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(r.woc))
}

// C64 (section 7): the Steps form. Ports
// TestProbe_r3wfdefaults_TDSyncStepsItems.
func TestRegressionR4_C64_TDSyncStepsItems(t *testing.T) {
	ctx, r := r4StartLocked(t, r4C64TDSync("r4-c64-td-steps", `
  - name: main
    steps:
    - - name: a
        template: work
        withItems: [1, 2]
  - name: work
    container:
      image: alpine
      command: [echo]
`))
	r.woc = r4DriveToEnd(t, ctx, r.controller, r.woc, 6)
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(r.woc))
}

// C64 (section 7): two plain suspend tasks holding the templateDefaults
// semaphore, resumed in turn, release it. Ports
// TestProbe_r3wfdefaults_TDSyncSuspendPlainDAG.
func TestRegressionR4_C64_TDSyncSuspendPlainDAG(t *testing.T) {
	ctx, r := r4StartLocked(t, r4C64TDSync("r4-c64-td-suspend", `
  - name: main
    dag:
      tasks:
      - name: s1
        template: approve
      - name: s2
        template: approve
  - name: approve
    suspend: {}
`))
	for i := 0; i < 4 && !r.woc.wf.Status.Phase.Completed(); i++ {
		r.r4Resume(ctx)
		r.op(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(r.woc))
}

// C62: the completion metric of a task whose argument comes from another
// task's output is labelled with the resolved argument. Ports
// TestProbe_v1x49_DAGOutputArgMetricLabel.
func TestRegressionR4_C62_DAGOutputArgMetricLabel(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c62-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: gen
        template: gen
      - name: use
        depends: gen
        template: use
        arguments:
          parameters:
          - name: p
            value: "{{tasks.gen.outputs.parameters.out}}"
  - name: gen
    outputs:
      parameters:
      - name: out
        valueFrom:
          path: /tmp/out
    container:
      image: busybox
  - name: use
    inputs:
      parameters:
      - name: p
    metrics:
      prometheus:
      - name: r4_c62_dag_output_arg
        help: "labelled by an input fed from another task's output"
        labels:
        - key: p
          value: "{{inputs.parameters.p}}"
        counter:
          value: "1"
    container:
      image: busybox
`, 0, nil, func(n *wfv1.NodeStatus) *wfv1.Outputs {
		if n.TemplateName != "gen" {
			return nil
		}
		return &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "out", Value: wfv1.AnyStringPtr("val")}}}
	})
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	for _, c := range woc.wf.Status.Conditions {
		assert.NotEqual(t, wfv1.ConditionTypeMetricsError, c.Type, c.Message)
	}
	assert.InDelta(t, 1.0, r4C65Counter(t, "r4_c62_dag_output_arg", "p", "val"), 0.001)
}

// C62: the completion metric of an expanded step is labelled with its item.
// Ports TestProbe_v1x49_StepsItemMetricInputLabel.
func TestRegressionR4_C62_StepsItemMetricInputLabel(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c62-steps-items
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: echo
        arguments:
          parameters:
          - name: tag
            value: "{{item}}"
        withItems: [s1, s2]
  - name: echo
    inputs:
      parameters:
      - name: tag
    metrics:
      prometheus:
      - name: r4_c62_steps_items
        help: "per item counter"
        labels:
        - key: tag
          value: "{{inputs.parameters.tag}}"
        counter:
          value: "1"
    container:
      image: busybox
`, 0, nil, nil)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	for _, c := range woc.wf.Status.Conditions {
		assert.NotEqual(t, wfv1.ConditionTypeMetricsError, c.Type, c.Message)
	}
	assert.InDelta(t, 1.0, r4C65Counter(t, "r4_c62_steps_items", "tag", "s1"), 0.001)
	assert.InDelta(t, 1.0, r4C65Counter(t, "r4_c62_steps_items", "tag", "s2"), 0.001)
}

// C63: a retried item that completes inside its own dispatch counts once.
// Ports TestProbe_v1x52_DAGItemsRetryCounter.
func TestRegressionR4_C63_DAGItemsRetryCounter(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c63-retry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: work
        withItems: [x, y]
  - name: work
    retryStrategy:
      limit: "1"
    metrics:
      prometheus:
      - name: r4_c63_items_retry
        help: count
        labels:
        - key: status
          value: "{{status}}"
        counter:
          value: "1"
    container:
      image: alpine
`, 2, nil, nil)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.InDelta(t, 2.0, r4C65Counter(t, "r4_c63_items_retry", "status", "Succeeded"), 0.001)
}

// C63: a nested Steps item that completes inside its own dispatch counts
// once. Ports TestProbe_v1x52_DAGItemsNestedStepsCounter.
func TestRegressionR4_C63_DAGItemsNestedStepsCounter(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c63-nested
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: inner
        withItems: [x, y]
  - name: inner
    metrics:
      prometheus:
      - name: r4_c63_items_nested
        help: count
        counter:
          value: "1"
    steps:
    - - name: s
        template: work
  - name: work
    container:
      image: alpine
`, 2, nil, nil)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.InDelta(t, 2.0, r4C65Counter(t, "r4_c63_items_nested"), 0.001)
}

// C63 and P22: a memoize cache hit emits its template's completion metrics
// once, like an unmemoized run (a decided deviation, P22: main emitted
// none for a hit, so this fails at base). Rewrites
// TestProbe_v1x52_MemoCacheHitCounter, which expected 0.
func TestRegressionR4_C63_MemoCacheHitCounter(t *testing.T) {
	memoTmpl := func(name, metric string) string {
		return `
  - name: ` + name + `
    memoize:
      key: hit
      cache:
        configMap:
          name: r4-c63-memo-cache
    metrics:
      prometheus:
      - name: ` + metric + `
        help: count
        counter:
          value: "1"
    container:
      image: alpine
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
`
	}
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c63-memo
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: static-memo
      - name: B
        template: item-memo
        withItems: [x, y]
`+memoTmpl("static-memo", "r4_c63_memo_static")+memoTmpl("item-memo", "r4_c63_memo_item"), 2, r4MemoCache("r4-c63-memo-cache", "v"), nil)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.InDelta(t, 1.0, r4C65Counter(t, "r4_c63_memo_static"), 0.001, "static hit")
	assert.InDelta(t, 2.0, r4C65Counter(t, "r4_c63_memo_item"), 0.001, "one per item hit")
}

// C42: a newer value exported further down the tree (by a nested step)
// wins over an earlier step's: the Steps boundary does not re-export its
// steps' outputs when it completes. Ports TestProbe_v2x3_NestedStepsOverwrite.
func TestRegressionR4_C42_NestedStepsOverwrite(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c42-nested
  namespace: default
spec:
  entrypoint: entry
  onExit: exit
  templates:
  - name: entry
    steps:
    - - name: mid
        template: mid
    - - name: d
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
  - name: mid
    steps:
    - - name: a
        template: produce
    - - name: b
        template: nested
  - name: nested
    steps:
    - - name: inner
        template: produce
`+r4GlobalTemplates, 0, nil, func(n *wfv1.NodeStatus) *wfv1.Outputs {
		switch n.DisplayName {
		case "a":
			return r4GlobalOut("A")
		case "inner":
			return r4GlobalOut("B")
		}
		return nil
	})
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, "B", r4InputParam(woc.wf, "d"), "step d")
	assert.Equal(t, "B", r4InputParam(woc.wf, "e"), "onExit e")
	assert.Equal(t, "B", r4GlobalParam(woc.wf), "wf.status.outputs g")
}

// C42: the value an expanded step's item exported last wins over an earlier
// step's. Ports TestProbe_v2x3_ExpandedStepOverwrite.
func TestRegressionR4_C42_ExpandedStepOverwrite(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c42-expanded
  namespace: default
spec:
  entrypoint: entry
  onExit: exit
  templates:
  - name: entry
    steps:
    - - name: a
        template: produce
    - - name: b
        template: produce
        withItems: [1]
`+r4GlobalTemplates, 0, nil, func(n *wfv1.NodeStatus) *wfv1.Outputs {
		if n.DisplayName == "a" {
			return r4GlobalOut("A")
		}
		if n.TemplateName == "produce" {
			return r4GlobalOut("B")
		}
		return nil
	})
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, "B", r4InputParam(woc.wf, "e"), "onExit e")
	assert.Equal(t, "B", r4GlobalParam(woc.wf), "wf.status.outputs g")
}

// P9: a memoize cache hit in a Steps template exports its globalName output
// when it is fulfilled, so the next step reads it (a decided deviation:
// main's Steps exported only pod outputs, so this fails at base).
func TestRegressionR4_P9_MemoHitStepExportsGlobal(t *testing.T) {
	woc := r4MetricsRun(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p9-memo
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    steps:
    - - name: a
        template: memo
    - - name: c
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
  - name: memo
    memoize:
      key: hit
      cache:
        configMap:
          name: r4-p9-memo-cache
    container:
      image: alpine
    outputs:
      parameters:
      - name: p
        globalName: g
        valueFrom:
          path: /tmp/p
`+r4GlobalTemplates, 0, r4MemoCache("r4-p9-memo-cache", "M"), nil)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, "M", r4InputParam(woc.wf, "c"), "step c")
	assert.Equal(t, "M", r4GlobalParam(woc.wf), "wf.status.outputs g")
}

// P9: an HTTP step exports its globalName output when task-set
// reconciliation completes it, so the next step reads it (a decided
// deviation: main's Steps exported only pod outputs, so this fails at base).
func TestRegressionR4_P9_HTTPStepExportsGlobal(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p9-http
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    steps:
    - - name: h
        template: http
    - - name: c
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
  - name: http
    http:
      url: http://example.com
    outputs:
      parameters:
      - name: p
        globalName: g
        valueFrom:
          expression: "response.body"
` + r4GlobalTemplates)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := r4Operate(t, ctx, controller, wf)
	h := woc.wf.Status.Nodes.FindByDisplayName("h")
	require.NotNil(t, h)
	podcs := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace)
	agent, err := podcs.Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
	require.NoError(t, err)
	agent.Status.Phase = apiv1.PodRunning
	agent, err = podcs.Update(ctx, agent, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, controller.PodController.TestingPodInformer(), agent, func(obj any) bool {
		return obj.(*apiv1.Pod).Status.Phase == apiv1.PodRunning
	})
	r4ReportTaskSet(ctx, t, woc, map[string]wfv1.NodeResult{h.ID: {Phase: wfv1.NodeSucceeded, Outputs: r4GlobalOut("H")}})
	for i := 0; i < 6 && !woc.wf.Status.Phase.Completed(); i++ {
		woc = r4Operate(t, ctx, controller, woc.wf)
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Type != wfv1.NodeTypePod || n.Fulfilled() {
				return ""
			}
			return apiv1.PodSucceeded
		})
	}
	assert.Equal(t, "H", r4InputParam(woc.wf, "c"), "step c")
	assert.Equal(t, "H", r4GlobalParam(woc.wf), "wf.status.outputs g")
}

// P9: two DAG tasks export the same globalName; the one that finished last
// wins, also once the DAG completes (a decided deviation: main re-exported
// every task's outputs in declaration order when the DAG completed, so b's
// older value won at base).
func TestRegressionR4_P9_DAGGlobalCompletionOrder(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p9-order
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    dag:
      tasks:
      - name: a
        template: produce
      - name: b
        template: produce
      - name: c
        depends: a && b
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
`+r4GlobalTemplates)
	// b finishes first, a after it.
	finish := func(display, value string) {
		n := r.woc.wf.Status.Nodes.FindByDisplayName(display)
		require.NotNil(t, n, display)
		r4TaskResultOutputs(ctx, r.woc, n.Name, *r4GlobalOut(value))
		r.r4SetPods(ctx, map[string]apiv1.PodPhase{display: apiv1.PodSucceeded})
	}
	finish("b", "B")
	require.Equal(t, "B", r4GlobalParam(r.woc.wf))
	finish("a", "A")
	for i := 0; i < 4 && !r.woc.wf.Status.Phase.Completed(); i++ {
		r.r4SetPods(ctx, map[string]apiv1.PodPhase{"c": apiv1.PodSucceeded})
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase)
	assert.Equal(t, "A", r4InputParam(r.woc.wf, "c"), "task c")
	assert.Equal(t, "A", r4GlobalParam(r.woc.wf), "wf.status.outputs g")
}

// r4C64DefaultsMetric is a workflow whose only metric comes from
// spec.templateDefaults, labelled by node name and status.
func r4C64DefaultsMetric(name, metric, body string) string {
	return `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ` + name + `
  namespace: default
spec:
  entrypoint: main
  templateDefaults:
    metrics:
      prometheus:
      - name: ` + metric + `
        help: "templateDefaults counter"
        labels:
        - key: node
          value: "{{node.name}}"
        - key: status
          value: "{{status}}"
        counter:
          value: "1"
  templates:
` + body + `
  - name: echo
    container:
      image: busybox
`
}

// r4C64AssertItemsCounted checks that each pod node of woc was counted once.
func r4C64AssertItemsCounted(t *testing.T, woc *wfOperationCtx, metric string) {
	t.Helper()
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	var pods []string
	for _, n := range woc.wf.Status.Nodes {
		if n.Type == wfv1.NodeTypePod {
			pods = append(pods, n.Name)
		}
	}
	require.Len(t, pods, 2)
	for _, n := range pods {
		assert.InDelta(t, 1.0, r4C65Counter(t, metric, "node", n, "status", "Succeeded"), 0.001, n)
	}
}

// C64: spec.templateDefaults metrics are emitted once per withItems item of a
// DAG task. Ports TestProbe_v1x70_DAGItemsTemplateDefaultsMetric.
func TestRegressionR4_C64_DAGItemsTemplateDefaultsMetric(t *testing.T) {
	woc := r4MetricsRun(t, r4C64DefaultsMetric("r4-c64-dag-items", "r4_c64_dag_items", `
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        withItems: [d1, d2]
`), 3, nil, nil)
	r4C64AssertItemsCounted(t, woc, "r4_c64_dag_items")
}

// C64: the Steps form. Ports
// TestProbe_v1x70_StepsItemsTemplateDefaultsMetric.
func TestRegressionR4_C64_StepsItemsTemplateDefaultsMetric(t *testing.T) {
	woc := r4MetricsRun(t, r4C64DefaultsMetric("r4-c64-steps-items", "r4_c64_steps_items", `
  - name: main
    steps:
    - - name: A
        template: echo
        withItems: [s1, s2]
`), 3, nil, nil)
	r4C64AssertItemsCounted(t, woc, "r4_c64_steps_items")
}

// C87: a step exit hook whose optional artifact argument names an output the
// step did not produce ends Error "missing location information", as the
// hook requires its input, instead of running without it. Ports
// TestProbe_v2x5_StepsExitHookOptionalArgMissing.
func TestRegressionR4_C87_StepsExitHookOptionalArgMissing(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c87-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: gen
        hooks:
          exit:
            template: consume
            arguments:
              artifacts:
              - name: in
                from: "{{steps.a.outputs.artifacts.out}}"
                optional: true
  - name: gen
    container:
      image: busybox
    outputs:
      artifacts:
      - name: out
        path: /tmp/out
        optional: true
  - name: consume
    inputs:
      artifacts:
      - name: in
        path: /tmp/in
    container:
      image: busybox
`)
	makePodsPhase(ctx, r.woc, apiv1.PodSucceeded, withOutputs(ctx, wfv1.Outputs{Parameters: []wfv1.Parameter{}}))
	for range 3 {
		r.op(ctx)
	}
	makePodsPhase(ctx, r.woc, apiv1.PodSucceeded)
	for range 3 {
		r.op(ctx)
	}
	hook, err := r.woc.wf.GetNodeByName("r4-c87-steps[0].a.onExit")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, hook.Phase)
	assert.Contains(t, hook.Message, "missing location information")
	assert.Len(t, r4PodNodeNames(ctx, t, r.woc), 1, "no hook pod")
	assert.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase)
}

// C87: a workflow exit hook whose artifact argument is a raw
// {{workflow.outputs.artifacts.g}} tag ends Error "missing location
// information" instead of creating a pod with a location-less artifact.
// Ports TestProbe_v2x5_WfExitHookGlobalArtifact.
func TestRegressionR4_C87_WfExitHookGlobalArtifact(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c87-wf-exit
  namespace: default
spec:
  entrypoint: main
  hooks:
    exit:
      template: consume
      arguments:
        artifacts:
        - name: in
          from: "{{workflow.outputs.artifacts.g}}"
  templates:
  - name: main
    container:
      image: busybox
    outputs:
      artifacts:
      - name: out
        path: /tmp/out
        globalName: g
  - name: consume
    inputs:
      artifacts:
      - name: in
        path: /tmp/in
    container:
      image: busybox
`)
	makePodsPhase(ctx, r.woc, apiv1.PodSucceeded, withOutputs(ctx, wfv1.Outputs{Artifacts: []wfv1.Artifact{{Name: "out", GlobalName: "g", ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: "gen/out.tgz"}}}}}))
	for range 3 {
		r.op(ctx)
	}
	setPodPhases(ctx, r.woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.Fulfilled() {
			return ""
		}
		return apiv1.PodFailed
	})
	for range 3 {
		r.op(ctx)
	}
	assert.Equal(t, wfv1.WorkflowError, r.woc.wf.Status.Phase)
	assert.Contains(t, r.woc.wf.Status.Message, "missing location information")
	assert.Len(t, r4PodNodeNames(ctx, t, r.woc), 1, "no hook pod")
}

// P9: a suspend resumed with `argo resume` takes its supplied output's
// default, and that value is exported as its globalName when the node is
// fulfilled (by ResumeWorkflow, outside the controller), so the consumer and
// wf.status.outputs see it. A decided deviation (P9) that fails at base in
// both forms: main's DAG exported g only in its completion sweep, so
// wf.status.outputs ended right but c, dispatched before, got the raw tag;
// main's Steps never exported it.
func TestRegressionR4_P9_ResumedSuspendDefaultExportsGlobal(t *testing.T) {
	const approve = `
  - name: approve
    suspend: {}
    outputs:
      parameters:
      - name: p
        globalName: g
        valueFrom:
          supplied: {}
          default: D
`
	for name, entry := range map[string]string{
		"DAG": `
  - name: entry
    dag:
      tasks:
      - name: approve
        template: approve
      - name: c
        depends: approve
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
`,
		"Steps": `
  - name: entry
    steps:
    - - name: approve
        template: approve
    - - name: c
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p9-resume-default
  namespace: default
spec:
  entrypoint: entry
  templates:
`+entry+approve+r4GlobalTemplates)
			require.Equal(t, wfv1.NodeRunning, r.r4Phase("approve"))
			r.r4Resume(ctx)
			r.woc = r4DriveToEnd(t, ctx, r.controller, r.woc, 6)
			assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase, "unfulfilled: %v", r4Unfulfilled(r.woc))
			assert.Equal(t, "D", r4InputParam(r.woc.wf, "c"), "task c")
			assert.Equal(t, "D", r4GlobalParam(r.woc.wf), "wf.status.outputs g")
		})
	}
}

// P9: a pod that ends Failed before its task result is complete is stored
// Failed but not fulfilled. When its task result syncs in a later reconcile,
// that is its completion: its globalName output is exported (the consumer
// dispatched in that reconcile sees it) and its completion metric is emitted
// once. Main exported g but missed the metric (it took its "before" snapshot
// after merging task results, so the node already looked fulfilled): the
// metric assertion is a fix of main, so this fails at base.
func TestRegressionR4_P9_LateSyncedFailedPodExportsGlobal(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p9-late-sync
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    dag:
      tasks:
      - name: a
        template: produce-fail
      - name: c
        depends: a.Failed
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
  - name: produce-fail
    metrics:
      prometheus:
      - name: r4_p9_late_sync
        help: count
        labels:
        - key: status
          value: "{{status}}"
        counter:
          value: "1"
    container:
      image: alpine
    outputs:
      parameters:
      - name: p
        globalName: g
        valueFrom:
          path: /tmp/p
`+r4GlobalTemplates)
	const a = "r4-p9-late-sync.a"
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodRunning, r4PodForNode(a), r4IncompleteTaskResult(ctx))
	r.op(ctx)
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodFailed, r4PodForNode(a), withExitCode(1))
	r.op(ctx)
	node, err := r.woc.wf.GetNodeByName(a)
	require.NoError(t, err)
	require.Equal(t, wfv1.NodeFailed, node.Phase)
	require.False(t, node.Fulfilled(), "a waits for its task result")

	// The wait container reports the outputs and completes the task result.
	trs := r.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(r.woc.wf.Namespace)
	tr, err := trs.Get(ctx, node.ID, metav1.GetOptions{})
	require.NoError(t, err)
	tr.Outputs = r4GlobalOut("F")
	tr.Labels[common.LabelKeyReportOutputsCompleted] = "true"
	tr, err = trs.Update(ctx, tr, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, r.controller.taskResultInformer, tr, func(obj any) bool {
		return obj.(*wfv1.WorkflowTaskResult).Labels[common.LabelKeyReportOutputsCompleted] == "true"
	})
	r.op(ctx)
	assert.Equal(t, "F", r4InputParam(r.woc.wf, "c"), "task c, dispatched when a completed")

	r.woc = r4DriveToEnd(t, ctx, r.controller, r.woc, 6)
	assert.Equal(t, "F", r4GlobalParam(r.woc.wf), "wf.status.outputs g")
	assert.InDelta(t, 1.0, r4C65Counter(t, "r4_p9_late_sync", "status", "Failed"), 0.001)
}

// C66 and lead 2: a retried template's metrics belong to its Retry node.
//
// A realtime gauge must be registered once, for the Retry node
// (status=Running, duration covering the whole retry), not once per attempt
// (status=Pending, duration resetting each attempt). A completion counter
// must count each fulfilled attempt once, not twice.

// r4GaugeStatuses reads name's realtime gauge value for each of a fixed set
// of {{status}} label values, returning only the ones actually registered.
func r4GaugeStatuses(ctx context.Context, name string) map[string]float64 {
	out := map[string]float64{}
	for _, status := range []string{"Running", "Pending", "Succeeded", "Failed"} {
		attribs := attribute.NewSet(attribute.String("status", status))
		if v, err := testExporter.GetFloat64GaugeValue(ctx, name, &attribs); err == nil {
			out[status] = v
		}
	}
	return out
}

// C66: ports TestProbe_v1x53_RealtimeMetricRetriedTemplate.
func TestRegressionR4_C66_RealtimeMetricRetriedTemplate(t *testing.T) {
	for _, kind := range []string{"root", "dag", "steps"} {
		t.Run(kind, func(t *testing.T) {
			entry := "work"
			switch kind {
			case "dag":
				entry = "maindag"
			case "steps":
				entry = "mainsteps"
			}
			metric := "r4_c66_rt_retry_" + kind
			wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c66-rt-retry-` + kind + `
  namespace: default
spec:
  entrypoint: ` + entry + `
  templates:
  - name: maindag
    dag:
      tasks:
      - name: A
        template: work
  - name: mainsteps
    steps:
    - - name: A
        template: work
  - name: work
    retryStrategy:
      limit: "2"
    metrics:
      prometheus:
      - name: ` + metric + `
        help: g
        labels:
        - key: status
          value: "{{status}}"
        gauge:
          realtime: true
          value: "{{duration}}"
    container:
      image: alpine
      command: [echo, hi]
`)
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			s1 := r4GaugeStatuses(ctx, metric)

			// Fail the first attempt and drive a second attempt into being.
			makePodsPhase(ctx, woc, apiv1.PodFailed)
			for range 3 {
				woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
				woc.operate(ctx)
			}

			_, running := s1["Running"]
			_, pending := s1["Pending"]
			assert.True(t, running, "status=Running series from the Retry node")
			assert.False(t, pending, "no status=Pending series from an attempt node")
		})
	}
}

// lead2, folded into C66: a Steps/DAG template with retryStrategy and a
// {{status}} counter counts each failed non-final attempt once, not twice.
// Ports TestProbe_lead2_RetriedNestedFailOnceThenSucceed.
func TestRegressionR4_C66_RetriedNestedCounterCountsOnce(t *testing.T) {
	for _, parent := range []string{"steps", "dag"} {
		for _, inner := range []string{"steps", "dag"} {
			t.Run(parent+"/"+inner, func(t *testing.T) {
				metric := "r4_c66_once_" + parent + "_" + inner
				parentBody := `
    steps:
    - - name: s
        template: inner`
				if parent == "dag" {
					parentBody = `
    dag:
      tasks:
      - name: s
        template: inner`
				}
				innerBody := `
    steps:
    - - name: w
        template: work`
				if inner == "dag" {
					innerBody = `
    dag:
      tasks:
      - name: w
        template: work`
				}
				wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c66-once-` + parent + `-` + inner + `
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main` + parentBody + `
  - name: inner
    retryStrategy:
      limit: "1"
    metrics:
      prometheus:
      - name: ` + metric + `
        help: "inner completions by status"
        labels:
        - key: status
          value: "{{status}}"
        counter:
          value: "1"` + innerBody + `
  - name: work
    container:
      image: alpine
      command: [sh, -c, "exit 1"]
`)
				require.NoError(t, validate.Workflow(logging.TestContext(t.Context()), nil, nil, wf, nil, validate.Opts{}))

				ctx := logging.TestContext(t.Context())
				cancel, controller := newController(ctx, wf)
				defer cancel()
				woc := newWorkflowOperationCtx(ctx, wf, controller)
				woc.operate(ctx)
				for _, ph := range []apiv1.PodPhase{apiv1.PodFailed, apiv1.PodSucceeded, apiv1.PodSucceeded} {
					if woc.wf.Status.Phase.Completed() {
						break
					}
					code := int32(0)
					if ph == apiv1.PodFailed {
						code = 1
					}
					makePodsPhase(ctx, woc, ph, withExitCode(code))
					woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
					woc.operate(ctx)
				}

				failed := r4C65Counter(t, metric, "status", "Failed")
				t.Logf("wf=%s Failed=%.0f", woc.wf.Status.Phase, failed)
				require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
				require.InDelta(t, 1.0, failed, 0.001, "one failed attempt should count once")
			})
		}
	}
}

// r4C39Inner is a Steps template whose output parameter reads a when-false
// step's output with no default: the output cannot be resolved, an error of
// the template itself.
const r4C39Inner = `
  - name: inner
    retryStrategy:
      limit: "2"
      retryPolicy: OnError
    steps:
    - - name: a
        template: produce
        when: "false"
      - name: b
        template: echo
    outputs:
      parameters:
      - name: x
        valueFrom:
          parameter: "{{steps.a.outputs.parameters.p}}"
  - name: produce
    container:
      image: alpine
      command: [sh, -c, "echo 1 > /tmp/p"]
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
  - name: echo
    container:
      image: alpine
      command: [echo, hi]
`

// TestRegressionR4_C39_RetryOnError ports TestProbe_v1x28_RetryOnError
// (v1x28-1_test.go / C39). A Steps template whose outputs cannot be
// resolved ends Error, as executeSteps' returned error did, so its
// retryStrategy with retryPolicy OnError retries it: b runs once per
// attempt. HEAD ended it Failed, which OnError does not retry.
func TestRegressionR4_C39_RetryOnError(t *testing.T) {
	woc := r4RunResults(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c39-onerror
  namespace: default
spec:
  entrypoint: top
  templates:
  - name: top
    steps:
    - - name: call
        template: inner
`+r4C39Inner, nil, 12, 3, nil)
	require.True(t, woc.wf.Status.Phase.Completed())
	ctx := logging.TestContext(t.Context())
	assert.Len(t, r4CommandsMatching(ctx, t, woc, "].b"), 3, "b runs once per attempt: %v", r4PodNodeNames(ctx, t, woc))
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "r4-c39-onerror[0].call(0)"), "the attempt ends Error")
}

// TestRegressionR4_C39_ItemsScriptResultOutput ports
// TestProbe_v1x28_ItemsScriptResultOutput (v1x28-1_test.go / C39). The Steps
// template's output reads the aggregated result of a withItems script that
// printed plain text, which cannot be aggregated: the template itself errors,
// and it and the workflow end Error, not Failed.
func TestRegressionR4_C39_ItemsScriptResultOutput(t *testing.T) {
	woc := r4RunResults(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c39-itemsout
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: say
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [a, b]
    outputs:
      parameters:
      - name: all
        valueFrom:
          parameter: "{{steps.gen.outputs.result}}"
  - name: say
    inputs:
      parameters:
      - name: msg
    script:
      image: alpine
      command: [sh]
      source: echo hello {{inputs.parameters.msg}}
`, map[string]string{"say": "hello x"}, 10, 3, nil)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "r4-c39-itemsout"))
}

// TestRegressionR4_C59_PodRejectedMessage ports
// TestProbe_v1x30_PodRejectedMessage (v1x30-1_test.go / C59). A step whose
// pod an admission webhook rejects ends Error; its StepGroup ends Error with
// main's "step group deemed errored due to child <name> error: <reason>",
// which the Steps node and the workflow carry. HEAD said only "child '<id>'
// failed" and left the StepGroup Failed. Per P23 the rule is keyed on the
// step's phase.
func TestRegressionR4_C59_PodRejectedMessage(t *testing.T) {
	woc := r4RunResults(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c59-rej
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
    - - name: b
        template: badpod
  - name: badpod
    container:
      image: alpine
      args: [reject-me]
  - name: work
    container:
      image: alpine
`, nil, 8, 2, func(controller *WorkflowController) {
		r4RejectPodCreate(controller, func(pod *apiv1.Pod) bool {
			return slices.Contains(pod.Spec.Containers[len(pod.Spec.Containers)-1].Args, "reject-me")
		}, apierr.NewBadRequest(`admission webhook "policy.example.com" denied the request: image not allowed`))
	})
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "step group deemed errored due to child r4-c59-rej[1].b error:")
	assert.Contains(t, woc.wf.Status.Message, "denied the request", "the workflow message carries the reason")
	root, err := woc.wf.GetNodeByName("r4-c59-rej")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, root.Phase)
	assert.Contains(t, root.Message, "denied the request", "the Steps node message carries the reason")
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "r4-c59-rej[1]"), "the StepGroup")
}

// TestRegressionR4_C59_WithParamNotJSONMessage ports
// TestProbe_v1x30_WithParamNotJSONMessage (v1x30-1_test.go / C59). A step
// whose withParam is not a JSON list errors, and the workflow message says
// why rather than only naming the step's node.
func TestRegressionR4_C59_WithParamNotJSONMessage(t *testing.T) {
	woc := r4RunResults(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c59-wp
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: gen
    - - name: fan
        template: echo
        arguments: {parameters: [{name: msg, value: "{{item}}"}]}
        withParam: "{{steps.gen.outputs.result}}"
  - name: gen
    script:
      image: alpine
      command: [sh]
      source: echo hi
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine
      args: ["{{inputs.parameters.msg}}"]
`, map[string]string{"gen": "not json"}, 8, 2, nil)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "withParam value could not be parsed as a JSON list")
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "r4-c59-wp[1]"), "the StepGroup")
}

// r4C40Say is a withItems script template that prints plain text, whose
// results cannot be aggregated into a JSON list, and a plain step.
const r4C40Say = `
  - name: say
    inputs:
      parameters:
      - name: msg
    script:
      image: alpine
      command: [sh]
      source: echo hello {{inputs.parameters.msg}}
  - name: echo
    inputs:
      parameters:
      - name: in
        value: ""
    container:
      image: alpine
      command: [echo, "{{inputs.parameters.in}}"]
`

// TestRegressionR4_C40_AggregationErrorWorkflowError ports
// TestProbe_v1x60_AggregationErrorWorkflowError (v1x60-1_test.go / C40).
// gen's items print plain text, so their results cannot be aggregated for
// the next group: as when executeSteps aggregated a finished group into the
// template's scope, that is an error of the Steps template itself, which
// ends Error, as does the workflow; use never runs. HEAD recorded the error
// on use, and the Steps node and the workflow ended Failed.
func TestRegressionR4_C40_AggregationErrorWorkflowError(t *testing.T) {
	woc := r4RunResults(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c40-agg
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: gen
        template: say
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [a, b]
    - - name: use
        template: echo
        arguments:
          parameters:
          - name: in
            value: "{{steps.gen.outputs.result}}"
`+r4C40Say, map[string]string{"say": "hello x"}, 6, 0, nil)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodeError, r4NodePhase(woc, "r4-c40-agg"))
	assert.Empty(t, r4NodePhase(woc, "r4-c40-agg[1].use"), "use never starts")
}

// TestRegressionR4_C40_AggErrContinueOnError ports
// TestProbe_v1x60_AggErrContinueOnError (v1x60-1_test.go / C40). The
// aggregation error is the template's, not use's, so use's continueOn.error
// does not apply: neither use nor after runs, and the workflow ends Error.
func TestRegressionR4_C40_AggErrContinueOnError(t *testing.T) {
	woc := r4RunResults(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c40-cont
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: fan
        template: say
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [a, b]
    - - name: use
        template: echo
        continueOn:
          error: true
    - - name: after
        template: echo
`+r4C40Say, map[string]string{"say": "hello world"}, 8, 0, nil)
	ctx := logging.TestContext(t.Context())
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Empty(t, r4CommandsMatching(ctx, t, woc, ".use"))
	assert.Empty(t, r4CommandsMatching(ctx, t, woc, ".after"))
}

// TestRegressionR4_P16_DaemonDiesAfterGroupSucceeded is a decided deviation
// (P16): it encodes the chosen behaviour, not main's. db's StepGroup [0] is
// recorded Succeeded once db is up; db then dies while test runs. [0] keeps
// the Succeeded it was recorded with, while db, the Steps node and the
// workflow go Failed, naming db.
func TestRegressionR4_P16_DaemonDiesAfterGroupSucceeded(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-p16
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: db
        template: daemon
    - - name: test
        template: work
  - name: daemon
    daemon: true
    container: {image: alpine, command: [sleep, infinity]}
  - name: work
    container: {image: alpine, command: [echo]}
`)
	const db, test = "r4-p16[0].db", "r4-p16[1].test"
	stage := func(phase apiv1.PodPhase, name string, with ...with) {
		r4SetPodsPhase(t, ctx, r.woc, phase, r4PodForNode(name), with...)
		r.op(ctx)
		r.op(ctx)
	}
	stage(apiv1.PodRunning, db, r4WithReady)
	stage(apiv1.PodRunning, test)
	require.Equal(t, wfv1.NodeSucceeded, r4NodePhase(r.woc, "r4-p16[0]"), "[0] is recorded Succeeded once db is up")
	stage(apiv1.PodFailed, db)
	dbNode, err := r.woc.wf.GetNodeByName(db)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, dbNode.Phase, "db")
	assert.Equal(t, wfv1.NodeSucceeded, r4NodePhase(r.woc, "r4-p16[0]"), "[0] keeps its recorded phase")
	assert.Equal(t, wfv1.NodeFailed, r4NodePhase(r.woc, "r4-p16"), "the Steps node")
	assert.Equal(t, wfv1.WorkflowFailed, r.woc.wf.Status.Phase)
	assert.Equal(t, fmt.Sprintf("child '%s' failed", dbNode.ID), r.woc.wf.Status.Message)
}

// r4WorkflowFailures decodes {{workflow.failures}} into its display names.
func r4WorkflowFailures(t *testing.T, woc *wfOperationCtx) []string {
	t.Helper()
	raw := woc.globalParams()[varkeys.WorkflowFailures.Template()]
	if uq, err := strconv.Unquote(raw); err == nil {
		raw = uq
	}
	var entries []failedNodeStatus
	require.NoError(t, json.Unmarshal([]byte(raw), &entries))
	var names []string
	for _, e := range entries {
		names = append(names, e.DisplayName)
	}
	sort.Strings(names)
	return names
}

// TestRegressionR4_D1_WorkflowFailuresExcludeTaskGroups ports
// TestProbe_v1x36_StepsFailuresOneItem and _DagFailuresOneItem
// (v1x36-1_test.go / D1). {{workflow.failures}} must not list a TaskGroup's
// own entry: only the failed items and the enclosing boundary node. HEAD
// added this entry for Steps (fixed here). Base already emitted it for a
// failed expanded DAG task, so excluding it for DAG too is a decided
// deviation from main (D1 in pr-16290-round4-regressions.md): the DAG
// subtest is written to the decided behaviour and fails at base as well as
// on the branch before this fix.
func TestRegressionR4_D1_WorkflowFailuresExcludeTaskGroups(t *testing.T) {
	t.Run("Steps", func(t *testing.T) {
		ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-d1-steps
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: print
        template: echo
        withItems: ["a", "b"]
  - name: echo
    container: {image: alpine, command: [echo]}
`)
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodFailed, r4PodForNode("r4-d1-steps[0].print(0:a)"))
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode("r4-d1-steps[0].print(1:b)"))
		r.op(ctx)

		// At base there is no intermediate TaskGroup node at all: the items
		// hang directly off the StepGroup. HEAD introduced one (named bare
		// "print", the same DisplayName as the TaskGroup on the DAG side
		// below) and started listing it here too; that is the regression.
		names := r4WorkflowFailures(t, r.woc)
		assert.NotContains(t, names, "print", "workflow.failures must not list the TaskGroup: %v", names)
		assert.Contains(t, names, "print(0:a)")
	})

	t.Run("DAG", func(t *testing.T) {
		ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-d1-dag
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: print
        template: echo
        withItems: ["a", "b"]
  - name: echo
    container: {image: alpine, command: [echo]}
`)
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodFailed, r4PodForNode("r4-d1-dag.print(0:a)"))
		r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, r4PodForNode("r4-d1-dag.print(1:b)"))
		r.op(ctx)

		taskGroup, err := r.woc.wf.GetNodeByName("r4-d1-dag.print")
		require.NoError(t, err)
		require.Equal(t, wfv1.NodeTypeTaskGroup, taskGroup.Type)
		require.True(t, taskGroup.FailedOrError(), "the TaskGroup should have failed along with its item")

		names := r4WorkflowFailures(t, r.woc)
		assert.NotContains(t, names, taskGroup.DisplayName, "workflow.failures must not list the TaskGroup (decided deviation from base, D1): %v", names)
		assert.Contains(t, names, "print(0:a)")
	})
}

// TestRegressionR4_C57_CsNoMainExpandedSteps ports
// TestProbe_v2x10_CsNoMainExpandedSteps (v2x10-1_test.go / C57). A
// containerSet with no container named "main" has no result for the
// executor to ever report; validation still puts the aggregated
// {{steps.cs.outputs.result}} in scope for a later step, since it doesn't
// know which container(s) exist at runtime. assessNodeStatus's gate case 2
// held the pod node at its old (non-terminal) phase forever waiting for that
// result, so the items never finished, print was never created, and the
// workflow hung with no message.
func TestRegressionR4_C57_CsNoMainExpandedSteps(t *testing.T) {
	ctx, r := r4Start(t, `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-c57-cs-nomain
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: cs
        template: cs
        withItems: ["1", "2"]
    - - name: print
        template: echo
        arguments:
          parameters:
          - name: x
            value: "{{steps.cs.outputs.result}}"
  - name: cs
    containerSet:
      containers:
      - name: a
        image: argoproj/argosay:v2
      - name: b
        image: argoproj/argosay:v2
        dependencies: [a]
  - name: echo
    inputs:
      parameters:
      - name: x
    container:
      image: argoproj/argosay:v2
`)
	pods, err := listPods(ctx, r.woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 2, "the two containerSet item pods should be created")

	all := func(*apiv1.Pod) bool { return true }
	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, all, withExitCode(0), withOutputs(ctx, wfv1.Outputs{}))
	for range 3 {
		r.op(ctx)
	}

	for _, n := range r.woc.wf.Status.Nodes {
		if n.Type == wfv1.NodeTypePod && strings.Contains(n.Name, ".cs") {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase, "containerSet pod node %s should not be stuck waiting for a result", n.Name)
		}
	}
	pods, err = listPods(ctx, r.woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 3, "print's pod should be created once the containerSet items finish")

	r4SetPodsPhase(t, ctx, r.woc, apiv1.PodSucceeded, all, withExitCode(0), withOutputs(ctx, wfv1.Outputs{}))
	for range 3 {
		r.op(ctx)
	}
	assert.Equal(t, wfv1.WorkflowSucceeded, r.woc.wf.Status.Phase)
}

// TestRegressionR4_StepGroupErrorMessageFallsBackToPhase covers a Task 5.4
// review follow-up: stepGroupOutcome (engine.go) built the "step group
// deemed errored due to child <name> error: <reason>" message straight from
// the failed node's Message, which trails off into "...error: " when that
// message is empty. There is no realistic pod path that leaves an Error
// node's Message empty (markNodeError always writes err.Error()), so this
// builds an in-flight status directly, the way the C12 legacy-upgrade tests
// do, with the failed step node already Error and Message "": engine.go
// must still produce a message naming what happened, falling back to the
// node's phase. This is a defect in this branch's own new message-building
// code (T5.4), not a base-vs-branch regression, so it is not basecheck-clean:
// base's steps.go builds Step Group messages on an entirely different path
// that predates stepGroupOutcome.
func TestRegressionR4_StepGroupErrorMessageFallsBackToPhase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	t.Cleanup(cancel)

	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: r4-sg-msg
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: denied
        template: echo
  - name: echo
    container:
      image: alpine
`)
	now := metav1.NewTime(time.Now().Add(-time.Minute))
	root, sg := wf.Name, wf.Name+"[0]"
	step := sg + ".denied"
	id := wf.NodeID
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.StartedAt = now
	wf.Status.Nodes = wfv1.Nodes{
		id(root): {ID: id(root), Name: root, DisplayName: root, Type: wfv1.NodeTypeSteps, TemplateName: "main", TemplateScope: "local/" + wf.Name, Phase: wfv1.NodeRunning, StartedAt: now, Children: []string{id(sg)}},
		id(sg):   {ID: id(sg), Name: sg, DisplayName: "[0]", Type: wfv1.NodeTypeStepGroup, TemplateName: "main", TemplateScope: "local/" + wf.Name, Phase: wfv1.NodeRunning, BoundaryID: id(root), StartedAt: now, Children: []string{id(step)}},
		id(step): {ID: id(step), Name: step, DisplayName: "denied", Type: wfv1.NodeTypePod, TemplateName: "echo", TemplateScope: "local/" + wf.Name, Phase: wfv1.NodeError, Message: "", BoundaryID: id(root), StartedAt: now, FinishedAt: now},
	}
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	sgNode, err := woc.wf.GetNodeByName(sg)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, sgNode.Phase)
	assert.NotEqual(t, fmt.Sprintf("step group deemed errored due to child %s error: ", step), sgNode.Message,
		"an empty child message must not leave the group message trailing off")
	assert.Equal(t, fmt.Sprintf("step group deemed errored due to child %s error: %s", step, wfv1.NodeError), sgNode.Message)
}
