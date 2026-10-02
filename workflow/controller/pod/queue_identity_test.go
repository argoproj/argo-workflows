package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"

	argoConfig "github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func identityTestPod(uid string) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "default",
			Name:            "same-name",
			UID:             types.UID(uid),
			ResourceVersion: "10",
			Labels:          map[string]string{common.LabelKeyCompleted: "false", common.LabelKeyWorkflow: "workflow"},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: wfv1.SchemeGroupVersion.String(), Kind: "Workflow", Name: "workflow", UID: "workflow-uid", Controller: new(true)}},
			Finalizers:      []string{"example.com/keep", common.FinalizerPodStatus},
		},
	}
}

func identityTestController(t *testing.T, client *fake.Clientset) *Controller {
	t.Helper()
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.NewTypedItemExponentialFailureRateLimiter[string](0, 0))
	t.Cleanup(queue.ShutDown)
	return &Controller{
		config:        &argoConfig.Config{},
		kubeclientset: client,
		workqueue:     queue,
		log:           logging.RequireLoggerFromContext(logging.TestContext(t.Context())),
		lookupWorkflow: func(_ context.Context, namespace, name string, _ bool) (*wfv1.Workflow, error) {
			return &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, UID: "workflow-uid"}}, nil
		},
	}
}

func TestCleanupIgnoresReplacementUID(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, action := range []podCleanupAction{removeFinalizer, labelPodCompleted, deletePod, deletePodByUID, terminateContainers, killContainers} {
		t.Run(action, func(t *testing.T) {
			replacement := identityTestPod("replacement")
			client := fake.NewSimpleClientset(replacement)
			c := identityTestController(t, client)
			c.workqueue.Add(newPodCleanupKeyWithUID(replacement.Namespace, replacement.Name, action, "old-pod"))

			require.True(t, c.processNextPodCleanupItem(logging.TestContext(t.Context())))

			// The authoritative GET is allowed, but the old decision must not even
			// attempt PATCH, DELETE, or exec against the replacement.
			require.Len(t, client.Actions(), 1)
			assert.Equal(t, "get", client.Actions()[0].GetVerb())
			current, err := client.CoreV1().Pods(replacement.Namespace).Get(t.Context(), replacement.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, replacement.Finalizers, current.Finalizers)
			assert.Equal(t, "false", current.Labels[common.LabelKeyCompleted])
		})
	}
}

func TestCleanupPatchConflictPreservesConcurrentFinalizer(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	pod := identityTestPod("original")
	client := fake.NewSimpleClientset(pod)
	c := identityTestController(t, client)
	key := newPodCleanupKeyWithUID(pod.Namespace, pod.Name, removeFinalizer, string(pod.UID))
	patches := 0
	client.PrependReactor("patch", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		patches++
		patch := action.(clienttesting.PatchAction)
		assert.Equal(t, types.MergePatchType, patch.GetPatchType())
		var mutation apiv1.Pod
		require.NoError(t, json.Unmarshal(patch.GetPatch(), &mutation))
		assert.Equal(t, pod.UID, mutation.UID)
		if patches == 1 {
			assert.Equal(t, "10", mutation.ResourceVersion)
			// Model a concurrent writer after our read. The fake API does not
			// implement resourceVersion conflicts, so inject its expected error.
			concurrent := pod.DeepCopy()
			concurrent.ResourceVersion = "11"
			concurrent.Finalizers = append(concurrent.Finalizers, "example.com/concurrent")
			require.NoError(t, client.Tracker().Update(apiv1.SchemeGroupVersion.WithResource("pods"), concurrent, pod.Namespace))
			return true, nil, apierr.NewConflict(schema.GroupResource{Resource: "pods"}, pod.Name, fmt.Errorf("resourceVersion changed"))
		}
		assert.Equal(t, "11", mutation.ResourceVersion)
		assert.Equal(t, []string{"example.com/keep", "example.com/concurrent"}, mutation.Finalizers)
		return false, nil, nil
	})
	c.workqueue.Add(key)
	require.True(t, c.processNextPodCleanupItem(logging.TestContext(t.Context())))
	assert.Equal(t, 1, c.workqueue.NumRequeues(key))
	current, err := client.CoreV1().Pods(pod.Namespace).Get(t.Context(), pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, current.Finalizers, common.FinalizerPodStatus)
	assert.Contains(t, current.Finalizers, "example.com/concurrent")

	require.True(t, c.processNextPodCleanupItem(logging.TestContext(t.Context())))
	assert.Equal(t, 0, c.workqueue.NumRequeues(key))
	current, err = client.CoreV1().Pods(pod.Namespace).Get(t.Context(), pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/keep", "example.com/concurrent"}, current.Finalizers)
}

func TestCleanupDeletionKeepsUIDPrecondition(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, action := range []podCleanupAction{deletePod, deletePodByUID} {
		t.Run(action, func(t *testing.T) {
			pod := identityTestPod("original")
			client := fake.NewSimpleClientset(pod)
			c := identityTestController(t, client)
			client.PrependReactor("delete", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
				options := action.(clienttesting.DeleteAction).GetDeleteOptions()
				require.NotNil(t, options.Preconditions)
				require.NotNil(t, options.Preconditions.UID)
				assert.Equal(t, pod.UID, *options.Preconditions.UID)
				return true, nil, nil
			})
			c.workqueue.Add(newPodCleanupKeyWithUID(pod.Namespace, pod.Name, action, string(pod.UID)))
			require.True(t, c.processNextPodCleanupItem(logging.TestContext(t.Context())))
			require.Len(t, client.Actions(), 3)
			assert.Equal(t, "get", client.Actions()[0].GetVerb())
			assert.Equal(t, "patch", client.Actions()[1].GetVerb())
			assert.Equal(t, "delete", client.Actions()[2].GetVerb())
		})
	}
}

func TestCleanupRejectsUnboundRequest(t *testing.T) {
	pod := identityTestPod("original")
	client := fake.NewSimpleClientset(pod)
	c := identityTestController(t, client)
	ctx := logging.TestContext(t.Context())
	c.queuePodForCleanup(ctx, pod.Namespace, pod.Name, removeFinalizer, "")
	assert.Zero(t, c.workqueue.Len())
	c.workqueue.Add(newPodCleanupKey(pod.Namespace, pod.Name, removeFinalizer))
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Empty(t, client.Actions())
}

func TestDeletePodByUIDKeepsRequestedUID(t *testing.T) {
	replacement := identityTestPod("replacement")
	client := fake.NewSimpleClientset(replacement)
	c := identityTestController(t, client)
	c.DeletePodByUID(logging.TestContext(t.Context()), replacement.Namespace, replacement.Name, "old-pod")
	key, quit := c.workqueue.Get()
	require.False(t, quit)
	defer c.workqueue.Done(key)
	assert.Equal(t, newPodCleanupKeyWithUID(replacement.Namespace, replacement.Name, deletePodByUID, "old-pod"), key)
	assert.Empty(t, client.Actions(), "enqueue must not rebind an old decision to a new API object")
}

func TestPendingTerminationFollowupKeepsUID(t *testing.T) {
	pod := identityTestPod("original")
	pod.Status.Phase = apiv1.PodPending
	client := fake.NewSimpleClientset(pod)
	c := identityTestController(t, client)
	c.workqueue.Add(newPodCleanupKeyWithUID(pod.Namespace, pod.Name, terminateContainers, string(pod.UID)))
	require.True(t, c.processNextPodCleanupItem(logging.TestContext(t.Context())))
	key, quit := c.workqueue.Get()
	require.False(t, quit)
	defer c.workqueue.Done(key)
	assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, deletePod, string(pod.UID)), key)
}
