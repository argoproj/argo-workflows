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
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfsync "github.com/argoproj/argo-workflows/v4/workflow/sync"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
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
type r4PodCalls struct {
	mu            sync.Mutex
	creates       map[string]int // pod name -> Create calls
	alreadyExists int
	gets          int
}

// r4CountPodCalls prepends fake-clientset reactors that count pod Create and
// Get calls, for asserting that a pod is only ever created once per
// reconcile even under informer lag (see r4DelayPodWatch).
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
//nolint:unparam // a general builder of the legacy shape; today's callers happen to share the group and step
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

// r4StartLocked is r4Start with a real lock manager and the my-config
// semaphore ConfigMap (workflow: 2, template: 1), so template locks are
// taken and released as in production.
func r4StartLocked(t *testing.T, manifest string, objects ...any) (context.Context, *r4Run) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, append([]any{wf}, objects...)...)
	t.Cleanup(cancel)
	var err error
	controller.syncManager, err = wfsync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(string) {}, workflowExistenceFunc, false)
	require.NoError(t, err)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)
	return ctx, &r4Run{t: t, controller: controller, woc: r4Operate(t, ctx, controller, wf)}
}

// r4SetPods sets the pods of the nodes named by display name to their phase
// in phases, moves every other unfinished node's pod to Running, as a
// kubelet would, and reconciles.
func (r *r4Run) r4SetPods(ctx context.Context, phases map[string]apiv1.PodPhase) {
	r.t.Helper()
	setPodPhases(ctx, r.woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if p, ok := phases[n.DisplayName]; ok {
			return p
		}
		if !n.Fulfilled() {
			return apiv1.PodRunning
		}
		return ""
	})
	r.op(ctx)
}

// r4Phase is the phase of the node with the given display name, or "".
func (r *r4Run) r4Phase(display string) wfv1.NodePhase {
	if n := r.woc.wf.Status.Nodes.FindByDisplayName(display); n != nil {
		return n.Phase
	}
	return ""
}

// r4AgeNode moves the stored start of the node with the given display name d
// into the past, as if that much time had passed since it started.
func (r *r4Run) r4AgeNode(ctx context.Context, display string, d time.Duration) {
	r.t.Helper()
	wfs := r.controller.wfclientset.ArgoprojV1alpha1().Workflows(r.woc.wf.Namespace)
	stored, err := wfs.Get(ctx, r.woc.wf.Name, metav1.GetOptions{})
	require.NoError(r.t, err)
	n := stored.Status.Nodes.FindByDisplayName(display)
	require.NotNil(r.t, n, display)
	n.StartedAt = metav1.NewTime(n.StartedAt.Add(-d))
	stored.Status.Nodes[n.ID] = *n
	_, err = wfs.Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(r.t, err)
}

// r4Resume resumes every suspended node, as `argo resume` does, and
// reconciles.
func (r *r4Run) r4Resume(ctx context.Context) {
	r.t.Helper()
	wfs := r.controller.wfclientset.ArgoprojV1alpha1().Workflows(r.woc.wf.Namespace)
	require.NoError(r.t, wfutil.ResumeWorkflow(ctx, wfs, r.controller.hydrator, r.woc.wf.Name, ""))
	r.op(ctx)
}

// r4MetricsRun drives manifest with every pod succeeding (with the outputs
// of outputsFor, if it returns any) until the workflow completes, then
// reconciles extra more times, so that a completion metric emitted twice
// shows. setup runs on the controller before the first reconcile.
func r4MetricsRun(t *testing.T, manifest string, extra int, setup func(context.Context, *WorkflowController), outputsFor func(*wfv1.NodeStatus) *wfv1.Outputs) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	require.NoError(t, validate.Workflow(ctx, nil, nil, wf.DeepCopy(), nil, validate.Opts{}))
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	if setup != nil {
		setup(ctx, controller)
	}
	woc := r4Operate(t, ctx, controller, wf)
	for i := 0; i < 10 && !woc.wf.Status.Phase.Completed(); i++ {
		for _, n := range woc.wf.Status.Nodes {
			if outputsFor == nil || n.Type != wfv1.NodeTypePod || n.Fulfilled() {
				continue
			}
			if out := outputsFor(&n); out != nil {
				r4TaskResultOutputs(ctx, woc, n.Name, *out)
			}
		}
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Fulfilled() {
				return ""
			}
			return apiv1.PodSucceeded
		})
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	for range extra {
		woc = r4Operate(t, ctx, controller, woc.wf)
	}
	return woc
}

// r4RunResults validates manifest and reconciles it until the workflow
// completes or rounds run out, then extra more times. After each reconcile
// every unfinished pod is succeeded, a pod of a template named in results
// with that script result. setup, if not nil, runs on the controller before
// the first reconcile. It reconciles from the in-memory status, as the
// probes it ports did.
func r4RunResults(t *testing.T, manifest string, results map[string]string, rounds, extra int, setup func(*WorkflowController)) *wfOperationCtx {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(manifest)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	r4ValidateWithTemplates(ctx, t, controller, wf)
	if setup != nil {
		setup(controller)
	}
	withResult := func(pod *apiv1.Pod, w *wfOperationCtx) {
		node := w.wf.Status.Nodes[w.nodeID(pod)]
		if r, ok := results[node.TemplateName]; ok {
			withOutputs(ctx, wfv1.Outputs{Result: &r})(pod, w)
		}
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range rounds {
		woc.operate(ctx)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.Phase.Fulfilled(nil) {
				return ""
			}
			return apiv1.PodSucceeded
		}, withResult)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
	for range extra {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	dumpNodes(t, "final", woc.wf)
	return woc
}

// r4MemoCache is a memoization cache ConfigMap holding a hit for key "hit"
// with output p=value, exported as g (a cached output keeps its globalName,
// as the node outputs it was saved from carry it).
func r4MemoCache(name, value string) func(context.Context, *WorkflowController) {
	return func(ctx context.Context, controller *WorkflowController) {
		_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels:    map[string]string{common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache},
			},
			Data: map[string]string{
				"hit": `{"nodeID":"old","outputs":{"parameters":[{"name":"p","value":"` + value + `","globalName":"g"}]},"creationTimestamp":"2020-09-21T18:12:56Z"}`,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
	}
}

// r4GlobalParam is the value of the workflow output parameter g.
func r4GlobalParam(wf *wfv1.Workflow) string {
	if wf.Status.Outputs != nil {
		for _, p := range wf.Status.Outputs.Parameters {
			if p.Name == "g" && p.Value != nil {
				return p.Value.String()
			}
		}
	}
	return "<missing>"
}

// r4InputParam is the value of input parameter x of the node with the given
// display name.
func r4InputParam(wf *wfv1.Workflow, display string) string {
	if n := wf.Status.Nodes.FindByDisplayName(display); n != nil && n.Inputs != nil {
		for _, p := range n.Inputs.Parameters {
			if p.Name == "x" && p.Value != nil {
				return p.Value.String()
			}
		}
	}
	return "<missing>"
}

// r4GlobalOut is the output p, exported as the workflow output g.
// r4Restart replaces the controller with a new one built only from the
// cluster state, as after a controller restart: the workflow as stored, its
// pods, task results and ConfigMaps (memoization caches). newController's
// initManagers re-establishes recorded lock holders, as start-up does. The
// old controller is stopped; nothing it held in memory is carried over.
//
//nolint:revive // matches the r4 harness convention (t before ctx)
func r4Restart(t *testing.T, ctx context.Context, old *WorkflowController, stopOld context.CancelFunc, namespace, name string) (*WorkflowController, context.CancelFunc) {
	t.Helper()
	wf, err := old.wfclientset.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
	require.NoError(t, err)
	pods, err := old.kubeclientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	trs, err := old.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	cms, err := old.kubeclientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	stopOld()

	wf.ResourceVersion = ""
	cancel, controller := newController(ctx, wf)
	for i := range cms.Items {
		cm := cms.Items[i]
		cm.ResourceVersion = ""
		created, err := controller.kubeclientset.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metav1.CreateOptions{})
		require.NoError(t, err)
		// typedConfigMapInformer only watches configmaps carrying the
		// configmap-type label (memoization caches, and executor-plugin
		// configmaps): waiting on it for one without the label would block
		// until the timeout, since it would never appear in that informer's
		// store.
		if _, ok := created.Labels[common.LabelKeyConfigMapType]; ok {
			r4WaitForInformer(ctx, controller.typedConfigMapInformer, created, func(any) bool { return true })
		}
	}
	for i := range pods.Items {
		pod := pods.Items[i]
		pod.ResourceVersion = ""
		created, err := controller.kubeclientset.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		require.NoError(t, err)
		r4WaitForInformer(ctx, controller.PodController.TestingPodInformer(), created, func(any) bool { return true })
	}
	for i := range trs.Items {
		tr := trs.Items[i]
		tr.ResourceVersion = ""
		created, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(namespace).Create(ctx, &tr, metav1.CreateOptions{})
		require.NoError(t, err)
		r4WaitForInformer(ctx, controller.taskResultInformer, created, func(any) bool { return true })
	}
	return controller, cancel
}

// r4WaitForInformer gives waitForInformer (controller_test.go) a longer
// ceiling than its fixed 10s, for the restart helper above: r4Restart
// rebuilds a whole controller and its informers from scratch on every
// reconcile of TestRegressionR4_RestartBetweenReconciles, so it does many
// times what a normal test's single waitForInformer call does. Under a
// loaded machine (for example the full controller suite running
// concurrently) that adds up and 10s was once not enough, panicking the
// test.
//
// This file must still compile when overlaid onto wt-base (basecheck),
// where waitForInformer has no timeout parameter and must stay that way for
// every other caller, so this cannot add a parameter to it or call a
// shared helper with a longer timeout baked in — either would be a symbol
// this file depends on that wt-base's controller_test.go does not have.
// Instead of duplicating its poll loop, this just retries the unchanged
// waitForInformer (recovering the panic it raises on its own timeout)
// until a longer deadline. A synced informer still returns on
// waitForInformer's first attempt, so this does not slow the normal case.
func r4WaitForInformer(ctx context.Context, informer cache.SharedIndexInformer, obj any, upToDate func(obj any) bool) {
	deadline := time.Now().Add(time.Minute)
	for {
		if r4TryWaitForInformer(ctx, informer, obj, upToDate) {
			return
		}
		if time.Now().After(deadline) {
			waitForInformer(ctx, informer, obj, upToDate) // out of time: let it panic with its own message
			return
		}
	}
}

// r4TryWaitForInformer runs waitForInformer once and reports whether it
// caught up, recovering the panic it raises on its own 10s timeout instead
// of failing the test.
func r4TryWaitForInformer(ctx context.Context, informer cache.SharedIndexInformer, obj any, upToDate func(obj any) bool) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	waitForInformer(ctx, informer, obj, upToDate)
	return true
}

func r4GlobalOut(value string) *wfv1.Outputs {
	return &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", GlobalName: "g", Value: wfv1.AnyStringPtr(value)}}}
}

const r4GlobalTemplates = `
  - name: produce
    container:
      image: alpine
      command: [echo, hi]
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
      command: [echo, hi]
  - name: exit
    steps:
    - - name: e
        template: consume
        arguments:
          parameters:
          - name: x
            value: "{{workflow.outputs.parameters.g}}"
`
