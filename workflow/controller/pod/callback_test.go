package pod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestPodEventCallbackReceivesOldPod drives the handlers the informer registers and
// checks that only significant updates pass the previous pod, in order, so the
// workflow controller can tell a phase transition from an add or delete.
func TestPodEventCallbackReceivesOldPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	type call struct{ oldPod, newPod *apiv1.Pod }
	var calls []call
	c := &Controller{
		log: logging.RequireLoggerFromContext(ctx),
		callBack: func(_ context.Context, oldPod, newPod *apiv1.Pod) error {
			calls = append(calls, call{oldPod, newPod})
			return nil
		},
	}
	handlers := c.eventHandlers(ctx)

	running := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default", ResourceVersion: "1"}, Status: apiv1.PodStatus{Phase: apiv1.PodRunning}}
	succeeded := running.DeepCopy()
	succeeded.ResourceVersion = "2"
	succeeded.Status.Phase = apiv1.PodSucceeded
	sameVersion := succeeded.DeepCopy()
	insignificant := succeeded.DeepCopy()
	insignificant.ResourceVersion = "3"

	handlers.OnAdd("not a pod", false)
	handlers.OnUpdate("not a pod", "not a pod") // no object key
	handlers.OnAdd(running, false)
	handlers.OnUpdate(running, succeeded)
	handlers.OnUpdate(succeeded, sameVersion)   // resync: same resource version
	handlers.OnUpdate(succeeded, insignificant) // nothing significant changed
	handlers.OnDelete(succeeded)
	handlers.OnDelete(cache.DeletedFinalStateUnknown{Key: "default/pod", Obj: succeeded})

	assert.Equal(t, []call{{nil, running}, {running, succeeded}, {nil, succeeded}, {nil, succeeded}}, calls)
}
