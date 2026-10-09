package events

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestEventRecorderAnnotationAggregation(t *testing.T) {
	for _, enabled := range []string{"false", "true"} {
		t.Run(enabled, func(t *testing.T) {
			t.Setenv(aggregationWithAnnotationsEnvKey, enabled)
			manager, transport := newTestEventManager(t)
			ctx := logging.TestContext(t.Context())
			obj := eventPod("alpha", "annotated", "annotated")
			var last eventRequest
			for i := range 10 {
				annotations := map[string]string{"node": fmt.Sprintf("node-%d", i%2)}
				manager.Get(ctx, "alpha").AnnotatedEventf(obj, annotations, corev1.EventTypeNormal, "Annotated", "message %d", i)
				last = nextEventRequest(t, transport)
				if enabled == "false" && i == 9 {
					// client-go creates a fresh aggregate Event without annotations.
					assert.Empty(t, last.event.Annotations)
				} else {
					assert.Equal(t, annotations, last.event.Annotations)
				}
			}
			// With annotations enabled, each node has only five distinct messages;
			// without them the tenth message reaches client-go's aggregation threshold.
			assert.Equal(t, enabled == "false", strings.HasPrefix(last.event.Message, "(combined from similar events):"))
		})
	}
}

func TestEventRecorderBurstIsPerObject(t *testing.T) {
	manager, transport := newTestEventManager(t)
	ctx := logging.TestContext(t.Context())
	recorder := manager.Get(ctx, "alpha")
	obj := eventPod("alpha", "burst", "alpha-uid")
	for range defaultSpamBurst {
		recorder.Event(obj, corev1.EventTypeNormal, "Burst", "same message")
		nextEventRequest(t, transport)
	}
	// The next event exhausts this object's bucket, but must not consume the
	// budget for a different namespace or a recreated object with a new UID.
	recorder.Event(obj, corev1.EventTypeNormal, "Burst", "same message")
	manager.Get(ctx, "beta").Event(eventPod("beta", "burst", "beta-uid"), corev1.EventTypeNormal, "OtherObject", "same message")
	req := nextEventRequest(t, transport)
	assert.Equal(t, "OtherObject", req.event.Reason)
	assert.Equal(t, http.MethodPost, req.method)
	recorder.Event(eventPod("alpha", "burst", "new-uid"), corev1.EventTypeNormal, "Recreated", "same message")
	assert.Equal(t, "Recreated", nextEventRequest(t, transport).event.Reason)
}
