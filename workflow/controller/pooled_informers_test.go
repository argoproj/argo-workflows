package controller

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

// filterWatches makes a fake client's watches apply their label selector, as
// the API server's do (the fakes apply it only to lists), and returns a
// channel that receives once each watch is registered, so a test can tell
// when changes will reach an informer.
func filterWatches(fake *k8stesting.Fake, tracker k8stesting.ObjectTracker) <-chan struct{} {
	started := make(chan struct{}, 16)
	fake.PrependWatchReactor("*", func(action k8stesting.Action) (bool, watch.Interface, error) {
		w, err := tracker.Watch(action.GetResource(), action.GetNamespace())
		if err != nil {
			return false, nil, err
		}
		selector := labels.Everything()
		if a, ok := action.(k8stesting.WatchAction); ok && a.GetWatchRestrictions().Labels != nil {
			selector = a.GetWatchRestrictions().Labels
		}
		started <- struct{}{}
		return true, watch.Filter(w, func(e watch.Event) (watch.Event, bool) {
			m, err := meta.Accessor(e.Object)
			return e, err != nil || selector.Matches(labels.Set(m.GetLabels()))
		}), nil
	})
	return started
}

// metadataTaskSetListWatch serves the typed fake's TaskSets as metadata, as
// the API server serves a metadata informer, so TaskSets created through the
// typed client reach a pooled TaskSet informer in tests. It applies the label
// selector to watches as well as lists.
func metadataTaskSetListWatch(client wfclientset.Interface, selector string) cache.ListerWatcher {
	parsed := labels.SelectorFromSet(nil)
	if s, err := labels.Parse(selector); err == nil {
		parsed = s
	}
	taskSets := client.ArgoprojV1alpha1().WorkflowTaskSets("")
	return cache.ToListWatcherWithWatchListSemantics(&cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = selector
			list, err := taskSets.List(ctx, options)
			if err != nil {
				return nil, err
			}
			out := &metav1.PartialObjectMetadataList{ListMeta: list.ListMeta}
			for i := range list.Items {
				out.Items = append(out.Items, *taskSetMetadata(&list.Items[i]))
			}
			return out, nil
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = selector
			w, err := taskSets.Watch(ctx, options)
			if err != nil {
				return nil, err
			}
			return watch.Filter(w, func(e watch.Event) (watch.Event, bool) {
				taskSet, ok := e.Object.(*wfv1.WorkflowTaskSet)
				if !ok {
					return e, true
				}
				e.Object = taskSetMetadata(taskSet)
				return e, parsed.Matches(labels.Set(taskSet.Labels))
			}), nil
		},
	}, client)
}

func taskSetMetadata(taskSet *wfv1.WorkflowTaskSet) *metav1.PartialObjectMetadata {
	return &metav1.PartialObjectMetadata{
		TypeMeta:   metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskSetKind},
		ObjectMeta: *taskSet.ObjectMeta.DeepCopy(),
	}
}

// newTestPooledTaskSetInformer is newPooledTaskSetInformer over the typed fake.
func newTestPooledTaskSetInformer(wfc *WorkflowController) cache.SharedIndexInformer {
	informer := cache.NewSharedIndexInformer(
		metadataTaskSetListWatch(wfc.wfclientset, wfc.poolSelector()),
		&metav1.PartialObjectMetadata{},
		0,
		pooledTaskSetIndexers(),
	)
	wfc.configurePooledTaskSetInformer(informer)
	return informer
}

// newInformerTestController is a controller with just what the pooled
// informers use, and a queue that adds rate-limited keys at once.
func newInformerTestController(instanceID string) *WorkflowController {
	return &WorkflowController{
		Config:  config.Config{InstanceID: instanceID},
		wfQueue: workqueue.NewTypedRateLimitingQueue[string](&fixedItemIntervalRateLimiter{}),
	}
}

func runInformer(t *testing.T, informer cache.SharedIndexInformer) {
	t.Helper()
	go informer.Run(t.Context().Done())
	require.True(t, cache.WaitForCacheSync(t.Context().Done(), informer.HasSynced))
}

func waitForWatch(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("informer never started watching")
	}
}

func cacheKeys(informer cache.SharedIndexInformer) []string {
	return informer.GetStore().ListKeys()
}

// queued drains the queue, waiting for at least one key.
func queued(t *testing.T, q workqueue.TypedRateLimitingInterface[string]) []string {
	t.Helper()
	require.Eventually(t, func() bool { return q.Len() > 0 }, 10*time.Second, 5*time.Millisecond)
	var keys []string
	for q.Len() > 0 {
		key, _ := q.Get()
		q.Done(key)
		keys = append(keys, key)
	}
	return keys
}

func assertNothingQueued(t *testing.T, q workqueue.TypedRateLimitingInterface[string]) {
	t.Helper()
	assert.Never(t, func() bool { return q.Len() > 0 }, 100*time.Millisecond, 5*time.Millisecond)
}

func pooledTaskSetMeta(nodeID, workflowName string, extraLabels map[string]string) *metav1.PartialObjectMetadata {
	l := map[string]string{
		common.LabelKeyWorkflow: workflowName,
		common.LabelKeyPoolName: "my-pool",
		common.LabelKeyPoolHash: "hash-" + nodeID,
	}
	maps.Copy(l, extraLabels)
	return &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskSetKind},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      nodeID + "-task",
			UID:       types.UID("uid-" + nodeID),
			Labels:    l,
		},
	}
}

func agentTaskSetMeta(namespace, workflowName string) *metav1.PartialObjectMetadata {
	return &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskSetKind},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      workflowName,
			Labels:    map[string]string{common.LabelKeyWorkflow: workflowName},
		},
	}
}

func newMetadataClient(t *testing.T, objects ...runtime.Object) *metadatafake.FakeMetadataClient {
	t.Helper()
	scheme := metadatafake.NewTestScheme()
	require.NoError(t, metav1.AddMetaToScheme(scheme))
	return metadatafake.NewSimpleMetadataClient(scheme, objects...)
}

var taskSetGVR = wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTaskSetPlural)

func TestPooledTaskSetInformer(t *testing.T) {
	other := map[string]string{common.LabelKeyControllerInstanceID: "other"}
	mine := map[string]string{common.LabelKeyControllerInstanceID: "mine"}
	for _, tt := range []struct {
		name       string
		instanceID string
		want       []string
		poolHash   string
	}{
		{name: "no instance ID", instanceID: "", want: []string{"ns/plain-1-task", "ns/plain-2-task"}, poolHash: "hash-plain-1"},
		{name: "instance ID", instanceID: "mine", want: []string{"ns/mine-1-task", "ns/mine-2-task"}, poolHash: "hash-mine-1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newMetadataClient(t,
				pooledTaskSetMeta("plain-1", "wf", nil),
				pooledTaskSetMeta("mine-1", "wf", mine),
				pooledTaskSetMeta("other-1", "wf", other),
				agentTaskSetMeta("ns", "wf"),
			)
			started := filterWatches(&client.Fake, client.Tracker())
			wfc := newInformerTestController(tt.instanceID)
			wfc.metadataInterface = client
			informer := wfc.newPooledTaskSetInformer(logging.TestContext(t.Context()))
			runInformer(t, informer)
			waitForWatch(t, started)

			// Created after the initial list, the matching ones last, so
			// once they are cached the others would have been too.
			resource := client.Resource(taskSetGVR).Namespace("ns").(metadatafake.MetadataClient)
			for _, obj := range []*metav1.PartialObjectMetadata{
				pooledTaskSetMeta("other-2", "wf", other),
				agentTaskSetMeta("ns", "wf-2"),
				pooledTaskSetMeta("plain-2", "wf", nil),
				pooledTaskSetMeta("mine-2", "wf", mine),
			} {
				_, err := resource.CreateFake(obj, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			require.Eventually(t, func() bool { return len(cacheKeys(informer)) == len(tt.want) }, 10*time.Second, 5*time.Millisecond)
			assert.ElementsMatch(t, tt.want, cacheKeys(informer))

			byWorkflow, err := informer.GetIndexer().ByIndex(indexes.WorkflowIndex, indexes.WorkflowIndexValue("ns", "wf"))
			require.NoError(t, err)
			assert.Len(t, byWorkflow, 2)
			byPool, err := informer.GetIndexer().ByIndex(indexes.PoolIndex, indexes.MetaNamespaceLabelIndex("ns", tt.poolHash))
			require.NoError(t, err)
			assert.Len(t, byPool, 1)
			assert.Nil(t, byPool[0].(*metav1.PartialObjectMetadata).ManagedFields)
		})
	}
}

func TestPooledTaskSetDeleteRequeuesWorkflow(t *testing.T) {
	client := newMetadataClient(t, pooledTaskSetMeta("node-1", "my-wf", nil))
	started := filterWatches(&client.Fake, client.Tracker())
	wfc := newInformerTestController("")
	wfc.metadataInterface = client
	informer := wfc.newPooledTaskSetInformer(logging.TestContext(t.Context()))
	runInformer(t, informer)
	waitForWatch(t, started)
	assertNothingQueued(t, wfc.wfQueue)

	require.NoError(t, client.Resource(taskSetGVR).Namespace("ns").Delete(t.Context(), "node-1-task", metav1.DeleteOptions{}))
	assert.Equal(t, []string{"ns/my-wf"}, queued(t, wfc.wfQueue))

	t.Run("tombstone", func(t *testing.T) {
		wfc.enqueueWorkflowFromLabel(cache.DeletedFinalStateUnknown{Key: "ns/node-2-task", Obj: pooledTaskSetMeta("node-2", "other-wf", nil)})
		assert.Equal(t, []string{"ns/other-wf"}, queued(t, wfc.wfQueue))
	})
}

func TestWorkflowTaskSetInformerExcludesPooledTaskSets(t *testing.T) {
	agent := func(name string) *wfv1.WorkflowTaskSet {
		return &wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name}}
	}
	pooled := func(nodeID string) *wfv1.WorkflowTaskSet {
		return &wfv1.WorkflowTaskSet{ObjectMeta: pooledTaskSetMeta(nodeID, "wf", nil).ObjectMeta}
	}
	client := fakewfclientset.NewClientset(agent("wf-1"), pooled("node-1"))
	started := filterWatches(&client.Fake, client.Tracker())
	wfc := newInformerTestController("")
	wfc.wfclientset = client
	informer := wfc.newWorkflowTaskSetInformer().Informer()
	runInformer(t, informer)
	waitForWatch(t, started)

	taskSets := client.ArgoprojV1alpha1().WorkflowTaskSets("ns")
	_, err := taskSets.Create(t.Context(), pooled("node-2"), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = taskSets.Create(t.Context(), agent("wf-2"), metav1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(cacheKeys(informer)) == 2 }, 10*time.Second, 5*time.Millisecond)
	assert.ElementsMatch(t, []string{"ns/wf-1", "ns/wf-2"}, cacheKeys(informer))
}

func poolPod(name, uid string, extraLabels map[string]string) *apiv1.Pod {
	l := map[string]string{
		common.LabelKeyPoolName: "my-pool",
		common.LabelKeyPoolHash: "my-hash",
	}
	maps.Copy(l, extraLabels)
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            name,
			UID:             types.UID(uid),
			Labels:          l,
			Annotations:     map[string]string{"big": "annotation"},
			ManagedFields:   []metav1.ManagedFieldsEntry{{Manager: "kubelet"}},
			OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "my-pool-abc"}},
		},
		Spec: apiv1.PodSpec{
			NodeName:   "node-a",
			Containers: []apiv1.Container{{Name: "main", Image: "my-image"}, {Name: "worker", Image: "argoexec"}},
			Volumes:    []apiv1.Volume{{Name: "var-run-argo"}},
		},
		Status: apiv1.PodStatus{
			Phase:      apiv1.PodRunning,
			Conditions: []apiv1.PodCondition{{Type: apiv1.PodReady, Status: apiv1.ConditionTrue}},
			PodIP:      "10.0.0.1",
			PodIPs:     []apiv1.PodIP{{IP: "10.0.0.1"}},
			HostIP:     "192.168.0.1",
			ContainerStatuses: []apiv1.ContainerStatus{
				{Name: "main", Ready: true},
				{Name: "worker", Ready: true, RestartCount: 2},
			},
		},
	}
}

func TestTransformPoolPod(t *testing.T) {
	pod := poolPod("pool-pod", "pod-uid", nil)
	pod.ResourceVersion = "7"
	now := metav1.Now()
	pod.DeletionTimestamp = &now
	want := pod.DeepCopy()

	out, err := transformPoolPod(pod)
	require.NoError(t, err)
	cached := out.(*apiv1.Pod)
	assert.Equal(t, want.Status, cached.Status)
	assert.Equal(t, "node-a", cached.Spec.NodeName)
	assert.Equal(t, "10.0.0.1", cached.Status.PodIP)
	assert.Equal(t, int32(2), cached.Status.ContainerStatuses[1].RestartCount)
	assert.Equal(t, want.Name, cached.Name)
	assert.Equal(t, want.Namespace, cached.Namespace)
	assert.Equal(t, want.UID, cached.UID)
	assert.Equal(t, want.ResourceVersion, cached.ResourceVersion)
	assert.Equal(t, want.Labels, cached.Labels)
	assert.Equal(t, want.OwnerReferences, cached.OwnerReferences)
	assert.Equal(t, want.DeletionTimestamp, cached.DeletionTimestamp)
	assert.Empty(t, cached.Spec.Containers)
	assert.Empty(t, cached.Spec.Volumes)
	assert.Empty(t, cached.Annotations)
	assert.Empty(t, cached.ManagedFields)

	tombstone := cache.DeletedFinalStateUnknown{Key: "ns/pool-pod", Obj: pod}
	out, err = transformPoolPod(tombstone)
	require.NoError(t, err)
	assert.Equal(t, tombstone, out)
}

func TestPoolPodInformer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, tt := range []struct {
		name       string
		instanceID string
		want       []string
	}{
		{name: "no instance ID", instanceID: "", want: []string{"ns/plain-1", "ns/plain-2"}},
		{name: "instance ID", instanceID: "mine", want: []string{"ns/mine-1", "ns/mine-2"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mine := map[string]string{common.LabelKeyControllerInstanceID: "mine"}
			other := map[string]string{common.LabelKeyControllerInstanceID: "other"}
			// A workflow pod a user labelled with a pool hash.
			workflowPod := map[string]string{common.LabelKeyWorkflow: "wf"}
			kube := fake.NewClientset(
				poolPod("plain-1", "uid-plain-1", nil),
				poolPod("mine-1", "uid-mine-1", mine),
				poolPod("other-1", "uid-other-1", other),
				poolPod("wf-pod-1", "uid-wf-pod-1", workflowPod),
			)
			started := filterWatches(&kube.Fake, kube.Tracker())
			wfc := newInformerTestController(tt.instanceID)
			wfc.kubeclientset = kube
			wfc.taskResultInformer = wfc.newWorkflowTaskResultInformer(ctx)
			informer := wfc.newPoolPodInformer(ctx)
			runInformer(t, informer)
			waitForWatch(t, started)

			for _, pod := range []*apiv1.Pod{
				poolPod("other-2", "uid-other-2", other),
				poolPod("wf-pod-2", "uid-wf-pod-2", workflowPod),
				poolPod("plain-2", "uid-plain-2", nil),
				poolPod("mine-2", "uid-mine-2", mine),
			} {
				_, err := kube.CoreV1().Pods("ns").Create(ctx, pod, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			require.Eventually(t, func() bool { return len(cacheKeys(informer)) == len(tt.want) }, 10*time.Second, 5*time.Millisecond)
			assert.ElementsMatch(t, tt.want, cacheKeys(informer))

			obj, exists, err := informer.GetStore().GetByKey(tt.want[0])
			require.NoError(t, err)
			require.True(t, exists)
			cached := obj.(*apiv1.Pod)
			assert.Equal(t, apiv1.PodRunning, cached.Status.Phase)
			assert.Equal(t, "node-a", cached.Spec.NodeName)
			assert.Equal(t, "10.0.0.1", cached.Status.PodIP)
			assert.Len(t, cached.Status.ContainerStatuses, 2)
			assert.Empty(t, cached.Spec.Containers)

			byPool, err := informer.GetIndexer().ByIndex(indexes.PoolIndex, indexes.MetaNamespaceLabelIndex("ns", "my-hash"))
			require.NoError(t, err)
			assert.Len(t, byPool, 2)
		})
	}
}

func claimResult(namespace, nodeID, workflowName, workerPodUID string, completed bool) *wfv1.WorkflowTaskResult {
	return &wfv1.WorkflowTaskResult{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      nodeID,
			Labels: map[string]string{
				common.LabelKeyWorkflow:               workflowName,
				common.LabelKeyTaskSetUID:             "ts-" + nodeID,
				common.LabelKeyWorkerPod:              "pool-pod",
				common.LabelKeyWorkerPodUID:           workerPodUID,
				common.LabelKeyReportOutputsCompleted: map[bool]string{true: "true", false: "false"}[completed],
			},
		},
	}
}

func TestPoolPodChangeRequeuesClaimingWorkflows(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewClientset(poolPod("pool-pod", "pod-uid", nil))
	started := filterWatches(&kube.Fake, kube.Tracker())
	wfc := newInformerTestController("")
	wfc.kubeclientset = kube
	// Filled directly, so its own handlers don't requeue anything.
	wfc.taskResultInformer = wfc.newWorkflowTaskResultInformer(ctx)
	for _, result := range []*wfv1.WorkflowTaskResult{
		claimResult("ns", "node-a", "claiming-wf", "pod-uid", false),
		claimResult("ns", "node-b", "completed-wf", "pod-uid", true),
		claimResult("ns", "node-c", "other-pod-wf", "other-pod-uid", false),
		claimResult("other-ns", "node-d", "other-ns-wf", "pod-uid", false),
	} {
		require.NoError(t, wfc.taskResultInformer.GetIndexer().Add(result))
	}
	informer := wfc.newPoolPodInformer(ctx)
	runInformer(t, informer)
	waitForWatch(t, started)
	assert.Equal(t, []string{"ns/claiming-wf"}, queued(t, wfc.wfQueue), "initial add")

	pods := kube.CoreV1().Pods("ns")
	pod, err := pods.Get(t.Context(), "pool-pod", metav1.GetOptions{})
	require.NoError(t, err)
	pod.Status.ContainerStatuses[1].RestartCount++
	pod.ResourceVersion = "2"
	_, err = pods.UpdateStatus(t.Context(), pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"ns/claiming-wf"}, queued(t, wfc.wfQueue), "status change")

	require.NoError(t, pods.Delete(t.Context(), "pool-pod", metav1.DeleteOptions{}))
	assert.Equal(t, []string{"ns/claiming-wf"}, queued(t, wfc.wfQueue), "delete")

	t.Run("resync", func(t *testing.T) {
		cached := transformPoolPodOrFail(t, pod)
		wfc.poolPodHandlers().OnUpdate(cached, cached)
		assertNothingQueued(t, wfc.wfQueue)
	})
	t.Run("tombstone", func(t *testing.T) {
		wfc.enqueueWorkflowsClaimedBy(cache.DeletedFinalStateUnknown{Key: "ns/pool-pod", Obj: transformPoolPodOrFail(t, pod)})
		assert.Equal(t, []string{"ns/claiming-wf"}, queued(t, wfc.wfQueue))
	})
}

func transformPoolPodOrFail(t *testing.T, pod *apiv1.Pod) *apiv1.Pod {
	t.Helper()
	out, err := transformPoolPod(pod.DeepCopy())
	require.NoError(t, err)
	return out.(*apiv1.Pod)
}

func TestCachesToSyncWaitsForPooledInformers(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx)
	defer cancel()
	allSynced := func() bool {
		for _, synced := range wfc.cachesToSync() {
			if !synced() {
				return false
			}
		}
		return true
	}
	require.True(t, allSynced())

	wfc.pooledTaskSetInformer = newTestPooledTaskSetInformer(wfc)
	wfc.poolPodInformer = wfc.newPoolPodInformer(ctx)
	assert.False(t, allSynced(), "neither started")
	runInformer(t, wfc.pooledTaskSetInformer)
	assert.False(t, allSynced(), "pool pods not synced")
	runInformer(t, wfc.poolPodInformer)
	assert.True(t, allSynced())
}

func TestControllerHarnessCachesPooledTaskSets(t *testing.T) {
	pooled := &wfv1.WorkflowTaskSet{ObjectMeta: pooledTaskSetMeta("node-1", "wf", nil).ObjectMeta}
	agent := &wfv1.WorkflowTaskSet{ObjectMeta: agentTaskSetMeta("ns", "wf").ObjectMeta}
	cancel, wfc := newController(logging.TestContext(t.Context()), pooled, agent)
	defer cancel()
	assert.Equal(t, []string{"ns/node-1-task"}, cacheKeys(wfc.pooledTaskSetInformer))
	obj, _, err := wfc.pooledTaskSetInformer.GetStore().GetByKey("ns/node-1-task")
	require.NoError(t, err)
	assert.IsType(t, &metav1.PartialObjectMetadata{}, obj)
}
