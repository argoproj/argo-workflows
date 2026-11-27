package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type workflowCleanupHandlerRecorder struct {
	cache.SharedIndexInformer
	handlers []cache.ResourceEventHandler
}

func (recorder *workflowCleanupHandlerRecorder) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	recorder.handlers = append(recorder.handlers, handler)
	return nil, nil
}

// Isolate the Workflow callback so Pod informer events cannot enqueue their own cleanup.
func workflowCleanupTestConfig(instanceID string) func(*WorkflowController) {
	return func(c *WorkflowController) {
		c.Config.InstanceID = instanceID
		c.kubeclientset.(*fake.Clientset).PrependWatchReactor("pods", func(k8stesting.Action) (bool, watch.Interface, error) {
			return true, watch.NewRaceFreeFake(), nil
		})
	}
}

func cleanupWorkflow() *unstructured.Unstructured {
	wf := &unstructured.Unstructured{}
	wf.SetNamespace("ns")
	wf.SetName("workflow")
	wf.SetUID("workflow-uid")
	return wf
}

func cleanupPod(wf *unstructured.Unstructured, name string) *apiv1.Pod {
	podLabels := wf.GetLabels()
	if podLabels == nil {
		podLabels = map[string]string{}
	}
	podLabels[common.LabelKeyWorkflow] = wf.GetName()
	podLabels[common.LabelKeyCompleted] = "true"
	return &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: wf.GetNamespace(), Name: name, UID: types.UID(name + "-uid"),
		Labels: podLabels, Finalizers: []string{common.FinalizerPodStatus, "example.com/retain"},
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind("Workflow"))},
	}}
}

func TestWorkflowPodFinalizerCleanupEvents(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	for _, event := range []string{"completed", "foreground", "delete", "tombstone"} {
		t.Run(event, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, workflowCleanupTestConfig(""))
			defer cancel()
			wf := cleanupWorkflow()
			wf.SetLabels(map[string]string{common.LabelKeyCompleted: "true"})
			client := controller.kubeclientset.(*fake.Clientset)
			pod := cleanupPod(wf, "completed")
			require.NoError(t, client.Tracker().Create(apiv1.SchemeGroupVersion.WithResource("pods"), pod, pod.Namespace))
			recorder := &workflowCleanupHandlerRecorder{SharedIndexInformer: controller.wfInformer}
			controller.wfInformer = recorder
			require.NoError(t, controller.addWorkflowInformerHandlers(ctx))
			lists := 0
			client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				lists++
				selector := action.(k8stesting.ListAction).GetListRestrictions().Labels
				assert.False(t, selector.Matches(labels.Set{common.LabelKeyWorkflow: wf.GetName(), common.LabelKeyControllerInstanceID: ""}))
				return false, nil, nil
			})
			for _, handler := range recorder.handlers {
				handler.OnAdd(wf, true)
				handler.OnUpdate(wf, wf)
			}
			require.Zero(t, lists)
			deleting := wf.DeepCopy()
			now := metav1.Now()
			deleting.SetDeletionTimestamp(&now)
			deleting.SetFinalizers([]string{metav1.FinalizerDeleteDependents})
			for _, handler := range recorder.handlers {
				switch event {
				case "completed":
					active := wf.DeepCopy()
					active.SetLabels(nil)
					handler.OnUpdate(active, wf)
				case "foreground":
					handler.OnUpdate(wf, deleting)
					handler.OnUpdate(deleting, deleting)
				case "delete":
					handler.OnDelete(wf)
				case "tombstone":
					handler.OnDelete(cache.DeletedFinalStateUnknown{Key: wf.GetNamespace() + "/" + wf.GetName(), Obj: wf})
				}
			}
			require.Equal(t, 1, lists)
			require.Equal(t, 1, controller.PodController.TestingQueueLen())
			require.True(t, controller.PodController.TestingProcessNextItem(ctx))
			updated, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, []string{"example.com/retain"}, updated.Finalizers)
		})
	}
}

func TestWorkflowPodFinalizerCleanupOwnership(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, workflowCleanupTestConfig("instance-a"))
	defer cancel()
	wf := cleanupWorkflow()
	wf.SetLabels(map[string]string{common.LabelKeyControllerInstanceID: "instance-a"})
	client := controller.kubeclientset.(*fake.Clientset)
	for _, tc := range []struct {
		name   string
		change func(*apiv1.Pod)
	}{
		{"other namespace", func(p *apiv1.Pod) { p.Namespace = "other" }},
		{"other instance", func(p *apiv1.Pod) { p.Labels[common.LabelKeyControllerInstanceID] = "instance-b" }},
		{"empty instance", func(p *apiv1.Pod) { p.Labels[common.LabelKeyControllerInstanceID] = "" }},
		{"other workflow label", func(p *apiv1.Pod) { p.Labels[common.LabelKeyWorkflow] = "other" }},
		{"unowned", func(p *apiv1.Pod) { p.OwnerReferences = nil }},
		{"other workflow UID", func(p *apiv1.Pod) { p.OwnerReferences[0].UID = "other" }},
		{"other owner name", func(p *apiv1.Pod) { p.OwnerReferences[0].Name = "other" }},
		{"other owner kind", func(p *apiv1.Pod) { p.OwnerReferences[0].Kind = "Job" }},
		{"other owner API", func(p *apiv1.Pod) { p.OwnerReferences[0].APIVersion = "example.com/v1" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := cleanupPod(wf, "target")
			tc.change(pod)
			client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, &apiv1.PodList{Items: []apiv1.Pod{*pod}}, nil
			})
			require.NoError(t, controller.removeWorkflowPodFinalizers(ctx, wf))
			assert.Zero(t, controller.PodController.TestingQueueLen())
		})
	}
	client.ClearActions()
	wf.SetLabels(nil)
	require.NoError(t, controller.removeWorkflowPodFinalizers(ctx, wf))
	assert.Empty(t, client.Actions(), "a workflow from another instance must not trigger a list")
}

func TestWorkflowPodFinalizerCleanupPagination(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, workflowCleanupTestConfig("instance-a"))
	defer cancel()
	wf := cleanupWorkflow()
	wf.SetLabels(map[string]string{common.LabelKeyControllerInstanceID: "instance-a"})
	pods := make([]apiv1.Pod, 501)
	for i := range pods {
		pods[i] = *cleanupPod(wf, fmt.Sprintf("pod-%d", i))
	}
	var tokens []string
	client := controller.kubeclientset.(*fake.Clientset)
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		options := action.(k8stesting.ListActionImpl).GetListOptions()
		assert.Equal(t, wf.GetNamespace(), action.GetNamespace())
		assert.Equal(t, int64(500), options.Limit)
		assert.Equal(t, labels.Set{common.LabelKeyWorkflow: wf.GetName(), common.LabelKeyControllerInstanceID: "instance-a"}.String(), options.LabelSelector)
		tokens = append(tokens, options.Continue)
		if len(tokens) == 1 {
			return true, &apiv1.PodList{ListMeta: metav1.ListMeta{Continue: "next"}, Items: pods[:500]}, nil
		}
		return true, &apiv1.PodList{Items: pods[500:]}, nil
	})
	require.NoError(t, controller.removeWorkflowPodFinalizers(ctx, wf))
	assert.Equal(t, []string{"", "next"}, tokens)
	assert.Equal(t, len(pods), controller.PodController.TestingQueueLen())
}

func TestWorkflowPodFinalizerCleanupPaginationError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	workflow := cleanupWorkflow()
	client := controller.kubeclientset.(*fake.Clientset)
	var continueTokens []string
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		options := action.(k8stesting.ListActionImpl).GetListOptions()
		continueTokens = append(continueTokens, options.Continue)
		if options.Continue == "" {
			return true, &apiv1.PodList{ListMeta: metav1.ListMeta{Continue: "second-page"}}, nil
		}
		return true, nil, assert.AnError
	})
	require.ErrorIs(t, controller.removeWorkflowPodFinalizers(ctx, workflow), assert.AnError)
	assert.Equal(t, []string{"", "second-page"}, continueTokens)
}
