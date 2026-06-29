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

// TestTolerantWatch_ConvertsBookmarkEvent guards the Bookmark decode-failure
// fallback: a bookmark whose payload fails typed decoding must still reach the
// reflector as a typed *wfv1.Workflow carrying the bookmark's resourceVersion AND
// the k8s.io/initial-events-end annotation, otherwise the reflector rejects the
// mismatched type, relists from "", or hangs initial watch-list sync forever.
func TestTolerantWatch_ConvertsBookmarkEvent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)
	defer proxy.Stop()

	go func() {
		// Malformed: metadata.labels.foo is a number, but ObjectMeta.Labels is
		// map[string]string, so typed decoding fails and we exercise the
		// Bookmark fallback branch rather than the success path.
		bookmark := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": wfv1.SchemeGroupVersion.String(),
			"kind":       workflow.WorkflowKind,
			"metadata": map[string]any{
				"resourceVersion": "42",
				"labels":          map[string]any{"foo": int64(1)},
				"annotations":     map[string]any{"k8s.io/initial-events-end": "true"},
			},
		}}
		upstream.Action(watch.Bookmark, bookmark)
		upstream.Stop()
	}()

	got := drain(t, proxy, 1)
	require.Equal(t, watch.Bookmark, got[0].Type)
	wf, ok := got[0].Object.(*wfv1.Workflow)
	require.True(t, ok, "bookmark event must carry typed *wfv1.Workflow, got %T", got[0].Object)
	assert.Equal(t, "42", wf.ResourceVersion)
	// The watch-list end-of-sync marker must survive the fallback, or the
	// reflector never completes initial sync.
	assert.Equal(t, "true", wf.Annotations["k8s.io/initial-events-end"])
}

// TestTolerantWatch_EvictsMalformedDelete guards the Delete decode-failure
// fallback: a well-formed object can be mutated into a type-incompatible shape
// and then deleted, so its Delete event no longer decodes. Dropping it would
// leave a phantom cache entry until the next relist, so the event must still
// reach the store carrying the UID it deletes by.
func TestTolerantWatch_EvictsMalformedDelete(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)
	defer proxy.Stop()

	go func() {
		// Malformed: labels.foo is a number, but Labels is map[string]string, so
		// typed decoding fails and we exercise the Delete fallback branch.
		broken := newUnstructuredWorkflow("broken", map[string]any{"foo": int64(1)})
		broken.SetUID("uid-123")
		upstream.Delete(broken)
		upstream.Stop()
	}()

	got := drain(t, proxy, 1)
	require.Equal(t, watch.Deleted, got[0].Type)
	wf, ok := got[0].Object.(*wfv1.Workflow)
	require.True(t, ok, "delete event must carry typed *wfv1.Workflow, got %T", got[0].Object)
	// UID is what SQLiteStore.Delete keys on; without it the row is never removed.
	assert.Equal(t, "uid-123", string(wf.UID))
	assert.Equal(t, "broken", wf.Name)
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

// TestTolerantWatch_StopUnblocksPendingSend guards the inner select arm in run()
// (`case <-p.done: return` while blocked on `p.out <- out`): if the consuming
// reflector stops reading and then Stop()s while an event is in flight, the proxy
// goroutine must exit rather than leak. We observe the exit via the downstream
// channel closing — run() closes it only on return.
func TestTolerantWatch_StopUnblocksPendingSend(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	upstream := watch.NewFake()
	proxy := runProxy(ctx, upstream)

	// NewFake's channel is unbuffered, so Add returns only once run() has received
	// the event — run() is then headed for `p.out <- out`, which blocks because we
	// never drain proxy.ResultChan().
	upstream.Add(newUnstructuredWorkflow("good-1", nil))

	// Stop with that send pending. The inner `case <-p.done` arm must fire so run()
	// exits instead of leaking.
	proxy.Stop()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, open := <-proxy.ResultChan():
			if !open {
				return // run() exited and closed the channel — no leak.
			}
			// Won the select race and received the in-flight event; loop to await close.
		case <-deadline:
			t.Fatal("run() did not exit after Stop with a pending send (goroutine leak)")
		}
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
