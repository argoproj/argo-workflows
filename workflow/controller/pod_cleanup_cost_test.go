package controller

import (
	"context"
	"encoding/json"
	"fmt"
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
	podcontroller "github.com/argoproj/argo-workflows/v4/workflow/controller/pod"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
)

type legacyCostStats struct{ Hydrate, Decompress, RepoGet, RepoSave, Attempts int }
type legacyCostRepo struct {
	sqldb.OffloadNodeStatusRepo
	uid   string
	rows  map[string][]byte
	stats *legacyCostStats
}

func (r *legacyCostRepo) Get(_ context.Context, uid, version string) (wfv1.Nodes, error) {
	r.stats.RepoGet++
	if uid != r.uid {
		return nil, fmt.Errorf("wrong owner")
	}
	data, ok := r.rows[version]
	if !ok {
		return nil, fmt.Errorf("wrong published reference %s", version)
	}
	var nodes wfv1.Nodes
	err := json.Unmarshal(data, &nodes)
	return nodes, err
}
func (r *legacyCostRepo) Save(_ context.Context, uid, _ string, nodes wfv1.Nodes) (string, error) {
	r.stats.RepoSave++
	if uid != r.uid {
		return "", fmt.Errorf("wrong save owner")
	}
	data, err := json.Marshal(nodes)
	version := fmt.Sprintf("cost-saved-%d", r.stats.RepoSave)
	r.rows[version] = data
	return version, err
}
func (r *legacyCostRepo) IsEnabled() bool { return true }

type legacyCostHydrator struct {
	hydrator.Interface
	stats *legacyCostStats
}

func (h *legacyCostHydrator) Hydrate(ctx context.Context, wf *wfv1.Workflow) error {
	h.stats.Hydrate++
	if wf.Status.CompressedNodes != "" {
		h.stats.Decompress++
	}
	return h.Interface.Hydrate(ctx, wf)
}

func TestLegacyCleanupCostEvent(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, storage := range []string{"raw", "compressed", "offload"} {
		for _, count := range []int{1, 32, 1024} {
			for _, retained := range []int{1, 16} {
				if count == 1 && retained != 1 {
					continue
				}
				t.Run(fmt.Sprintf("%s/nodes=%d/retained=%d", storage, count, retained), func(t *testing.T) {
					ctx := logging.TestContext(t.Context())
					var wf *wfv1.Workflow
					var pods []*apiv1.Pod
					if count == 1 {
						var p *apiv1.Pod
						wf, p = legacyCaptureFixture(t)
						p.Finalizers = []string{"example.com/keep", common.FinalizerPodStatus}
						pods = []*apiv1.Pod{p}
					} else {
						wf = legacyCostWorkflow(count)
						wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
						wf.Status.Phase = wfv1.WorkflowSucceeded
						for id, node := range wf.Status.Nodes {
							node.Phase = wfv1.NodeSucceeded
							node.TaskResultSynced = new(true)
							wf.Status.Nodes[id] = node
						}
						for i := range retained {
							name := fmt.Sprintf("%s.task-%04d", wf.Name, i)
							pods = append(pods, &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: wf.Namespace, Name: fmt.Sprintf("cleanup-pod-%04d", i), UID: types.UID(fmt.Sprintf("pod-%04d", i)), ResourceVersion: "10", Labels: map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyCompleted: "false"}, Annotations: map[string]string{common.AnnotationKeyNodeID: wf.NodeID(name), common.AnnotationKeyNodeName: name}, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind("Workflow"))}, Finalizers: []string{"example.com/keep", common.FinalizerPodStatus}}, Status: apiv1.PodStatus{Phase: apiv1.PodSucceeded}})
						}
					}
					original := wf.DeepCopy()
					stats := &legacyCostStats{}
					nodeJSON, err := json.Marshal(wf.Status.Nodes)
					require.NoError(t, err)
					repo := &legacyCostRepo{uid: string(wf.UID), rows: map[string][]byte{"cost-v1": nodeJSON}, stats: stats}
					if storage == "offload" {
						t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "true")
					} else {
						t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
					}
					h := &legacyCostHydrator{Interface: hydrator.New(repo), stats: stats}
					encoded := wf.DeepCopy()
					switch storage {
					case "compressed":
						encoded.Status.CompressedNodes = file.CompressEncodeString(ctx, string(nodeJSON))
						encoded.Status.Nodes = nil
					case "offload":
						encoded.Status.OffloadNodeStatusVersion = "cost-v1"
						encoded.Status.Nodes = nil
					}
					objects := make([]runtime.Object, 0, len(pods))
					for _, p := range pods {
						objects = append(objects, p)
					}
					kube := fake.NewSimpleClientset(objects...)
					wfClient := wffake.NewClientset(encoded)
					controller := &WorkflowController{kubeclientset: kube, wfclientset: wfClient, hydrator: h}
					wfInformer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
					metric, _, err := metrics.CreateDefaultTestMetrics(ctx)
					require.NoError(t, err)
					c := podcontroller.NewController(ctx, &argoConfig.Config{PodGCDeleteDelayDuration: &metav1.Duration{}}, nil, "default", kube, wfInformer, metric, func(*apiv1.Pod) error { return nil }, controller.lookupWorkflowForPodCleanup)
					c.SetLegacyPodRecapture(controller.recaptureLegacyPod)
					legacyCostInstallHydration(c, controller)
					clock := clocktesting.NewFakeClock(time.Unix(10000, 0))
					q := workqueue.NewTypedRateLimitingQueueWithConfig(workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{Clock: clock})
					c.TestingSetQueue(q)
					t.Cleanup(q.ShutDown)
					drain := func() {
						for range 5 {
							time.Sleep(2 * time.Millisecond)
							clock.Step(20 * time.Millisecond)
							time.Sleep(2 * time.Millisecond)
							for c.TestingQueueLen() > 0 {
								stats.Attempts++
								require.Less(t, stats.Attempts, 10000)
								require.True(t, c.TestingProcessNextItem(ctx))
							}
						}
					}
					for _, p := range pods {
						c.TestingPodEvent(ctx, p, false)
					}
					drain()
					snapshot := func(cycle string) {
						podOps, wfOps := map[string]int{}, map[string]int{}
						for _, a := range kube.Actions() {
							podOps[a.GetVerb()]++
						}
						for _, a := range wfClient.Actions() {
							wfOps[a.GetVerb()]++
						}
						storedObj, e := wfClient.Tracker().Get(wfv1.SchemeGroupVersion.WithResource("workflows"), wf.Namespace, wf.Name)
						require.NoError(t, e)
						stored := storedObj.(*wfv1.Workflow)
						raw, _ := json.Marshal(wf)
						storedJSON, _ := json.Marshal(stored)
						outcomes := []map[string]any{}
						for _, p := range pods {
							obj, podErr := kube.Tracker().Get(apiv1.SchemeGroupVersion.WithResource("pods"), p.Namespace, p.Name)
							require.NoError(t, podErr)
							actual := obj.(*apiv1.Pod)
							outcomes = append(outcomes, map[string]any{"uid": actual.UID, "finalizers": actual.Finalizers, "completed": actual.Labels[common.LabelKeyCompleted]})
						}
						row := map[string]any{"flagEnabled": true, "events": len(pods), "scenario": "legacy", "storage": storage, "nodes": len(wf.Status.Nodes), "pods": retained, "workflows": 1, "nodeBytes": len(nodeJSON), "rawWorkflowBytes": len(raw), "storedWorkflowBytes": len(storedJSON), "cycle": cycle, "pod": podOps, "workflow": wfOps, "counts": *stats, "outcomes": outcomes, "storedCompressed": stored.Status.CompressedNodes != "", "storedOffload": stored.Status.IsOffloadNodeStatus()}
						b, e := json.Marshal(row)
						require.NoError(t, e)
						t.Log("COST_RESULT " + string(b))
					}
					snapshot("initial")
					if count != 1 {
						before := stats.Attempts
						clock.Step(30 * time.Second)
						drain()
						require.Equal(t, retained, stats.Attempts-before)
						require.Zero(t, stats.RepoSave)
						for _, a := range wfClient.Actions() {
							require.NotEqual(t, "update", a.GetVerb())
						}
						snapshot("initial+one-stable-retry")
					}
					// Outcome read uses the tracker and a separate hydrator to avoid counting
					// verification itself as cleanup work.
					obj, err := wfClient.Tracker().Get(wfv1.SchemeGroupVersion.WithResource("workflows"), wf.Namespace, wf.Name)
					require.NoError(t, err)
					actualWF := obj.(*wfv1.Workflow).DeepCopy()
					verifyRepo := &legacyCostRepo{uid: repo.uid, rows: repo.rows, stats: &legacyCostStats{}}
					require.NoError(t, hydrator.New(verifyRepo).Hydrate(ctx, actualWF))
					for _, p := range pods {
						obj, e := kube.Tracker().Get(apiv1.SchemeGroupVersion.WithResource("pods"), p.Namespace, p.Name)
						require.NoError(t, e)
						actual := obj.(*apiv1.Pod)
						if count == 1 {
							require.NotContains(t, actual.Finalizers, common.FinalizerPodStatus)
							node := actualWF.Status.Nodes[wf.Name]
							require.Equal(t, string(p.UID), node.CapturedPodUID)
							node.CapturedPodUID = ""
							actualWF.Status.Nodes[wf.Name] = node
						} else {
							require.Contains(t, actual.Finalizers, common.FinalizerPodStatus)
						}
					}
					beforeStatus, e := json.Marshal(original.Status)
					require.NoError(t, e)
					afterStatus, e := json.Marshal(actualWF.Status)
					require.NoError(t, e)
					require.JSONEq(t, string(beforeStatus), string(afterStatus), "recapture may only add receipt; held legacy must keep exact serialized result")
					if count == 1 {
						if storage == "offload" {
							require.Equal(t, 1, stats.RepoSave, "supported offload recapture saves one immutable node version")
						} else {
							require.Zero(t, stats.RepoSave, "raw/compressed recapture must not introduce an offload write")
						}
					}
					legacyCostBudget(t, count, retained, stats, kube, wfClient)
					// This exported input is replayed unchanged by the base overlay, whose event
					// path has no legacy recapture guarantee. No private files are needed in-tree.
					fixtureJSON, e := json.Marshal(map[string]any{"workflow": original, "pods": pods, "storage": storage})
					require.NoError(t, e)
					t.Log("COST_FIXTURE " + string(fixtureJSON))
				})
			}
		}
	}
}

func legacyCostInstallHydration(c *podcontroller.Controller, controller *WorkflowController) {
	c.SetWorkflowHydrator(controller.hydrateWorkflowForPodCleanup)
}
func legacyCostWorkflow(podNodes int) *wfv1.Workflow {
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

func legacyCostBudget(t *testing.T, count, retained int, stats *legacyCostStats, kube *fake.Clientset, wfClient *wffake.Clientset) {
	t.Helper()
	podGets, wfGets, updates := 0, 0, 0
	for _, a := range kube.Actions() {
		if a.GetVerb() == "get" {
			podGets++
		}
	}
	for _, a := range wfClient.Actions() {
		if a.GetVerb() == "get" {
			wfGets++
		}
		if a.GetVerb() == "update" {
			updates++
		}
	}
	if count == 1 {
		require.LessOrEqual(t, podGets, 3)
		require.LessOrEqual(t, wfGets, 5)
		require.LessOrEqual(t, stats.Hydrate, 5)
		require.Equal(t, 1, updates)
	} else {
		require.LessOrEqual(t, podGets, 2*retained)
		require.LessOrEqual(t, wfGets, 2*retained)
		require.LessOrEqual(t, stats.Hydrate, 2*retained)
		require.Zero(t, updates)
	}
}

func TestCleanupCostHydratorReload(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	wf, p := legacyCaptureFixture(t)
	node := wf.Status.Nodes[wf.Name]
	node.CapturedPodUID = string(p.UID)
	wf.Status.Nodes[node.ID] = node
	p.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
	data, err := json.Marshal(wf.Status.Nodes)
	require.NoError(t, err)
	wf.Status.Nodes = nil
	wf.Status.OffloadNodeStatusVersion = "cost-v1"
	oldStats, newStats := &legacyCostStats{}, &legacyCostStats{}
	oldRepo := &legacyCostRepo{uid: string(wf.UID), rows: map[string][]byte{}, stats: oldStats}
	newRepo := &legacyCostRepo{uid: string(wf.UID), rows: map[string][]byte{"cost-v1": data}, stats: newStats}
	kube := fake.NewSimpleClientset(p)
	wfClient := wffake.NewClientset(wf)
	controller := &WorkflowController{kubeclientset: kube, wfclientset: wfClient, hydrator: &legacyCostHydrator{Interface: hydrator.New(oldRepo), stats: oldStats}}
	wfInformer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
	metric, _, err := metrics.CreateDefaultTestMetrics(ctx)
	require.NoError(t, err)
	c := podcontroller.NewController(ctx, &argoConfig.Config{PodGCDeleteDelayDuration: &metav1.Duration{}}, nil, "default", kube, wfInformer, metric, func(*apiv1.Pod) error { return nil }, controller.lookupWorkflowForPodCleanup)
	c.SetWorkflowHydrator(controller.hydrateWorkflowForPodCleanup)
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.NewTypedItemExponentialFailureRateLimiter[string](0, 0))
	c.TestingSetQueue(queue)
	t.Cleanup(queue.ShutDown)
	// Config reload replaces the interface after callbacks have been installed.
	controller.hydrator = &legacyCostHydrator{Interface: hydrator.New(newRepo), stats: newStats}
	c.TestingPodEvent(ctx, p, false)
	require.Eventually(t, func() bool { return c.TestingQueueLen() > 0 }, time.Second, time.Millisecond)
	require.True(t, c.TestingProcessNextItem(ctx))
	require.Eventually(t, func() bool { return c.TestingQueueLen() > 0 }, time.Second, time.Millisecond)
	require.True(t, c.TestingProcessNextItem(ctx))
	require.Zero(t, oldStats.Hydrate)
	require.Equal(t, 2, newStats.Hydrate)
	require.Equal(t, 2, newStats.RepoGet)
	current, err := kube.Tracker().Get(apiv1.SchemeGroupVersion.WithResource("pods"), p.Namespace, p.Name)
	require.NoError(t, err)
	require.Equal(t, []string{"example.com/keep"}, current.(*apiv1.Pod).Finalizers)
}
