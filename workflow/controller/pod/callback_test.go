package pod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestPodEventCallbackReceivesOldPod checks that only updates pass the previous
// pod, so the workflow controller can tell a phase transition from an add or delete.
func TestPodEventCallbackReceivesOldPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	type call struct{ oldPod, newPod *apiv1.Pod }
	var calls []call
	c := &Controller{callBack: func(_ context.Context, oldPod, newPod *apiv1.Pod) error {
		calls = append(calls, call{oldPod, newPod})
		return nil
	}}
	running := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}, Status: apiv1.PodStatus{Phase: apiv1.PodRunning}}
	succeeded := running.DeepCopy()
	succeeded.Status.Phase = apiv1.PodSucceeded

	c.addPodEvent(ctx, running)
	c.updatePodEvent(ctx, running, succeeded)
	c.deletePodEvent(ctx, succeeded)

	assert.Equal(t, []call{{nil, running}, {running, succeeded}, {nil, succeeded}}, calls)
}
