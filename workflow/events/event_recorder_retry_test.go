package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestEventRecorderRetriesTransportError(t *testing.T) {
	manager, transport := newTestEventManager(t)
	transport.transientError = true
	ctx := logging.TestContext(t.Context())
	emitEvent(manager.Get(ctx, "alpha"), "Event", eventPod("alpha", "retry", "retry"), "Retry")
	first := nextEventRequest(t, transport)
	assert.Zero(t, first.status)
	emitEvent(manager.Get(ctx, "beta"), "Event", eventPod("beta", "after-retry", "after-retry"), "AfterRetry")
	// client-go randomizes its first retry delay below ten seconds. Exercise the
	// production setting rather than injecting a different broadcaster in tests.
	select {
	case retried := <-transport.requests:
		assert.Equal(t, "Retry", retried.event.Reason)
		assert.Equal(t, first.event.Name, retried.event.Name)
	case <-time.After(15 * time.Second):
		t.Fatal("transport error was not retried")
	}
	assert.Equal(t, "AfterRetry", nextEventRequest(t, transport).event.Reason)
}
