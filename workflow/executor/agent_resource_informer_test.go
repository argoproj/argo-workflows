package executor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

const (
	testInformerWorkflow  = "wf-test"
	testInformerNamespace = "ns-test"
)

var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

func newInformerScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	return s
}

func newMonitoredPod(labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
	u.SetNamespace(testInformerNamespace)
	u.SetName("p1")
	u.SetLabels(labels)
	return u
}

type informerEvent struct {
	obj     *unstructured.Unstructured
	deleted bool
}

// firstEvent reads one event from received with a deadline, failing the test
// on timeout.
func firstEvent(t *testing.T, received <-chan informerEvent) informerEvent {
	t.Helper()
	select {
	case evt := <-received:
		return evt
	case <-time.After(2 * time.Second):
		t.Fatalf("expected informer event within 2s, got none")
	}
	return informerEvent{}
}

func TestMonitoredResourceInformer_FiresOnAdd(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))
	received := make(chan informerEvent, 4)

	inf := NewMonitoredResourceInformer(client, testInformerNamespace, testInformerWorkflow, 0,
		func(_ context.Context, obj *unstructured.Unstructured, deleted bool) {
			received <- informerEvent{obj: obj, deleted: deleted}
		})
	defer inf.Stop()

	require.NoError(t, inf.Watch(ctx, podGVR))

	pod := newMonitoredPod(map[string]string{
		common.LabelKeyMonitoredResource: testInformerWorkflow,
	})
	_, err := client.Resource(podGVR).Namespace(testInformerNamespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)

	evt := firstEvent(t, received)
	assert.False(t, evt.deleted, "Add event should not be marked deleted")
	assert.Equal(t, "p1", evt.obj.GetName())
}

// Label-selector filtering is enforced by the real Kubernetes API server's
// list/watch endpoints. client-go's fake tracker honors LabelSelector on
// List (so initial cache sync filters correctly) but not on Watch streams,
// so a unit test against the fake cannot meaningfully exercise it. The
// selector wiring is verified by the production code path and the
// end-to-end integration test against a real cluster.

func TestMonitoredResourceInformer_FiresOnUpdate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))
	received := make(chan informerEvent, 8)

	inf := NewMonitoredResourceInformer(client, testInformerNamespace, testInformerWorkflow, 0,
		func(_ context.Context, obj *unstructured.Unstructured, deleted bool) {
			received <- informerEvent{obj: obj, deleted: deleted}
		})
	defer inf.Stop()

	require.NoError(t, inf.Watch(ctx, podGVR))

	pod := newMonitoredPod(map[string]string{
		common.LabelKeyMonitoredResource: testInformerWorkflow,
	})
	created, err := client.Resource(podGVR).Namespace(testInformerNamespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	_ = firstEvent(t, received) // Add

	require.NoError(t, unstructured.SetNestedField(created.Object, "Running", "status", "phase"))
	_, err = client.Resource(podGVR).Namespace(testInformerNamespace).Update(ctx, created, metav1.UpdateOptions{})
	require.NoError(t, err)

	evt := firstEvent(t, received)
	assert.False(t, evt.deleted)
	phase, _, _ := unstructured.NestedString(evt.obj.Object, "status", "phase")
	assert.Equal(t, "Running", phase)
}

func TestMonitoredResourceInformer_FiresOnDelete(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))
	received := make(chan informerEvent, 8)

	inf := NewMonitoredResourceInformer(client, testInformerNamespace, testInformerWorkflow, 0,
		func(_ context.Context, obj *unstructured.Unstructured, deleted bool) {
			received <- informerEvent{obj: obj, deleted: deleted}
		})
	defer inf.Stop()

	require.NoError(t, inf.Watch(ctx, podGVR))

	pod := newMonitoredPod(map[string]string{
		common.LabelKeyMonitoredResource: testInformerWorkflow,
	})
	_, err := client.Resource(podGVR).Namespace(testInformerNamespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	_ = firstEvent(t, received) // Add

	require.NoError(t, client.Resource(podGVR).Namespace(testInformerNamespace).Delete(ctx, "p1", metav1.DeleteOptions{}))

	evt := firstEvent(t, received)
	assert.True(t, evt.deleted, "delete event should be marked deleted")
	assert.Equal(t, "p1", evt.obj.GetName())
}

// Calling Watch twice for the same GVR should not register duplicate
// dispatchers — events still fire exactly once per object change.
func TestMonitoredResourceInformer_WatchIdempotentPerGVR(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))
	var count atomic.Int32

	inf := NewMonitoredResourceInformer(client, testInformerNamespace, testInformerWorkflow, 0,
		func(_ context.Context, obj *unstructured.Unstructured, deleted bool) {
			count.Add(1)
		})
	defer inf.Stop()

	require.NoError(t, inf.Watch(ctx, podGVR))
	require.NoError(t, inf.Watch(ctx, podGVR))
	require.NoError(t, inf.Watch(ctx, podGVR))

	pod := newMonitoredPod(map[string]string{
		common.LabelKeyMonitoredResource: testInformerWorkflow,
	})
	_, err := client.Resource(podGVR).Namespace(testInformerNamespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool { return count.Load() >= 1 }, time.Second, 10*time.Millisecond)
	// Give any duplicate handlers a chance to misfire.
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load(), "exactly one dispatch per Create regardless of Watch call count")
}
