package pod

import (
	"encoding/json"
	"fmt"
	"os"
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

func TestRemoveFinalizerByUIDOutsidePodInformer(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	for _, tt := range []struct {
		name           string
		podUID         string
		queuedUID      string
		wantFinalizers []string
	}{
		{
			name:      "removes finalizer from completed pod missing from informer",
			podUID:    "original-pod-uid",
			queuedUID: "original-pod-uid",
		},
		{
			name:           "preserves finalizer on a replacement pod",
			podUID:         "replacement-pod-uid",
			queuedUID:      "original-pod-uid",
			wantFinalizers: []string{common.FinalizerPodStatus},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			ctx, log := logging.RequireLoggerFromContext(ctx).WithField("component", "pod_controller").InContext(ctx)
			pod := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "review-audit",
					Name:            "historical-pod",
					UID:             types.UID(tt.podUID),
					ResourceVersion: "123",
					Labels: map[string]string{
						common.LabelKeyWorkflow:  "historical-workflow",
						common.LabelKeyCompleted: "true",
					},
					Finalizers: []string{common.FinalizerPodStatus},
				},
				Status: apiv1.PodStatus{Phase: apiv1.PodSucceeded},
			}
			client := fake.NewClientset(pod)
			informer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &apiv1.Pod{}, 0, cache.Indexers{})
			queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
			defer queue.ShutDown()
			controller := &Controller{
				config:        &config.Config{},
				kubeclientset: client,
				podInformer:   informer,
				workqueue:     queue,
				log:           log,
			}

			controller.RemoveFinalizer(ctx, pod.Namespace, pod.Name, tt.queuedUID)
			require.True(t, controller.processNextPodCleanupItem(ctx))

			updatedPod, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			if tt.wantFinalizers == nil {
				assert.Empty(t, updatedPod.Finalizers)
			} else {
				assert.Equal(t, tt.wantFinalizers, updatedPod.Finalizers)
			}
		})
	}
}

func TestDeletePodByUIDCleanup(t *testing.T) {
	for _, captureFlag := range []string{"true", "false", "unset"} {
		t.Run(captureFlag, func(t *testing.T) {
			t.Setenv(common.EnvVarPodStatusCaptureFinalizer, captureFlag)
			if captureFlag == "unset" {
				require.NoError(t, os.Unsetenv(common.EnvVarPodStatusCaptureFinalizer))
			}
			for _, scenario := range []string{"matching", "outside informer", "replacement", "replaced before patch", "replaced before delete", "missing UID", "other instance"} {
				t.Run(scenario, func(t *testing.T) {
					ctx := logging.TestContext(t.Context())
					target := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
						Namespace: "cleanup-test", Name: "target", UID: "original-uid", ResourceVersion: "123",
						Finalizers: []string{common.FinalizerPodStatus, "example.com/retain"},
					}}
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
					client := fake.NewClientset(target.DeepCopy())
					informer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &apiv1.Pod{}, 0, cache.Indexers{})
					if scenario != "outside informer" {
						require.NoError(t, informer.GetStore().Add(target.DeepCopy()))
					}
					queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
					defer queue.ShutDown()
					controller := &Controller{config: &config.Config{}, kubeclientset: client, podInformer: informer, workqueue: queue, log: logging.RequireLoggerFromContext(ctx)}
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

					controller.DeletePodByUID(ctx, target.Namespace, target.Name, queuedUID)
					require.True(t, controller.processNextPodCleanupItem(ctx))
					updated, err := client.CoreV1().Pods(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
					require.NoError(t, err)
					switch scenario {
					case "matching", "outside informer":
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
		})
	}
}
