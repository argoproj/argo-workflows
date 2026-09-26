package events

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type eventRequest struct {
	method, path string
	event        corev1.Event
	status       int
}

// eventTransport uses the real typed Kubernetes client and serializer, without
// kubeconfig, credentials, sockets or a cluster. The store implements only the
// event Create/Patch responses needed to exercise client-go's correlation path.
type eventTransport struct {
	mu             sync.Mutex
	events         map[string]corev1.Event
	requests       chan eventRequest
	missingPatch   bool
	rejectNext     bool
	transientError bool
	block          <-chan struct{}
	entered        chan struct{}
}

func (s *eventTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.block != nil {
		select {
		case s.entered <- struct{}{}:
		default:
		}
		<-s.block
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var event corev1.Event
	status := http.StatusCreated
	path := req.URL.Path
	if req.Method == http.MethodPatch {
		event = s.events[path]
		status = http.StatusOK
		if s.missingPatch {
			status = http.StatusNotFound
			s.missingPatch = false
		}
	} else if req.Method != http.MethodPost {
		return nil, fmt.Errorf("unexpected event method %s", req.Method)
	}
	if err = json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	if s.transientError {
		s.transientError = false
		s.requests <- eventRequest{req.Method, path, event, 0}
		return nil, fmt.Errorf("synthetic transport failure")
	}
	if s.rejectNext {
		status = http.StatusForbidden
		s.rejectNext = false
	}
	var response []byte
	if status >= 400 {
		reason := metav1.StatusReasonForbidden
		if status == http.StatusNotFound {
			reason = metav1.StatusReasonNotFound
		}
		response, err = json.Marshal(&metav1.Status{TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: metav1.StatusFailure, Reason: reason, Code: int32(status)})
	} else {
		event.TypeMeta = metav1.TypeMeta{Kind: "Event", APIVersion: "v1"}
		event.ResourceVersion = "synthetic-version"
		key := path
		if req.Method == http.MethodPost {
			key += "/" + event.Name
		}
		s.events[key] = event
		response, err = json.Marshal(&event)
	}
	if err != nil {
		return nil, err
	}
	s.requests <- eventRequest{req.Method, path, event, status}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(response))), Request: req}, nil
}

func newTestEventManager(t *testing.T) (*eventRecorderManager, *eventTransport) {
	t.Helper()
	transport := &eventTransport{events: make(map[string]corev1.Event), requests: make(chan eventRequest, 8192)}
	client, err := kubernetes.NewForConfigAndClient(&rest.Config{Host: "https://events.invalid", ContentConfig: rest.ContentConfig{ContentType: "application/json"}, QPS: 100000, Burst: 100000}, &http.Client{Transport: transport})
	require.NoError(t, err)
	manager := NewEventRecorderManagerWithLogger(client, logging.NewTestLogger(logging.Info, logging.Text)).(*eventRecorderManager)
	// A stable component logger is separate from the request loggers passed to Get.
	manager.logger = logr.Discard()
	t.Cleanup(func() {
		if manager.broadcaster != nil {
			manager.broadcaster.Shutdown()
		}
	})
	return manager, transport
}

func nextEventRequest(t *testing.T, transport *eventTransport) eventRequest {
	t.Helper()
	select {
	case req := <-transport.requests:
		return req
	case <-time.After(5 * time.Second):
		t.Fatal("event did not reach typed HTTP transport")
		return eventRequest{}
	}
}

func eventPod(ns, name string, uid types.UID) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, UID: uid, ResourceVersion: "object-version"}}
}

func emitEvent(recorder record.EventRecorder, method string, obj k8sruntime.Object, reason string) {
	switch method {
	case "Event":
		recorder.Event(obj, corev1.EventTypeNormal, reason, "message 7")
	case "Eventf":
		recorder.Eventf(obj, corev1.EventTypeNormal, reason, "message %d", 7)
	case "AnnotatedEventf":
		recorder.AnnotatedEventf(obj, map[string]string{"node": "node-7"}, corev1.EventTypeNormal, reason, "message %d", 7)
	}
}

func TestEventRecorderNamespaceGuards(t *testing.T) {
	for _, method := range []string{"Event", "Eventf", "AnnotatedEventf"} {
		t.Run(method, func(t *testing.T) {
			manager, transport := newTestEventManager(t)
			ctx := logging.TestContext(t.Context())
			for i, tc := range []struct{ bound, object, want string }{
				{"alpha", "alpha", "alpha"}, {"beta", "beta", "beta"}, {"", "beta", "beta"}, {"default", "", "default"}, {"", "", "default"},
			} {
				obj := eventPod(tc.object, fmt.Sprintf("pod-%d", i), types.UID(fmt.Sprintf("uid-%d", i)))
				emitEvent(manager.Get(ctx, tc.bound), method, obj, "Allowed")
				req := nextEventRequest(t, transport)
				require.Equal(t, http.MethodPost, req.method)
				assert.Equal(t, "/api/v1/namespaces/"+tc.want+"/events", req.path)
				assert.Equal(t, tc.want, req.event.Namespace)
				assert.Equal(t, tc.object, req.event.InvolvedObject.Namespace)
				assert.Equal(t, obj.UID, req.event.InvolvedObject.UID)
				assert.Equal(t, "workflow-controller", req.event.Source.Component)
				assert.Equal(t, "workflow-controller", req.event.ReportingController)
				assert.Equal(t, "message 7", req.event.Message)
				assert.Equal(t, "Allowed", req.event.Reason)
				assert.Equal(t, int32(1), req.event.Count)
				if method == "AnnotatedEventf" {
					assert.Equal(t, map[string]string{"node": "node-7"}, req.event.Annotations)
				}
			}
			// Send a barrier through the same FIFO sink after the rejected calls, so
			// absence assertions don't depend on a sleep or on fake-client validation.
			emitEvent(manager.Get(ctx, "alpha"), method, eventPod("beta", "bad", "bad"), "Rejected")
			emitEvent(manager.Get(ctx, "alpha"), method, eventPod("", "bad-default", "bad-default"), "Rejected")
			emitEvent(manager.Get(ctx, "default"), method, eventPod("beta", "bad-beta", "bad-beta"), "Rejected")
			emitEvent(manager.Get(ctx, "alpha"), method, eventPod("alpha", "barrier", "barrier"), "Barrier")
			assert.Equal(t, "Barrier", nextEventRequest(t, transport).event.Reason)
		})
	}
}

func TestEventRecorderObjectReferences(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx := logging.TestContext(t.Context())
	objects := []k8sruntime.Object{
		eventPod("alpha", "pod", "pod-uid"),
		&coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Namespace: "alpha", Name: "lease", UID: "lease-uid"}},
		&corev1.ObjectReference{Kind: "Pod", APIVersion: "v1", Namespace: "alpha", Name: "pod-ref", UID: "ref-uid", FieldPath: "spec.containers{main}"},
	}
	// Unstructured CronWorkflow is the malformed cron caller's actual input;
	// Workflow and WorkflowEventBinding exercise the CRD reference fallback.
	for _, kind := range []string{"Workflow", "CronWorkflow", "WorkflowEventBinding"} {
		objects = append(objects, &unstructured.Unstructured{Object: map[string]any{"apiVersion": "argoproj.io/v1alpha1", "kind": kind, "metadata": map[string]any{"namespace": "alpha", "name": kind, "uid": kind + "-uid"}}})
	}
	for _, obj := range objects {
		emitEvent(manager.Get(ctx, "alpha"), "AnnotatedEventf", obj, "Sibling")
		req := nextEventRequest(t, transport)
		assert.NotEmpty(t, req.event.InvolvedObject.Kind)
		assert.NotEmpty(t, req.event.InvolvedObject.APIVersion)
		assert.NotEmpty(t, req.event.InvolvedObject.UID)
		assert.Equal(t, "alpha", req.event.InvolvedObject.Namespace)
		assert.Equal(t, map[string]string{"node": "node-7"}, req.event.Annotations)
		if ref, ok := obj.(*corev1.ObjectReference); ok {
			assert.Equal(t, *ref, req.event.InvolvedObject)
		}
	}
}

func TestEventRecorderCorrelation(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx := logging.TestContext(t.Context())
	for _, ns := range []string{"alpha", "beta"} {
		obj := eventPod(ns, "same", types.UID(ns+"-uid"))
		emitEvent(manager.Get(ctx, ns), "Event", obj, "Repeat")
		first := nextEventRequest(t, transport)
		assert.Equal(t, int32(1), first.event.Count)
		// Wildcard and namespace-bound handles now intentionally share history.
		emitEvent(manager.Get(ctx, ""), "Event", obj, "Repeat")
		second := nextEventRequest(t, transport)
		assert.Equal(t, http.MethodPatch, second.method)
		assert.Equal(t, first.event.Name, second.event.Name)
		assert.Equal(t, int32(2), second.event.Count)
		assert.Equal(t, obj.UID, second.event.InvolvedObject.UID)
	}
	recreated := eventPod("alpha", "same", "new-uid")
	emitEvent(manager.Get(ctx, "alpha"), "Event", recreated, "Repeat")
	assert.Equal(t, http.MethodPost, nextEventRequest(t, transport).method)
	transport.mu.Lock()
	transport.missingPatch = true
	transport.mu.Unlock()
	emitEvent(manager.Get(ctx, "alpha"), "Event", recreated, "Repeat")
	patch := nextEventRequest(t, transport)
	assert.Equal(t, http.MethodPatch, patch.method)
	assert.Equal(t, http.StatusNotFound, patch.status)
	created := nextEventRequest(t, transport)
	assert.Equal(t, http.MethodPost, created.method)
	assert.Equal(t, int32(2), created.event.Count)
	assert.Equal(t, types.UID("new-uid"), created.event.InvolvedObject.UID)
}

func TestEventRecorderConcurrentGetAndRequestCancellation(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() { manager.Get(ctx, fmt.Sprintf("first-%d", i)) })
	}
	wg.Wait()
	cancel()
	emitEvent(manager.Get(logging.TestContext(t.Context()), "alpha"), "Event", eventPod("alpha", "alive", "alive"), "AfterCancel")
	assert.Equal(t, "AfterCancel", nextEventRequest(t, transport).event.Reason)
	before := runtime.NumGoroutine()
	broadcaster := manager.broadcaster
	for i := range 128 {
		wg.Go(func() { manager.Get(ctx, fmt.Sprintf("churn-%d", i)) })
	}
	wg.Wait()
	assert.Same(t, broadcaster, manager.broadcaster)
	require.Eventually(t, func() bool { return runtime.NumGoroutine() <= before+4 }, time.Second, time.Millisecond)
	t.Logf("128 concurrent new namespaces: goroutine delta=%d", runtime.NumGoroutine()-before)
}

func TestEventRecorderAlreadyCanceledFirstRequest(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	cancel()
	first := manager.Get(ctx, "alpha")
	emitEvent(first, "Event", eventPod("alpha", "first", "first"), "AlreadyCanceled")
	assert.Equal(t, "AlreadyCanceled", nextEventRequest(t, transport).event.Reason)
	emitEvent(manager.Get(logging.TestContext(t.Context()), "beta"), "Event", eventPod("beta", "second", "second"), "StillAlive")
	assert.Equal(t, "StillAlive", nextEventRequest(t, transport).event.Reason)
}

func TestEventRecorderLoggingDoesNotRetainFirstRequest(t *testing.T) {
	manager, transport := newTestEventManager(t)
	componentHook := logging.NewTestHook()
	manager.logger = logr.New(&logrSink{logger: logging.NewTestLogger(logging.Info, logging.Text, componentHook)})
	firstHook, secondHook := logging.NewTestHook(), logging.NewTestHook()
	first := logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Info, logging.Text, firstHook).WithField("request", "first"))
	second := logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Info, logging.Text, secondHook).WithField("request", "second"))
	emitEvent(manager.Get(first, "alpha"), "Event", eventPod("alpha", "first", "first"), "First")
	nextEventRequest(t, transport)
	emitEvent(manager.Get(second, "beta"), "Event", eventPod("beta", "second", "second"), "Second")
	nextEventRequest(t, transport)
	require.Eventually(t, func() bool { return len(componentHook.AllEntries()) == 2 }, time.Second, time.Millisecond)
	entries := componentHook.AllEntries()
	for _, entry := range entries {
		assert.NotContains(t, entry.Fields, "request")
	}
	assert.Equal(t, klog.KRef("beta", "second"), entries[1].Fields["object"])
	assert.Empty(t, firstHook.AllEntries())
	assert.Empty(t, secondHook.AllEntries())
	// Validation errors still use the logger belonging to the current caller.
	manager.Get(second, "beta").Event(eventPod("beta", "bad", "bad"), "invalid-type", "Invalid", "bad")
	require.Len(t, secondHook.AllEntries(), 1)
	assert.Equal(t, "second", secondHook.LastEntry().Fields["request"])
	assert.Empty(t, firstHook.AllEntries())
}

func TestEventRecorderSharedLRUEviction(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx := logging.TestContext(t.Context())
	original := eventPod("alpha", "original", "original")
	emitEvent(manager.Get(ctx, "alpha"), "Event", original, "Evict")
	nextEventRequest(t, transport)
	// client-go's default correlator caches hold 4096 entries. Serial delivery
	// avoids conflating cache eviction with queue drops under a traffic burst.
	for i := range 4097 {
		ns := fmt.Sprintf("ns-%d", i%2)
		emitEvent(manager.Get(ctx, ns), "Event", eventPod(ns, fmt.Sprintf("pod-%d", i), types.UID(fmt.Sprintf("uid-%d", i))), "Evict")
		nextEventRequest(t, transport)
	}
	emitEvent(manager.Get(ctx, "alpha"), "Event", original, "Evict")
	req := nextEventRequest(t, transport)
	assert.Equal(t, http.MethodPost, req.method)
	assert.Equal(t, int32(1), req.event.Count)
}

func TestEventRecorderBackpressureAndRejectedEvent(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx := logging.TestContext(t.Context())
	release := make(chan struct{})
	transport.block = release
	transport.entered = make(chan struct{}, 1)
	// Release before manager cleanup even if an assertion fails.
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	emitEvent(manager.Get(ctx, "alpha"), "Event", eventPod("alpha", "blocked", "blocked"), "Blocked")
	select {
	case <-transport.entered:
	case <-time.After(time.Second):
		t.Fatal("sink did not start")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 3000 {
			ns := fmt.Sprintf("ns-%d", i%2)
			manager.Get(ctx, ns).Event(eventPod(ns, "overload", "overload"), corev1.EventTypeNormal, "Overload", "message")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("recorder blocked behind sink")
	}
	unblock()
	// A separate manager tests finite server rejection without leaving thousands
	// of queued events ahead of the observation barrier.
	other, rejections := newTestEventManager(t)
	rejections.rejectNext = true
	emitEvent(other.Get(ctx, "alpha"), "Event", eventPod("alpha", "rejected", "rejected"), "Rejected")
	assert.Equal(t, http.StatusForbidden, nextEventRequest(t, rejections).status)
	emitEvent(other.Get(ctx, "beta"), "Event", eventPod("beta", "after", "after"), "AfterRejection")
	assert.Equal(t, "AfterRejection", nextEventRequest(t, rejections).event.Reason)
}
