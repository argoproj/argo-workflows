package pod

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func newCleanupTestController(t *testing.T) (*Controller, *apiv1.Pod, *fake.Clientset) {
	t.Helper()
	pod := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "ns", Name: "pod", UID: "original-uid", ResourceVersion: "123",
		Labels:     map[string]string{common.LabelKeyCompleted: "true"},
		Finalizers: []string{common.FinalizerPodStatus, "example.com/retain"},
	}}
	client := fake.NewClientset(pod.DeepCopy())
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
	t.Cleanup(queue.ShutDown)
	return &Controller{
		config: &config.Config{}, kubeclientset: client, workqueue: queue,
		podInformer: cache.NewSharedIndexInformer(&cache.ListWatch{}, &apiv1.Pod{}, 0, cache.Indexers{}),
		log:         logging.RequireLoggerFromContext(logging.TestContext(t.Context())),
	}, pod, client
}

func TestRemoveFinalizerByUIDSkipsReplacement(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	ctx := logging.TestContext(t.Context())
	controller, pod, client := newCleanupTestController(t)
	controller.RemoveFinalizer(ctx, pod.Namespace, pod.Name, "old-uid")
	require.True(t, controller.processNextPodCleanupItem(ctx))
	updated, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, pod.Finalizers, updated.Finalizers)
}

func TestDeletePodByUIDCleanup(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	for _, scenario := range []string{"outside informer", "replacement", "replaced before patch", "replaced before delete", "missing UID", "other instance"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			controller, target, client := newCleanupTestController(t)
			queuedUID := string(target.UID)
			if scenario == "replacement" {
				target.UID = "replacement-uid"
			}
			if scenario == "missing UID" {
				queuedUID = ""
			}
			if scenario == "other instance" {
				target.Labels = map[string]string{common.LabelKeyControllerInstanceID: "other"}
			}
			require.NoError(t, client.Tracker().Update(apiv1.SchemeGroupVersion.WithResource("pods"), target, target.Namespace))
			if scenario != "outside informer" {
				require.NoError(t, controller.podInformer.GetStore().Add(target.DeepCopy()))
			}
			patchCount, deleteCount := 0, 0
			client.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				patchCount++
				var patch apiv1.Pod
				require.NoError(t, json.Unmarshal(action.(k8stesting.PatchAction).GetPatch(), &patch))
				assert.Equal(t, types.UID(queuedUID), patch.UID)
				assert.Equal(t, "123", patch.ResourceVersion)
				if scenario == "replaced before patch" {
					replacement := target.DeepCopy()
					replacement.UID = "replacement-uid"
					require.NoError(t, client.Tracker().Update(apiv1.SchemeGroupVersion.WithResource("pods"), replacement, replacement.Namespace))
					if patch.UID != replacement.UID {
						return true, nil, apierr.NewConflict(apiv1.Resource("pods"), replacement.Name, fmt.Errorf("UID changed"))
					}
				}
				return false, nil, nil
			})
			client.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				deleteCount++
				preconditions := action.(k8stesting.DeleteAction).GetDeleteOptions().Preconditions
				require.NotNil(t, preconditions)
				require.NotNil(t, preconditions.UID)
				assert.Equal(t, types.UID(queuedUID), *preconditions.UID)
				if scenario == "replaced before delete" {
					replacement := target.DeepCopy()
					replacement.UID = "replacement-uid"
					require.NoError(t, client.Tracker().Update(apiv1.SchemeGroupVersion.WithResource("pods"), replacement, replacement.Namespace))
					return true, nil, apierr.NewConflict(apiv1.Resource("pods"), replacement.Name, fmt.Errorf("UID changed"))
				}
				return true, nil, nil
			})

			if scenario == "missing UID" {
				controller.DeletePodByUID(ctx, target.Namespace, target.Name, queuedUID)
				controller.RemoveFinalizer(ctx, target.Namespace, target.Name, queuedUID)
				require.Zero(t, controller.workqueue.Len())
				return
			}
			controller.DeletePodByUID(ctx, target.Namespace, target.Name, queuedUID)
			require.True(t, controller.processNextPodCleanupItem(ctx))
			updated, err := client.CoreV1().Pods(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
			require.NoError(t, err)
			switch scenario {
			case "outside informer":
				assert.Equal(t, 1, patchCount)
				assert.Equal(t, 1, deleteCount)
				assert.Equal(t, []string{"example.com/retain"}, updated.Finalizers)
			case "replaced before patch":
				assert.Equal(t, 1, patchCount)
				assert.Zero(t, deleteCount)
				assert.Equal(t, target.Finalizers, updated.Finalizers)
			case "replaced before delete":
				assert.Equal(t, 1, patchCount)
				assert.Equal(t, 1, deleteCount)
				assert.Equal(t, target.Finalizers, updated.Finalizers)
			default:
				assert.Zero(t, patchCount)
				assert.Zero(t, deleteCount)
				assert.Equal(t, target.Finalizers, updated.Finalizers)
			}
		})
	}
}
