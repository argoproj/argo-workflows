package events

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type eventLogBuffer struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *eventLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func (b *eventLogBuffer) snapshot() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}

// Keep the original callable signature, including assignment to a function value.
func TestEventRecorderLegacyConstructor(t *testing.T) {
	var constructor func(kubernetes.Interface) EventRecorderManager = NewEventRecorderManager
	transport := &eventTransport{events: make(map[string]corev1.Event), requests: make(chan eventRequest, 8)}
	client, err := kubernetes.NewForConfigAndClient(&rest.Config{Host: "https://events.invalid"}, &http.Client{Transport: transport})
	require.NoError(t, err)
	manager := constructor(client).(*eventRecorderManager)
	t.Cleanup(func() { manager.broadcaster.Shutdown() })
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	first := manager.Get(ctx, "alpha")
	cancel()
	first.Event(eventPod("alpha", "first", "first"), corev1.EventTypeNormal, "First", "first event")
	require.Equal(t, http.StatusCreated, nextEventRequest(t, transport).status)
	second := manager.Get(logging.TestContext(t.Context()), "beta")
	second.Event(eventPod("beta", "second", "second"), corev1.EventTypeNormal, "Second", "second event")
	require.Equal(t, http.StatusCreated, nextEventRequest(t, transport).status)
}

// Use the public constructor, not private logger injection. The component
// configuration must survive independently of the first request and its logger.
func TestEventRecorderConfiguredLogger(t *testing.T) {
	for _, level := range []logging.Level{logging.Info, logging.Error} {
		t.Run(string(level), func(t *testing.T) {
			var output eventLogBuffer
			hook := logging.NewTestHook()
			root := logging.NewSlogLoggerCustom(level, logging.JSON, &output, hook).WithFields(logging.Fields{"component": "configured-root", "logger": "root-name"})
			transport := &eventTransport{events: make(map[string]corev1.Event), requests: make(chan eventRequest, 8)}
			client, err := kubernetes.NewForConfigAndClient(&rest.Config{Host: "https://events.invalid"}, &http.Client{Transport: transport})
			require.NoError(t, err)
			manager := NewEventRecorderManagerWithLogger(client, root).(*eventRecorderManager)
			t.Cleanup(func() { manager.broadcaster.Shutdown() })
			requestHook := logging.NewTestHook()
			requestCtx, cancel := context.WithCancel(logging.WithLogger(t.Context(), root.WithField("request", "first")))
			first := manager.Get(requestCtx, "alpha")
			cancel()
			first.Event(eventPod("alpha", "first", "first"), corev1.EventTypeNormal, "First", "first event")
			nextEventRequest(t, transport)
			secondCtx := logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Debug, logging.Text, requestHook).WithField("request", "second"))
			second := manager.Get(secondCtx, "beta")
			second.Event(eventPod("beta", "second", "second"), corev1.EventTypeNormal, "Second", "second event")
			nextEventRequest(t, transport)
			// The sink error proves error-level output is still routed, not just muted.
			transport.mu.Lock()
			transport.rejectNext = true
			transport.mu.Unlock()
			second.Event(eventPod("beta", "rejected", "rejected"), corev1.EventTypeNormal, "Rejected", "rejected event")
			require.Equal(t, http.StatusForbidden, nextEventRequest(t, transport).status)
			require.Eventually(t, func() bool {
				return strings.Contains(output.snapshot(), `"level":"ERROR"`)
			}, 3*time.Second, time.Millisecond, "sink error bypassed configured JSON logger")
			require.Eventually(t, func() bool { return len(hook.AllEntries()) >= 4 }, time.Second, time.Millisecond)
			entries := hook.AllEntries()
			for _, entry := range entries {
				assert.Equal(t, "configured-root", entry.Fields["component"])
				assert.Equal(t, "root-name", entry.Fields["logger"])
				assert.NotContains(t, entry.Fields, "request")
			}
			assert.Empty(t, requestHook.AllEntries())
			if level == logging.Info {
				require.Eventually(t, func() bool { return strings.Count(output.snapshot(), `"msg":"Event occurred"`) == 3 }, time.Second, time.Millisecond)
			}
			lines := strings.Split(strings.TrimSpace(output.snapshot()), "\n")
			infoCount := 0
			for _, line := range lines {
				var entry map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &entry), "actual output must be JSON")
				assert.Equal(t, "configured-root", entry["component"])
				assert.Equal(t, "root-name", entry["logger"])
				assert.NotContains(t, entry, "request")
				if entry["level"] == "INFO" {
					infoCount++
				}
				if level == logging.Error {
					assert.Equal(t, "ERROR", entry["level"])
				}
			}
			if level == logging.Info {
				assert.Equal(t, 3, infoCount)
			} else {
				assert.Zero(t, infoCount)
			}
		})
	}
}
