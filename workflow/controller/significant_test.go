package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_significantPodChange(t *testing.T) {
	assert.False(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{}), "No change")
	t.Run("Spec", func(t *testing.T) {
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{Spec: corev1.PodSpec{NodeName: "from"}}), "Node name change")

	})
	t.Run("Status", func(t *testing.T) {
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}), "Phase change")
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{Status: corev1.PodStatus{PodIP: "my-ip"}}), "Pod IP change")
	})
	t.Run("ContainerStatuses", func(t *testing.T) {
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{}}}}), "Number of container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}}},
		), "Ready of container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}}}}},
		), "Waiting of container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "my-reason"}}}}}},
		), "Waiting reason of container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Message: "my-message"}}}}}},
		), "Waiting message of container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Message: "my-message"}}}}}},
		), "Waiting message of container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}}},
		), "Running container status changes")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{}}}},
			&corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}}},
		), "Terminate container status changes")
	})
	t.Run("InitContainerStatuses", func(t *testing.T) {
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{Status: corev1.PodStatus{InitContainerStatuses: []corev1.ContainerStatus{{}}}}), "Number of container status changes")
	})
}
