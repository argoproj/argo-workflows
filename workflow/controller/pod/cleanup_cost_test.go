package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	argoConfig "github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wffake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
)

type cleanupCostCounts struct {
	Pod        map[string]int `json:"pod"`
	Workflow   map[string]int `json:"workflow"`
	Hydrate    int            `json:"hydrate"`
	Decompress int            `json:"decompress"`
	RepoGet    int            `json:"repoGet"`
	RepoSave   int            `json:"repoSave"`
	Attempts   int            `json:"attempts"`
	Signals    int            `json:"signals"`
}

type cleanupCostRepo struct {
	sqldb.OffloadNodeStatusRepo
	payload []byte
	rows    map[string][]byte
	failure error
	counts  *cleanupCostCounts
}

func (r *cleanupCostRepo) Get(_ context.Context, uid, version string) (wfv1.Nodes, error) {
	r.counts.RepoGet++
	if r.failure != nil {
		return nil, r.failure
	}
	payload := r.payload
	if version != "cost-v1" {
		var ok bool
		payload, ok = r.rows[version]
		if !ok {
			return nil, fmt.Errorf("unknown published reference %s", version)
		}
	}
	if uid != "956ecb34-5a75-44cc-b030-689f44286477" {
		return nil, fmt.Errorf("unexpected published identity %s/%s", uid, version)
	}
	var nodes wfv1.Nodes
	err := json.Unmarshal(payload, &nodes)
	return nodes, err
}
func (r *cleanupCostRepo) Save(_ context.Context, _, _ string, nodes wfv1.Nodes) (string, error) {
	r.counts.RepoSave++
	var err error
	r.payload, err = json.Marshal(nodes)
	return "cost-v1", err
}
func (r *cleanupCostRepo) IsEnabled() bool { return true }

type cleanupCostHarness struct {
	t        *testing.T
	c        *Controller
	client   *fake.Clientset
	wfClient *wffake.Clientset
	clock    *clocktesting.FakeClock
	counts   cleanupCostCounts
	repo     *cleanupCostRepo
	hydrate  hydrator.Interface
}

func newCleanupCostHarness(t *testing.T, wf *wfv1.Workflow, pods []*apiv1.Pod, storage string, absent bool) *cleanupCostHarness {
	t.Helper()
	h := &cleanupCostHarness{t: t, clock: clocktesting.NewFakeClock(time.Unix(10000, 0))}
	h.repo = &cleanupCostRepo{counts: &h.counts}
	var err error
	h.repo.payload, err = json.Marshal(wf.Status.Nodes)
	require.NoError(t, err)
	encoded := wf.DeepCopy()
	switch storage {
	case "compressed":
		encoded.Status.CompressedNodes = file.CompressEncodeString(logging.TestContext(t.Context()), string(h.repo.payload))
		encoded.Status.Nodes = nil
	case "offload":
		encoded.Status.OffloadNodeStatusVersion = "cost-v1"
		encoded.Status.Nodes = nil
	}
	h.hydrate = hydrator.New(h.repo)
	if absent {
		h.wfClient = wffake.NewClientset()
	} else {
		h.wfClient = wffake.NewClientset(encoded)
	}
	objects := make([]runtime.Object, 0, len(pods))
	for _, p := range pods {
		objects = append(objects, p)
	}
	h.client = fake.NewSimpleClientset(objects...)
	wfInformer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
	if !absent {
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(encoded)
		require.NoError(t, err)
		require.NoError(t, wfInformer.GetStore().Add(&unstructured.Unstructured{Object: u}))
	}
	podInformer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &apiv1.Pod{}, 0, cache.Indexers{})
	for _, p := range pods {
		require.NoError(t, podInformer.GetStore().Add(p))
	}
	h.c = &Controller{config: &argoConfig.Config{PodGCDeleteDelayDuration: &metav1.Duration{}}, kubeclientset: h.client, wfInformer: wfInformer, podInformer: podInformer,
		workqueue: workqueue.NewTypedRateLimitingQueueWithConfig(workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{Clock: h.clock}),
		callBack:  func(*apiv1.Pod) error { return nil }, log: logging.RequireLoggerFromContext(logging.TestContext(t.Context()))}
	t.Cleanup(h.c.workqueue.ShutDown)
	cleanupCostInstall(h)
	return h
}

func (h *cleanupCostHarness) lookup(ctx context.Context, namespace, name string, hydrate bool) (*wfv1.Workflow, error) {
	wf, err := h.wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil && hydrate {
		err = h.hydrateWorkflow(ctx, wf)
	}
	return wf, err
}
func (h *cleanupCostHarness) hydrateWorkflow(ctx context.Context, wf *wfv1.Workflow) error {
	h.counts.Hydrate++
	if wf.Status.CompressedNodes != "" {
		h.counts.Decompress++
	}
	return h.hydrate.Hydrate(ctx, wf)
}

// Only the scheduler clock advances. Every ready item passes through the real
// worker; emitted follow-up actions are drained as well, with no queue shortcut.
func (h *cleanupCostHarness) drain() {
	h.t.Helper()
	for range 5 {
		time.Sleep(2 * time.Millisecond)
		h.clock.Step(20 * time.Millisecond)
		time.Sleep(2 * time.Millisecond)
		for h.c.workqueue.Len() > 0 {
			h.counts.Attempts++
			require.Less(h.t, h.counts.Attempts, 10000)
			require.True(h.t, h.c.processNextPodCleanupItem(logging.TestContext(h.t.Context())))
		}
	}
}
func (h *cleanupCostHarness) snapshot() cleanupCostCounts {
	x := h.counts
	x.Pod = map[string]int{}
	x.Workflow = map[string]int{}
	for _, a := range h.client.Actions() {
		x.Pod[a.GetVerb()]++
	}
	for _, a := range h.wfClient.Actions() {
		x.Workflow[a.GetVerb()]++
	}
	return x
}
func (h *cleanupCostHarness) pod(p *apiv1.Pod) *apiv1.Pod {
	obj, err := h.client.Tracker().Get(apiv1.SchemeGroupVersion.WithResource("pods"), p.Namespace, p.Name)
	if err != nil {
		return nil
	}
	return obj.(*apiv1.Pod)
}
func cleanupCostPod(wf *wfv1.Workflow, index int) *apiv1.Pod {
	name := fmt.Sprintf("%s.task-%04d", wf.Name, index)
	nodeID := wf.NodeID(name)
	return &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: wf.Namespace, Name: fmt.Sprintf("cleanup-pod-%04d", index), UID: types.UID(fmt.Sprintf("pod-%04d", index)), ResourceVersion: "10",
		Labels:          map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyCompleted: "false"},
		Annotations:     map[string]string{common.AnnotationKeyNodeID: nodeID, common.AnnotationKeyNodeName: name},
		OwnerReferences: []metav1.OwnerReference{{APIVersion: wfv1.SchemeGroupVersion.String(), Kind: "Workflow", Name: wf.Name, UID: wf.UID, Controller: new(true)}}}, Status: apiv1.PodStatus{Phase: apiv1.PodRunning}}
}
func cleanupCostReceipt(t *testing.T, node wfv1.NodeStatus, uid types.UID) wfv1.NodeStatus {
	t.Helper()
	raw, err := json.Marshal(node)
	require.NoError(t, err)
	var object map[string]any
	require.NoError(t, json.Unmarshal(raw, &object))
	object["capturedPodUID"] = string(uid)
	raw, err = json.Marshal(object)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &node))
	return node
}
func cleanupCostReport(t *testing.T, scenario, storage string, wf *wfv1.Workflow, pods []*apiv1.Pod, h *cleanupCostHarness, cycle string) {
	t.Helper()
	nodes, _ := json.Marshal(wf.Status.Nodes)
	payload, _ := json.Marshal(wf)
	current, _ := h.wfClient.Tracker().Get(wfv1.SchemeGroupVersion.WithResource("workflows"), wf.Namespace, wf.Name)
	encodedBytes := 0
	if current != nil {
		b, _ := json.Marshal(current)
		encodedBytes = len(b)
	}
	events := len(pods)
	sweeps := 0
	if scenario == "100-add-events" {
		events = 100
	}
	if scenario == "completion-sweep" {
		events = 0
		sweeps = 1
	}
	row := map[string]any{"flagEnabled": os.Getenv(common.EnvVarPodStatusCaptureFinalizer) == "true", "events": events, "sweeps": sweeps, "scenario": scenario, "storage": storage, "nodes": len(wf.Status.Nodes), "pods": len(pods), "workflows": 1, "nodeBytes": len(nodes), "rawWorkflowBytes": len(payload), "storedWorkflowBytes": encodedBytes, "cycle": cycle, "counts": h.snapshot()}
	outcomes := []map[string]any{}
	for _, pod := range pods {
		actual := h.pod(pod)
		out := map[string]any{"uid": pod.UID, "exists": actual != nil}
		if actual != nil {
			out["barrier"] = hasOurFinalizer(actual.Finalizers)
			out["completed"] = actual.Labels[common.LabelKeyCompleted]
			out["phase"] = actual.Status.Phase
		}
		outcomes = append(outcomes, out)
	}
	row["outcomes"] = outcomes
	b, err := json.Marshal(row)
	require.NoError(t, err)
	t.Log("COST_RESULT " + string(b))
}

func TestCleanupCostEventMatrix(t *testing.T) {
	for _, size := range []int{32, 1024} {
		for _, storage := range []string{"raw", "compressed", "offload"} {
			for _, enabled := range []bool{false, true} {
				for _, phase := range []apiv1.PodPhase{apiv1.PodRunning, apiv1.PodPending} {
					for _, event := range []string{"add", "update"} {
						scenario := fmt.Sprintf("active/%d/%s/flag=%t/%s/%s", size, storage, enabled, phase, event)
						t.Run(scenario, func(t *testing.T) {
							t.Setenv(common.EnvVarPodStatusCaptureFinalizer, fmt.Sprint(enabled))
							wf := cleanupCostFixture(size)
							p := cleanupCostPod(wf, 0)
							p.Status.Phase = phase
							if enabled {
								p.Finalizers = []string{common.FinalizerPodStatus}
							}
							node := wf.Status.Nodes[p.Annotations[common.AnnotationKeyNodeID]]
							node.Phase = wfv1.NodePhase(phase)
							wf.Status.Nodes[node.ID] = node
							h := newCleanupCostHarness(t, wf, []*apiv1.Pod{p}, storage, false)
							callbacks := 0
							h.c.callBack = func(*apiv1.Pod) error { callbacks++; return nil }
							if event == "add" {
								h.c.addPodEvent(logging.TestContext(t.Context()), p)
							} else {
								old := p.DeepCopy()
								old.ResourceVersion = "9"
								old.Status.Phase = apiv1.PodUnknown
								require.True(t, significantPodChange(old, p))
								h.c.updatePodEvent(logging.TestContext(t.Context()), old, p)
							}
							h.drain()
							require.Equal(t, 1, callbacks, "workflow reconciliation must still receive ordinary live events")
							initial := h.snapshot()
							require.NotNil(t, h.pod(p))
							require.Equal(t, p.Finalizers, h.pod(p).Finalizers)
							require.Zero(t, initial.Pod["patch"]+initial.Pod["delete"])
							h.clock.Step(time.Minute)
							h.drain()
							require.Equal(t, initial, h.snapshot(), "known active state must not become permanent polling")
							cleanupCostReport(t, scenario, storage, wf, []*apiv1.Pod{p}, h, "initial+idle-minute")
							cleanupCostActiveBudget(t, initial)
							if event == "update" {
								require.Zero(t, initial.Attempts, "ordinary live updates only need workflow reconciliation")
								require.Zero(t, initial.Pod["get"])
								require.Zero(t, initial.Workflow["get"])
								require.Zero(t, initial.Hydrate)
							}
						})
					}
				}
			}
		}
	}
}

func TestCleanupCostUsefulRecovery(t *testing.T) {
	for _, storage := range []string{"raw", "compressed", "offload"} {
		for _, enabled := range []bool{false, true} {
			for _, scenario := range []string{"terminal", "orphan", "deleting-owner", "deleting-pending", "retirement", "daemon", "cancellation", "gc-delay", "lost-delete", "backlog"} {
				t.Run(fmt.Sprintf("%s/%s/flag=%t", scenario, storage, enabled), func(t *testing.T) {
					t.Setenv(common.EnvVarPodStatusCaptureFinalizer, fmt.Sprint(enabled))
					wf := cleanupCostFixture(32)
					p := cleanupCostPod(wf, 0)
					p.Finalizers = []string{"example.com/keep"}
					if enabled {
						p.Finalizers = append(p.Finalizers, common.FinalizerPodStatus)
					}
					node := wf.Status.Nodes[p.Annotations[common.AnnotationKeyNodeID]]
					node.Phase = wfv1.NodeSucceeded
					node.TaskResultSynced = new(true)
					node = cleanupCostReceipt(t, node, p.UID)
					p.Status.Phase = apiv1.PodSucceeded
					switch scenario {
					case "orphan":
						p.Annotations[common.AnnotationKeyPodGCStrategy] = "OnPodCompletion/0s"
					case "deleting-owner":
						p.Annotations[common.AnnotationKeyPodGCStrategy] = "OnPodCompletion/0s"
						wf.DeletionTimestamp = &metav1.Time{Time: time.Unix(9000, 0)}
					case "deleting-pending":
						p.Status.Phase = apiv1.PodPending
						p.DeletionTimestamp = &metav1.Time{Time: time.Unix(9000, 0)}
					case "retirement":
						p.Status.Phase = apiv1.PodFailed
						node.Phase = wfv1.NodePending
						node.RestartingPodUID = string(p.UID)
					case "daemon", "cancellation":
						p.Status.Phase = apiv1.PodRunning
						p.Spec.TerminationGracePeriodSeconds = new(int64(1))
						p.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: "main", State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}}}
						if scenario == "cancellation" {
							node.Phase = wfv1.NodeFailed
						}
					case "gc-delay", "lost-delete":
						wf.Spec.PodGC = &wfv1.PodGC{Strategy: wfv1.PodGCOnPodCompletion, DeleteDelayDuration: "5s"}
						if scenario == "lost-delete" {
							p.Finalizers = []string{"example.com/keep"}
						}
					case "backlog":
						node = cleanupCostReceipt(t, node, "")
					}
					wf.Status.Nodes[node.ID] = node
					h := newCleanupCostHarness(t, wf, []*apiv1.Pod{p}, storage, scenario == "orphan")
					h.c.addPodEvent(logging.TestContext(t.Context()), p)
					h.drain()
					cleanupCostReport(t, scenario, storage, wf, []*apiv1.Pod{p}, h, "initial")
					switch scenario {
					case "gc-delay", "lost-delete":
						require.NotNil(t, h.pod(p))
						h.clock.Step(5 * time.Second)
						h.drain()
					case "daemon", "cancellation":
						h.clock.Step(time.Second)
						h.drain()
					case "backlog":
						before := h.snapshot()
						h.clock.Step(30 * time.Second)
						h.drain()
						if enabled && cleanupCostRecoveryAvailable() {
							require.Greater(t, h.snapshot().Attempts, before.Attempts)
							require.Contains(t, h.pod(p).Finalizers, common.FinalizerPodStatus)
						}
					}
					cleanupCostUsefulOutcome(t, scenario, enabled, h, p)
					cleanupCostUsefulBudget(t, scenario, h.snapshot())
					cleanupCostReport(t, scenario, storage, wf, []*apiv1.Pod{p}, h, "complete-or-one-retry")
				})
			}
		}
	}
}

// This default is replaced only in immutable-version overlays. The production
// test remains strict; base comparisons explicitly record missing guarantees.
func cleanupCostUsefulOutcome(t *testing.T, scenario string, enabled bool, h *cleanupCostHarness, p *apiv1.Pod) {
	t.Helper()
	if !cleanupCostRecoveryAvailable() {
		return
	}
	current := h.pod(p)
	switch scenario {
	case "orphan", "deleting-owner", "retirement", "gc-delay", "lost-delete":
		require.Nil(t, current, "cleanup must actually delete exact target")
	case "daemon", "cancellation":
		require.GreaterOrEqual(t, h.snapshot().Attempts, 3, "recovery must deliver TERM and delayed KILL")
		require.Equal(t, 2, h.snapshot().Signals, "both TERM and KILL must be delivered")
	case "backlog":
		if enabled {
			require.Contains(t, current.Finalizers, common.FinalizerPodStatus)
		}
	default:
		require.NotNil(t, current)
		if enabled {
			require.NotContains(t, current.Finalizers, common.FinalizerPodStatus)
		}
		require.Contains(t, current.Finalizers, "example.com/keep")
	}
	if scenario == "terminal" {
		require.Equal(t, "true", current.Labels[common.LabelKeyCompleted])
	}
}

func cleanupCostFixture(podNodes int) *wfv1.Workflow {
	started := metav1.NewTime(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	wf := &wfv1.Workflow{
		TypeMeta: metav1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "hydration-cost", Namespace: "default",
			UID: "956ecb34-5a75-44cc-b030-689f44286477", ResourceVersion: "1",
			CreationTimestamp: started,
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{Name: "main", DAG: &wfv1.DAGTemplate{}},
				{Name: "worker", Container: &apiv1.Container{Image: "alpine:3.21", Command: []string{"true"}}},
			},
		},
		Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, StartedAt: started, Nodes: wfv1.Nodes{}},
	}
	root := wfv1.NodeStatus{
		ID: wf.Name, Name: wf.Name, DisplayName: wf.Name, Type: wfv1.NodeTypeDAG,
		TemplateName: "main", TemplateScope: "local/" + wf.Name,
		Phase: wfv1.NodeRunning, StartedAt: started,
	}
	for i := range podNodes {
		task := fmt.Sprintf("task-%04d", i)
		name := wf.Name + "." + task
		id := wf.NodeID(name)
		message := fmt.Sprintf("node-%06d:", i)
		message += strings.Repeat("m", 256-len(message))
		parameter := fmt.Sprintf("output-%06d:", i)
		parameter += strings.Repeat("v", 128-len(parameter))
		wf.Spec.Templates[0].DAG.Tasks = append(wf.Spec.Templates[0].DAG.Tasks, wfv1.DAGTask{Name: task, Template: "worker"})
		root.Children = append(root.Children, id)
		wf.Status.Nodes[id] = wfv1.NodeStatus{
			ID: id, Name: name, DisplayName: task, Type: wfv1.NodeTypePod,
			TemplateName: "worker", TemplateScope: "local/" + wf.Name, BoundaryID: root.ID,
			Phase: wfv1.NodeRunning, StartedAt: started, Message: message,
			Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "payload", Value: wfv1.AnyStringPtr(parameter)}}},
		}
	}
	wf.Status.Nodes[root.ID] = root
	return wf
}

func TestCleanupCostSharedBacklog(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, size := range []int{32, 1024} {
		for _, storage := range []string{"compressed", "offload"} {
			for _, retained := range []int{1, 16} {
				t.Run(fmt.Sprintf("nodes=%d/%s/retained=%d", size, storage, retained), func(t *testing.T) {
					wf := cleanupCostFixture(size)
					pods := make([]*apiv1.Pod, 0, retained)
					for i := range retained {
						p := cleanupCostPod(wf, i)
						p.Status.Phase = apiv1.PodSucceeded
						p.Finalizers = []string{common.FinalizerPodStatus}
						node := wf.Status.Nodes[p.Annotations[common.AnnotationKeyNodeID]]
						node.Phase = wfv1.NodeSucceeded
						node.TaskResultSynced = new(true)
						wf.Status.Nodes[node.ID] = node
						pods = append(pods, p)
					}
					h := newCleanupCostHarness(t, wf, pods, storage, false)
					for _, p := range pods {
						h.c.addPodEvent(logging.TestContext(t.Context()), p)
					}
					h.drain()
					cleanupCostReport(t, "shared-backlog", storage, wf, pods, h, "initial")
					first := h.snapshot()
					h.clock.Step(30 * time.Second)
					h.drain()
					second := h.snapshot()
					for _, p := range pods {
						require.Contains(t, h.pod(p).Finalizers, common.FinalizerPodStatus)
					}
					require.Zero(t, second.Pod["patch"]+second.Pod["delete"])
					if cleanupCostRecoveryAvailable() {
						require.Equal(t, retained, second.Attempts-first.Attempts)
					}
					cleanupCostReport(t, "shared-backlog", storage, wf, pods, h, "initial+one-stable-retry")
				})
			}
		}
	}
}

func TestCleanupCostSweepAndCoalescing(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, sweep := range []bool{false, true} {
		t.Run(fmt.Sprintf("sweep=%t", sweep), func(t *testing.T) {
			wf := cleanupCostFixture(32)
			p := cleanupCostPod(wf, 0)
			p.Status.Phase = apiv1.PodSucceeded
			p.Finalizers = []string{common.FinalizerPodStatus}
			node := wf.Status.Nodes[p.Annotations[common.AnnotationKeyNodeID]]
			node.Phase = wfv1.NodeSucceeded
			node.TaskResultSynced = new(true)
			node = cleanupCostReceipt(t, node, p.UID)
			wf.Status.Nodes[node.ID] = node
			h := newCleanupCostHarness(t, wf, []*apiv1.Pod{p}, "offload", false)
			scenario := "100-add-events"
			if sweep {
				scenario = "completion-sweep"
				cleanupCostSweep(logging.TestContext(t.Context()), h.c, wf)
			} else {
				for range 100 {
					h.c.addPodEvent(logging.TestContext(t.Context()), p)
				}
			}
			h.drain()
			// The100 source events also consume the default global token bucket;
			// allow the separately emitted mutation to reach its real deadline.
			h.clock.Step(time.Second)
			h.drain()
			if cleanupCostRecoveryAvailable() {
				require.Equal(t, "true", h.pod(p).Labels[common.LabelKeyCompleted])
				require.NotContains(t, h.pod(p).Finalizers, common.FinalizerPodStatus)
			}
			cleanupCostReport(t, scenario, "offload", wf, []*apiv1.Pod{p}, h, "complete")
		})
	}
}
