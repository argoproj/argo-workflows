package pod

import (
	"context"
	"fmt"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func signalTestPod() *apiv1.Pod {
	pod := identityTestPod("signal-pod")
	pod.Status.Phase = apiv1.PodRunning
	pod.Spec.TerminationGracePeriodSeconds = new(int64(2))
	pod.Status.ContainerStatuses = []apiv1.ContainerStatus{
		{Name: "main", State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
		{Name: "wait", State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
		{Name: "exited", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{}}},
	}
	return pod
}

func TestSignalContainersAttemptsAllAndSkipsExited(t *testing.T) {
	pod := signalTestPod()
	c := identityTestController(t, fake.NewSimpleClientset(pod))
	denied := apierr.NewForbidden(schema.GroupResource{Resource: "pods/exec"}, pod.Name, fmt.Errorf("denied"))
	missing := apierr.NewNotFound(schema.GroupResource{Resource: "pods/exec"}, pod.Name)
	blocked := true
	var calls []string
	c.signalContainer = func(_ context.Context, _ *rest.Config, actual *apiv1.Pod, name string, sig syscall.Signal) error {
		assert.Equal(t, pod.UID, actual.UID)
		assert.Equal(t, syscall.SIGTERM, sig)
		calls = append(calls, name)
		if name == "main" {
			return missing
		}
		if blocked {
			return denied
		}
		return nil
	}
	grace, err := c.signalContainers(t.Context(), pod, syscall.SIGTERM)
	require.ErrorIs(t, err, denied)
	require.ErrorIs(t, err, missing)
	assert.Equal(t, 2*time.Second, grace)
	assert.Equal(t, []string{"main", "wait"}, calls)
	// A later authoritative Pod read can observe the first container exited.
	// Its previous delivery failure must not permanently block the other one.
	pod.Status.ContainerStatuses[0].State = apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{}}
	calls = nil
	blocked = false
	_, err = c.signalContainers(t.Context(), pod, syscall.SIGTERM)
	require.NoError(t, err)
	assert.Equal(t, []string{"wait"}, calls)
}

func TestSignalCleanupRealQueueRetriesTermAndKillWithoutEvent(t *testing.T) {
	// Virtual time advances only after the delaying queue is durably blocked.
	// Counting fake-clock waiters cannot certify that a duplicate AddAfter has
	// finished rearming a timer; synctest.Wait supplies that synchronization.
	synctest.Test(t, func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		pod := signalTestPod()
		client := fake.NewSimpleClientset(pod)
		c := identityTestController(t, client)
		c.workqueue.ShutDown()
		c.workqueue = workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
		t.Cleanup(c.workqueue.ShutDown)
		denied := true
		var calls []syscall.Signal
		c.signalContainer = func(_ context.Context, _ *rest.Config, actual *apiv1.Pod, name string, sig syscall.Signal) error {
			assert.Equal(t, pod.UID, actual.UID)
			calls = append(calls, sig)
			if denied {
				if name == "main" {
					return apierr.NewNotFound(schema.GroupResource{Resource: "pods/exec"}, pod.Name)
				}
				return apierr.NewForbidden(schema.GroupResource{Resource: "pods/exec"}, pod.Name, fmt.Errorf("denied"))
			}
			return nil
		}
		termKey := newPodCleanupKeyWithUID(pod.Namespace, pod.Name, terminateContainers, string(pod.UID))
		killKey := newPodCleanupKeyWithUID(pod.Namespace, pod.Name, killContainers, string(pod.UID))
		c.workqueue.Add(termKey)
		require.True(t, c.processNextPodCleanupItem(ctx))
		assert.Equal(t, []syscall.Signal{syscall.SIGTERM, syscall.SIGTERM}, calls)
		synctest.Wait()
		time.Sleep(2*time.Second - time.Nanosecond)
		synctest.Wait()
		require.Zero(t, c.workqueue.Len())
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Equal(t, 1, c.workqueue.Len())
		require.True(t, c.processNextPodCleanupItem(ctx)) // Escalation still happens despite failed TERM.
		assert.Equal(t, []syscall.Signal{syscall.SIGTERM, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGKILL}, calls)
		synctest.Wait()
		time.Sleep(podCleanupRetryDelay - 2*time.Second - time.Nanosecond)
		synctest.Wait()
		require.Zero(t, c.workqueue.Len())
		assert.Equal(t, map[string]int{"get": 2}, captureTimingVerbCounts(client), "no polling before either deadline")
		denied = false // Restore only delivery capability: no Pod mutation, event or restart.
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Equal(t, 1, c.workqueue.Len())
		require.True(t, c.processNextPodCleanupItem(ctx)) // TERM retry at30s.
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Equal(t, 1, c.workqueue.Len())
		require.True(t, c.processNextPodCleanupItem(ctx)) // KILL retry at32s; duplicate escalation is deduplicated.
		assert.Equal(t, []syscall.Signal{syscall.SIGTERM, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGKILL}, calls)
		assert.Equal(t, map[string]int{"get": 4}, captureTimingVerbCounts(client))
		assert.Zero(t, c.workqueue.NumRequeues(termKey))
		assert.Zero(t, c.workqueue.NumRequeues(killKey))
		synctest.Wait()
		time.Sleep(time.Hour)
		synctest.Wait()
		require.Zero(t, c.workqueue.Len())
		assert.Len(t, calls, 8, "both successful UID-bound actions are forgotten")
	})
}
