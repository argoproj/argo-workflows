package controller

// Shared helpers for the round 4 regression red tests
// (engine_regression_r4_test.go and the per-item test files layered on top
// of it). Copied and renamed (r4 prefix) from the probe files under
// _pr-16290-round4/probes/, per pr-16290-round4-fix-plan/task-0-brief.md.
//
// These helpers must compile against wt-base (4389bbf96) as well as this
// branch: use only functions/types present in both trees. Do not add a
// helper that only exists to serve one later task's naming convenience;
// keep them general enough for reuse, matching the brief's interfaces.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// r4Operate re-reads the workflow's stored status from the fake clientset,
// builds a fresh wfOperationCtx from it, and operates once. It never reuses
// an in-memory *wfv1.Workflow across reconciles, so a bug that keeps state
// only in Engine memory (rather than in the persisted status) shows up as a
// regression here, simulating a controller restart between reconciles
// (Review Focus item 5).
//
//nolint:revive // task-0-brief.md mandates this exact signature (t before ctx)
func r4Operate(t *testing.T, ctx context.Context, controller *WorkflowController, wf *wfv1.Workflow) *wfOperationCtx {
	t.Helper()
	stored, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, stored, controller)
	woc.operate(ctx)
	return woc
}

// r4SetPodsPhase acts like makePodsPhase, but only touches pods for which
// filter returns true, leaving the rest alone.
//
//nolint:revive // task-0-brief.md mandates this exact signature (t before ctx)
func r4SetPodsPhase(t *testing.T, ctx context.Context, woc *wfOperationCtx, phase apiv1.PodPhase, filter func(*apiv1.Pod) bool, with ...with) {
	t.Helper()
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	for _, pod := range pods.Items {
		if !filter(&pod) || pod.Status.Phase == phase {
			continue
		}
		pod.Status.Phase = phase
		if phase == apiv1.PodFailed {
			pod.Status.Message = "Pod failed"
		}
		for _, w := range with {
			w(&pod, woc)
		}
		updatedPod, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		waitForInformer(ctx, woc.controller.PodController.TestingPodInformer(), updatedPod, func(obj any) bool {
			return obj.(*apiv1.Pod).Status.Phase == phase
		})
	}
}

// r4MoveNewPodsPending gives every fake pod with no phase yet a real Pending
// phase. The fake clientset creates a pod with an empty Status.Phase (real
// Kubernetes would report Pending as soon as the kubelet accepts it); an
// empty phase reaching the Engine is a harness artefact, not a real pod
// state (see the regressions document section 6.4).
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
func r4MoveNewPodsPending(ctx context.Context, woc *wfOperationCtx) {
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase != "" {
			continue
		}
		pod.Status.Phase = apiv1.PodPending
		updatedPod, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
		if err != nil {
			panic(err)
		}
		waitForInformer(ctx, woc.controller.PodController.TestingPodInformer(), updatedPod, func(obj any) bool {
			return obj.(*apiv1.Pod).Status.Phase == apiv1.PodPending
		})
	}
}

// r4RejectPodCreate prepends a fake-clientset reactor that fails pod Create
// calls matching match with err, e.g. to simulate an admission webhook
// denial or a transient quota error.
func r4RejectPodCreate(controller *WorkflowController, match func(*apiv1.Pod) bool, err error) {
	controller.kubeclientset.(*fake.Clientset).PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		pod := action.(k8stesting.CreateAction).GetObject().(*apiv1.Pod)
		if match(pod) {
			return true, nil, err
		}
		return false, nil, nil
	})
}

// r4PodCalls counts pod Create/Get calls seen by the fake clientset,
// installed by r4CountPodCalls.
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
type r4PodCalls struct {
	mu            sync.Mutex
	creates       map[string]int // pod name -> Create calls
	alreadyExists int
	gets          int
}

// r4CountPodCalls prepends fake-clientset reactors that count pod Create and
// Get calls, for asserting that a pod is only ever created once per
// reconcile even under informer lag (see r4DelayPodWatch).
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
func r4CountPodCalls(controller *WorkflowController) *r4PodCalls {
	c := &r4PodCalls{creates: map[string]int{}}
	cs := controller.kubeclientset.(*fake.Clientset)
	cs.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		pod := action.(k8stesting.CreateAction).GetObject().(*apiv1.Pod)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.creates[pod.Name]++
		if _, err := cs.Tracker().Get(apiv1.SchemeGroupVersion.WithResource("pods"), pod.Namespace, pod.Name); err == nil {
			c.alreadyExists++
			return true, nil, apierr.NewAlreadyExists(apiv1.Resource("pods"), pod.Name)
		}
		return false, nil, nil
	})
	cs.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		c.mu.Lock()
		c.gets++
		c.mu.Unlock()
		return false, nil, nil
	})
	return c
}

// r4DelayPodWatch makes the fake clientset's pod watch deliver every event d
// late, as a real API server's watch does relative to an in-process
// reconcile. Install it right after newController, before the pod informer
// has started, e.g. before the first r4Operate call.
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
func r4DelayPodWatch(controller *WorkflowController, d time.Duration) {
	if d == 0 {
		return
	}
	cs := controller.kubeclientset.(*fake.Clientset)
	cs.PrependWatchReactor("pods", func(action k8stesting.Action) (bool, watch.Interface, error) {
		w, err := cs.Tracker().Watch(action.GetResource(), action.GetNamespace())
		if err != nil {
			return false, nil, err
		}
		type stamped struct {
			ev watch.Event
			at time.Time
		}
		q := make(chan stamped, 1000)
		out := make(chan watch.Event, 1000)
		go func() {
			defer close(q)
			for ev := range w.ResultChan() {
				q <- stamped{ev, time.Now().Add(d)}
			}
		}()
		go func() {
			defer close(out)
			for s := range q {
				time.Sleep(time.Until(s.at))
				out <- s.ev
			}
		}()
		pw := watch.NewProxyWatcher(out)
		go func() { <-pw.StopChan(); w.Stop() }()
		return true, pw, nil
	})
}

// r4TaskResultOutputs writes a complete WorkflowTaskResult (Succeeded, with
// the report-outputs-completed label) for the node named nodeName, as the
// executor would after a successful run, and waits for the task result
// informer to catch up.
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
func r4TaskResultOutputs(ctx context.Context, woc *wfOperationCtx, nodeName string, out wfv1.Outputs) {
	nodeID := woc.wf.NodeID(nodeName)
	taskResult := &wfv1.WorkflowTaskResult{
		TypeMeta: metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskResultKind},
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeID,
			Labels: map[string]string{
				common.LabelKeyWorkflow:               woc.wf.Name,
				common.LabelKeyReportOutputsCompleted: "true",
			},
		},
		NodeResult: wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: out.DeepCopy()},
	}
	created, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(woc.wf.Namespace).Create(ctx, taskResult, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	waitForInformer(ctx, woc.controller.taskResultInformer, created, func(any) bool { return true })
}

// r4LegacyStepItem is one item of an expanded step in a legacy (pre-Engine)
// in-flight Steps status: its item name (as it appears in "step(item)") and
// the phase the base controller had recorded for it.
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
type r4LegacyStepItem struct {
	Name  string
	Phase wfv1.NodePhase
}

// r4LegacyStepsStatus builds a Steps workflow whose status is base-shaped:
// the items of the expanded step stepName, in StepGroup [groupIdx], hang
// directly off the StepGroup node, with no TaskGroup node in between, as
// the pre-Engine controller always wrote it. This is for C12 (in-flight
// upgrade): on the branch's first reconcile after the upgrade, the Engine
// must adopt these legacy items into a TaskGroup it creates for them,
// rather than creating a fresh, empty TaskGroup that ignores items which
// already finished or failed under the old controller.
//
//nolint:unused // helper for later round-4 red tests built on this harness (Task 0)
func r4LegacyStepsStatus(manifest string, groupIdx int, stepName, tmplName string, items []r4LegacyStepItem) *wfv1.Workflow {
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	now := metav1.NewTime(time.Now().Add(-time.Minute))
	root := wf.Name
	sg := fmt.Sprintf("%s[%d]", root, groupIdx)
	id := wf.NodeID
	mk := func(name, display string, typ wfv1.NodeType, tmpl string, phase wfv1.NodePhase, boundary string, children ...string) wfv1.NodeStatus {
		n := wfv1.NodeStatus{
			ID: id(name), Name: name, DisplayName: display, Type: typ, TemplateName: tmpl,
			TemplateScope: "local/" + wf.Name, Phase: phase, BoundaryID: boundary, StartedAt: now,
		}
		for _, c := range children {
			n.Children = append(n.Children, id(c))
		}
		if phase.Fulfilled(nil) {
			n.FinishedAt = now
		}
		if phase == wfv1.NodeFailed {
			n.Message = "Pod failed"
		}
		return n
	}
	itemNames := make([]string, 0, len(items))
	nodes := wfv1.Nodes{}
	for _, it := range items {
		name := fmt.Sprintf("%s.%s(%s)", sg, stepName, it.Name)
		itemNames = append(itemNames, name)
		nodes[id(name)] = mk(name, fmt.Sprintf("%s(%s)", stepName, it.Name), wfv1.NodeTypePod, tmplName, it.Phase, id(root))
	}
	nodes[id(root)] = mk(root, root, wfv1.NodeTypeSteps, wf.Spec.Entrypoint, wfv1.NodeRunning, "", sg)
	nodes[id(sg)] = mk(sg, fmt.Sprintf("[%d]", groupIdx), wfv1.NodeTypeStepGroup, wf.Spec.Entrypoint, wfv1.NodeRunning, id(root), itemNames...)
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.StartedAt = now
	wf.Status.Nodes = nodes
	return wf
}

// r4NamespacedMutex builds a Synchronization holding a single mutex named
// name, scoped to the workflow's own namespace (the default when Namespace
// is left unset).
func r4NamespacedMutex(name string) *wfv1.Synchronization {
	return &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: name}}}
}
