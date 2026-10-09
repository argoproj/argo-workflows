package pod

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestStatusCaptureLegacyRecaptureRechecksFailuresAndOwnership(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, failure := range []string{"recapture failure", "readback failure", "replaced owner", "changed instance"} {
		t.Run(failure, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureFixture()
			node := wf.Status.Nodes["node"]
			node.CapturedPodUID = ""
			wf.Status.Nodes["node"] = node
			client := fake.NewClientset(pod)
			c := identityTestController(t, client)
			clock := captureTimingQueue(t, c)
			broken, recaptured := true, false
			recaptureCalls := 0
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
				if broken && recaptured && failure == "readback failure" {
					return nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("readback denied"))
				}
				return wf.DeepCopy(), nil
			}
			c.SetLegacyPodRecapture(func(context.Context, *apiv1.Pod) (*wfv1.Workflow, error) {
				recaptureCalls++
				if broken && failure == "recapture failure" {
					return nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("capture write denied"))
				}
				node := wf.Status.Nodes["node"]
				node.CapturedPodUID = string(pod.UID)
				wf.Status.Nodes["node"] = node
				returned := wf.DeepCopy()
				recaptured = true
				// The callback can return a valid capture while the subsequent
				// authoritative read belongs to another owner or controller.
				switch failure {
				case "replaced owner":
					wf.UID = "replacement-workflow"
				case "changed instance":
					wf.Labels[common.LabelKeyControllerInstanceID] = "other-controller"
				}
				return returned, nil
			})
			c.workqueue.Add(newPodCleanupKeyWithUID(pod.Namespace, pod.Name, removeFinalizer, string(pod.UID)))
			require.True(t, c.processNextPodCleanupItem(ctx))
			require.Equal(t, 1, recaptureCalls)
			assert.Zero(t, captureTimingVerbCounts(client)["patch"], "the callback alone must not authorize cleanup")
			assert.Zero(t, captureTimingVerbCounts(client)["delete"])
			captureTimingWaitScheduled(t, clock)
			captureTimingNotReady(t, c)

			broken = false
			clock.Step(podCleanupRetryDelay)
			require.Eventually(t, func() bool { return c.workqueue.Len() == 1 }, time.Second, time.Millisecond)
			require.True(t, c.processNextPodCleanupItem(ctx)) // no new Pod event
			current, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			if failure == "changed instance" {
				assert.Equal(t, pod.Finalizers, current.Finalizers, "another controller owns the remaining obligation")
				assert.Zero(t, captureTimingVerbCounts(client)["patch"])
			} else {
				// An owner replacement only permits orphan cleanup on a fresh
				// attempt; the old callback never authorizes that transition.
				assert.Equal(t, []string{"example.com/keep"}, current.Finalizers)
				assert.Equal(t, 1, captureTimingVerbCounts(client)["patch"])
			}
			if failure == "recapture failure" {
				assert.Equal(t, 2, recaptureCalls)
			} else {
				assert.Equal(t, 1, recaptureCalls)
			}
			assert.Zero(t, c.workqueue.Len())
		})
	}
}

func TestStatusCapturePodReadFailureRetriesWithoutEvent(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, action := range []podCleanupAction{removeFinalizer, deletePod} {
		t.Run(action, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureFixture()
			client := fake.NewClientset(pod)
			c := identityTestController(t, client)
			clock := captureTimingQueue(t, c)
			broken := true
			client.PrependReactor("get", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
				if broken {
					return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, pod.Name, fmt.Errorf("temporary read denial"))
				}
				return false, nil, nil
			})
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
				return wf.DeepCopy(), nil
			}
			if action == deletePod {
				c.DeletePod(ctx, pod.Namespace, pod.Name, string(pod.UID))
			} else {
				c.RemoveFinalizer(ctx, pod.Namespace, pod.Name, string(pod.UID))
			}
			captureTimingWaitScheduled(t, clock)
			clock.Step(time.Second)
			require.Eventually(t, func() bool { return c.workqueue.Len() == 1 }, time.Second, time.Millisecond)
			require.True(t, c.processNextPodCleanupItem(ctx))
			assert.Zero(t, captureTimingVerbCounts(client)["patch"])
			assert.Zero(t, captureTimingVerbCounts(client)["delete"])
			captureTimingWaitScheduled(t, clock)
			broken = false
			clock.Step(podCleanupRetryDelay)
			require.Eventually(t, func() bool { return c.workqueue.Len() == 1 }, time.Second, time.Millisecond)
			require.True(t, c.processNextPodCleanupItem(ctx))
			current, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if action == deletePod {
				assert.True(t, apierr.IsNotFound(err))
			} else {
				require.NoError(t, err)
				assert.Equal(t, []string{"example.com/keep"}, current.Finalizers)
			}
			assert.Zero(t, c.workqueue.Len())
		})
	}
}

func TestStatusCapturePatchRequiresCompletePodIdentity(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, missing := range []string{"UID", "resourceVersion"} {
		t.Run(missing, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			_, pod := captureFixture()
			client := fake.NewClientset(pod)
			c := identityTestController(t, client)
			observed := pod.DeepCopy()
			if missing == "UID" {
				observed.UID = ""
			} else {
				observed.ResourceVersion = ""
			}
			err := c.patchPodForCleanup(ctx, client.CoreV1().Pods(pod.Namespace), observed, true)
			require.ErrorContains(t, err, "UID and resourceVersion")
			assert.Empty(t, client.Actions(), "incomplete identity must not reach the mutation API")
		})
	}
}

func TestStatusCaptureRecoveryWaitsForStoppedPodProvenance(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	wf, pod := captureFixture()
	pod.Status.Phase = apiv1.PodRunning
	node := wf.Status.Nodes["node"]
	node.CapturedPodUID = ""
	wf.Status.Nodes["node"] = node
	c := identityTestController(t, fake.NewClientset(pod))
	q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
	c.workqueue = q
	c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
		return wf.DeepCopy(), nil
	}
	c.workqueue.Add(newPodCleanupKeyWithUID(pod.Namespace, pod.Name, reconcilePodCleanup, string(pod.UID)))
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
	require.Equal(t, 1, c.workqueue.Len(), "the capture hold must retain a retry")

	node.CapturedPodUID = string(pod.UID)
	wf.Status.Nodes["node"] = node
	require.True(t, c.processNextPodCleanupItem(ctx)) // retained retry, no new Pod event
	require.Equal(t, 1, c.workqueue.Len(), "persisted capture must restore the termination obligation")
	key, quit := c.workqueue.Get()
	require.False(t, quit)
	c.workqueue.Done(key)
	assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, terminateContainers, string(pod.UID)), key)
}

func TestStatusCaptureDaemonIntentWaitsForExactPodCapture(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, capturedUID := range []string{"", "previous-pod"} {
		t.Run("captured="+capturedUID, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := workflowCleanupFixture()
			node := wf.Status.Nodes["node"]
			node.CapturedPodUID = capturedUID
			wf.Status.Nodes["node"] = node
			client := fake.NewClientset(pod)
			c := identityTestController(t, client)
			q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
			c.workqueue = q
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
				return wf.DeepCopy(), nil
			}
			c.QueueDaemonTermination(ctx, wf, node, pod.Name)
			require.True(t, c.processNextPodCleanupItem(ctx))
			assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
			require.Len(t, client.Actions(), 1)
			assert.Equal(t, "get", client.Actions()[0].GetVerb(), "a completed node alone cannot authorize termination")
			require.Equal(t, 1, c.workqueue.Len())

			node.CapturedPodUID = string(pod.UID)
			wf.Status.Nodes["node"] = node
			require.True(t, c.processNextPodCleanupItem(ctx)) // retained intent, no new Pod event
			require.Equal(t, 1, c.workqueue.Len())
			key, quit := c.workqueue.Get()
			require.False(t, quit)
			c.workqueue.Done(key)
			assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, terminateContainers, string(pod.UID)), key)
		})
	}
}
