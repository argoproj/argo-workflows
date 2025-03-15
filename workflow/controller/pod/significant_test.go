package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SgnificantPodChange(t *testing.T) {
	t.Run("NoChange", func(t *testing.T) {
		assert.False(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{}))
	})
	t.Run("ALL_POD_CHANGES_SIGNIFICANT", func(t *testing.T) {
		t.Setenv("ALL_POD_CHANGES_SIGNIFICANT", "true")
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{}))
	})
	t.Run("DeletionTimestamp", func(t *testing.T) {
		now := metav1.Now()
		assert.True(t, significantPodChange(&corev1.Pod{}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}}), "deletion timestamp change")
	})
	t.Run("Annotations", func(t *testing.T) {
		assert.True(t, significantPodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}}}), "new annotation")
		assert.True(t, significantPodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}}}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"foo": "baz"}}}), "changed annotation")
		assert.True(t, significantPodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}}}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}}), "deleted annotation")
	})
	t.Run("Labels", func(t *testing.T) {
		assert.True(t, significantPodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}}), "new label")
		assert.True(t, significantPodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "baz"}}}), "changed label")
		assert.True(t, significantPodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}}, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}), "deleted label")
	})
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
	t.Run("Conditions", func(t *testing.T) {
		assert.True(t, significantPodChange(
			&corev1.Pod{},
			&corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{}}}}),
			"condition added")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{}}}},
			&corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Reason: "es"}}}},
		), "condition changed")
		assert.True(t, significantPodChange(
			&corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{}}}},
			&corev1.Pod{},
		), "condition removed")
	})
}
