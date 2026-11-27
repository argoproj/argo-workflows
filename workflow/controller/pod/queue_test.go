package pod

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestPodCleanupQueueNormalEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		// A fresh event for each pod would fit in the burst. Repeated events for
		// the first pod must leave the other pods' cleanup immediately runnable.
		const burst = 5
		bucket := rate.NewLimiter(rate.Every(time.Minute), burst)
		queue := workqueue.NewTypedRateLimitingQueue(&workqueue.TypedBucketRateLimiter[string]{Limiter: bucket})
		defer queue.ShutDown()
		c := &Controller{workqueue: queue, log: logging.RequireLoggerFromContext(ctx)}

		for range burst + 1 {
			c.queuePodForCleanup(ctx, "ns", "completed", labelPodCompleted)
		}
		c.queuePodForCleanup(ctx, "ns", "running", terminateContainers)
		c.queuePodForCleanupAfter(ctx, "ns", "zero", deletePod, 0)
		c.queuePodForCleanupAfter(ctx, "ns", "expired", deletePod, -time.Second)
		c.queuePodForCleanupByUID(ctx, "ns", "replacement", "pod-uid")
		c.queuePodForCleanupAfter(ctx, "ns", "delayed", deletePod, time.Minute)
		synctest.Wait()
		assert.True(t, bucket.AllowN(time.Now(), burst), "normal events must leave all retry tokens available")

		ready := []string{
			"ns/completed/labelPodCompleted",
			"ns/running/terminateContainers",
			"ns/zero/deletePod",
			"ns/expired/deletePod",
			"ns/replacement/deletePodByUID/pod-uid",
		}
		require.Equal(t, len(ready), queue.Len())
		for _, want := range ready {
			key, shutdown := queue.Get()
			require.False(t, shutdown)
			_, cleanupKey, _ := strings.Cut(key, ":")
			assert.Equal(t, want, cleanupKey)
			queue.Done(key)
		}

		time.Sleep(time.Minute - time.Second)
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
		key, _ := queue.Get()
		_, cleanupKey, _ := strings.Cut(key, ":")
		assert.Equal(t, "ns/delayed/deletePod", cleanupKey)
		queue.Done(key)
	})
}

func TestPodCleanupQueueRetriesTransientError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		queue := workqueue.NewTypedRateLimitingQueue(workqueue.NewTypedItemExponentialFailureRateLimiter[string](time.Minute, time.Hour))
		defer queue.ShutDown()
		client := fake.NewClientset()
		c := &Controller{
			config:        &config.Config{},
			kubeclientset: client,
			podInformer:   newInformer(client, new(""), new("ns")),
			workqueue:     queue,
			log:           logging.RequireLoggerFromContext(ctx),
		}
		attempts := 0
		client.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			opts := action.(k8stesting.DeleteAction).GetDeleteOptions()
			require.NotNil(t, opts.Preconditions)
			require.Equal(t, new(types.UID("pod-uid")), opts.Preconditions.UID)
			attempts++
			if attempts == 1 {
				return true, nil, apierr.NewTimeoutError("try again", 1)
			}
			return false, nil, nil // the fake client returns NotFound on retry
		})

		c.queuePodForCleanupByUID(ctx, "ns", "pod", "pod-uid")
		require.True(t, c.processNextPodCleanupItem(ctx))
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(time.Minute)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
		require.True(t, c.processNextPodCleanupItem(ctx))
		time.Sleep(time.Minute)
		synctest.Wait()
		assert.Zero(t, queue.Len())
		assert.Equal(t, 2, attempts)
	})
}

func TestPodCleanupQueueCoalescesRetries(t *testing.T) {
	for _, duringRequest := range []bool{true, false} {
		name := "during backoff"
		if duringRequest {
			name = "during first attempt"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				const retryDelay = time.Second
				bucket := rate.NewLimiter(rate.Every(retryDelay), 1)
				require.True(t, bucket.Allow(), "deplete the shared retry bucket")
				queue := workqueue.NewTypedRateLimitingQueue(&workqueue.TypedBucketRateLimiter[string]{Limiter: bucket})
				defer queue.ShutDown()
				client := fake.NewClientset()
				informer := newInformer(client, new(""), new("ns"))
				for _, name := range []string{"a", "b"} {
					pod := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name}}
					require.NoError(t, informer.GetStore().Add(pod))
				}
				c := &Controller{workqueue: queue, kubeclientset: client, podInformer: informer, log: logging.RequireLoggerFromContext(ctx)}
				queueDuplicate := func() { c.queuePodForCleanup(ctx, "ns", "a", labelPodCompleted) }
				var attempts []string
				client.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
					attempts = append(attempts, action.(k8stesting.PatchAction).GetName())
					if len(attempts) == 1 {
						if duringRequest {
							queueDuplicate()
						}
						return true, nil, apierr.NewTimeoutError("try again", 1)
					}
					return true, nil, nil
				})

				queueDuplicate()
				require.True(t, c.processNextPodCleanupItem(ctx))
				if !duringRequest {
					queueDuplicate()
				}
				c.queuePodForCleanup(ctx, "ns", "b", labelPodCompleted)
				synctest.Wait()
				require.Equal(t, 1, queue.Len())
				require.True(t, c.processNextPodCleanupItem(ctx))
				require.Equal(t, []string{"a", "b"}, attempts)

				time.Sleep(retryDelay - time.Millisecond)
				synctest.Wait()
				require.Zero(t, queue.Len())
				time.Sleep(time.Millisecond)
				synctest.Wait()
				require.Equal(t, 1, queue.Len())
				require.True(t, c.processNextPodCleanupItem(ctx))
				time.Sleep(retryDelay)
				synctest.Wait()
				require.Zero(t, queue.Len())
				require.Equal(t, []string{"a", "b", "a"}, attempts)
			})
		})
	}
}

func TestPodCleanupQueueSupersedesDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		queue := workqueue.NewTypedRateLimitingQueue(workqueue.NewTypedItemExponentialFailureRateLimiter[string](time.Minute, time.Hour))
		defer queue.ShutDown()
		client := fake.NewClientset()
		c := &Controller{
			config: &config.Config{}, kubeclientset: client, workqueue: queue,
			podInformer: newInformer(client, new(""), new("ns")), log: logging.RequireLoggerFromContext(ctx),
		}

		attempts := 0
		client.PrependReactor("delete", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
			attempts++
			return true, nil, nil
		})
		c.queuePodForCleanupAfter(ctx, "ns", "pod", deletePod, time.Second)
		synctest.Wait()
		c.queuePodForCleanup(ctx, "ns", "pod", deletePod)
		require.True(t, c.processNextPodCleanupItem(ctx))
		require.Equal(t, 1, attempts)

		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
		require.True(t, c.processNextPodCleanupItem(ctx))
		require.Equal(t, 1, attempts)
	})
}

func TestParsePodCleanupKey(t *testing.T) {
	tests := []struct {
		name          string
		key           podCleanupKey
		wantNamespace string
		wantPodName   string
		wantAction    podCleanupAction
		wantUID       string
	}{
		{
			name:          "standard key without UID",
			key:           "default/my-pod/deletePod",
			wantNamespace: "default",
			wantPodName:   "my-pod",
			wantAction:    deletePod,
			wantUID:       "",
		},
		{
			name:          "key with UID",
			key:           "default/my-pod/deletePodByUID/abc-123-def",
			wantNamespace: "default",
			wantPodName:   "my-pod",
			wantAction:    deletePodByUID,
			wantUID:       "abc-123-def",
		},
		{
			name:          "invalid key - too few parts",
			key:           "default/my-pod",
			wantNamespace: "",
			wantPodName:   "",
			wantAction:    "",
			wantUID:       "",
		},
		{
			name:          "invalid key - too many parts",
			key:           "default/my-pod/deletePod/uid/extra",
			wantNamespace: "",
			wantPodName:   "",
			wantAction:    "",
			wantUID:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, podName, action, uid := parsePodCleanupKey(tt.key)
			assert.Equal(t, tt.wantNamespace, namespace)
			assert.Equal(t, tt.wantPodName, podName)
			assert.Equal(t, tt.wantAction, action)
			assert.Equal(t, tt.wantUID, uid)
		})
	}
}

func TestNewPodCleanupKeyWithUID(t *testing.T) {
	key := newPodCleanupKeyWithUID("my-namespace", "my-pod", deletePodByUID, "pod-uid-123")
	assert.Equal(t, "my-namespace/my-pod/deletePodByUID/pod-uid-123", key)

	// Verify round-trip
	namespace, podName, action, uid := parsePodCleanupKey(key)
	assert.Equal(t, "my-namespace", namespace)
	assert.Equal(t, "my-pod", podName)
	assert.Equal(t, deletePodByUID, action)
	assert.Equal(t, "pod-uid-123", uid)
}

func TestPodCleanupPatch(t *testing.T) {
	c := &Controller{}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          map[string]string{common.LabelKeyCompleted: "false"},
			Finalizers:      []string{common.FinalizerPodStatus},
			ResourceVersion: "123456",
		},
	}

	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")

	// pod finalizer enabled, patch label
	patch, err := c.getPodCleanupPatch(pod, true)
	require.NoError(t, err)
	expected := `{"metadata":{"resourceVersion":"123456","finalizers":[],"labels":{"workflows.argoproj.io/completed":"true"}}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer enabled, do not patch label
	patch, err = c.getPodCleanupPatch(pod, false)
	require.NoError(t, err)
	expected = `{"metadata":{"resourceVersion":"123456","finalizers":[]}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer enabled, do not patch label, nil/empty finalizers
	podWithNilFinalizers := &apiv1.Pod{}
	patch, err = c.getPodCleanupPatch(podWithNilFinalizers, false)
	require.NoError(t, err)
	assert.Nil(t, patch)

	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")

	// pod finalizer disabled, patch both
	patch, err = c.getPodCleanupPatch(pod, true)
	require.NoError(t, err)
	expected = `{"metadata":{"labels":{"workflows.argoproj.io/completed":"true"}}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer disabled, do not patch label
	patch, err = c.getPodCleanupPatch(pod, false)
	require.NoError(t, err)
	assert.Nil(t, patch)
}
