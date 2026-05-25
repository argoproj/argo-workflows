package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func newUnstructuredWorkflow(name string, labels map[string]any) *unstructured.Unstructured {
	meta := map[string]any{"name": name, "namespace": "ns1", "resourceVersion": "1"}
	if labels != nil {
		meta["labels"] = labels
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": wfv1.SchemeGroupVersion.String(),
		"kind":       workflow.WorkflowKind,
		"metadata":   meta,
	}}
}

// runProxy invokes the internal proxy directly so we control the upstream watch
// channel from the test without spinning up a fake dynamic client.
func runProxy(ctx context.Context, upstream watch.Interface) watch.Interface {
	return newTolerantWatchProxy[wfv1.Workflow, *wfv1.Workflow](
		ctx,
		upstream,
		wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowPlural),
	)
}

func TestTolerantWatch_DropsMalformedEvent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)
	defer proxy.Stop()

	go func() {
		upstream.Add(newUnstructuredWorkflow("good-1", nil))
		// Malformed: labels.foo is a number, but Labels is map[string]string.
		upstream.Modify(newUnstructuredWorkflow("broken", map[string]any{"foo": int64(1)}))
		upstream.Add(newUnstructuredWorkflow("good-2", nil))
		upstream.Stop()
	}()

	got := drain(t, proxy, 2)
	names := []string{got[0].Object.(*wfv1.Workflow).Name, got[1].Object.(*wfv1.Workflow).Name}
	assert.ElementsMatch(t, []string{"good-1", "good-2"}, names)
}

func TestTolerantWatch_ForwardsErrorEvent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)
	defer proxy.Stop()

	go func() {
		upstream.Error(&metav1.Status{Reason: metav1.StatusReasonInternalError, Message: "boom"})
		upstream.Stop()
	}()

	got := drain(t, proxy, 1)
	assert.Equal(t, watch.Error, got[0].Type)
}

// TestTolerantWatch_ConvertsBookmarkEvent guards against a regression where
// Bookmark events were forwarded as *unstructured.Unstructured. The typed
// reflector rejects mismatched object types and stalls watch-list initial sync
// until the bookmark is delivered with the expected typed payload.
func TestTolerantWatch_ConvertsBookmarkEvent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)
	defer proxy.Stop()

	go func() {
		bookmark := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": wfv1.SchemeGroupVersion.String(),
			"kind":       workflow.WorkflowKind,
			"metadata":   map[string]any{"resourceVersion": "42"},
		}}
		upstream.Action(watch.Bookmark, bookmark)
		upstream.Stop()
	}()

	got := drain(t, proxy, 1)
	require.Equal(t, watch.Bookmark, got[0].Type)
	wf, ok := got[0].Object.(*wfv1.Workflow)
	require.True(t, ok, "bookmark event must carry typed *wfv1.Workflow, got %T", got[0].Object)
	assert.Equal(t, "42", wf.ResourceVersion)
}

func TestTolerantWatch_PropagatesStop(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)

	proxy.Stop()

	// Downstream channel must close once we stop.
	select {
	case _, open := <-proxy.ResultChan():
		require.False(t, open, "downstream channel should be closed after Stop")
	case <-time.After(time.Second):
		t.Fatal("Stop did not close downstream channel within 1s")
	}
}

func drain(t *testing.T, w watch.Interface, n int) []watch.Event {
	t.Helper()
	out := make([]watch.Event, 0, n)
	deadline := time.After(2 * time.Second)
	for len(out) < n {
		select {
		case evt, ok := <-w.ResultChan():
			if !ok {
				t.Fatalf("downstream closed after %d events, expected %d", len(out), n)
			}
			out = append(out, evt)
		case <-deadline:
			t.Fatalf("timed out after %d events, expected %d", len(out), n)
		}
	}
	return out
}
