package resource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSummaryAge(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC))
	minuteAgo := metav1.NewTime(now.Add(-time.Minute))
	epoch := metav1.NewTime(time.Unix(0, 0))

	tests := []struct {
		name  string
		state corev1.ContainerState
		want  time.Duration
	}{
		{name: "no state"},
		{name: "waiting", state: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}},
		{name: "running", state: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: minuteAgo}}},
		{name: "missing timestamps", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}},
		{name: "missing start", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: now}}},
		{name: "missing finish", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: minuteAgo}}},
		{name: "CRI zero start", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: epoch, FinishedAt: now}}},
		{name: "CRI zero finish", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: metav1.NewTime(time.Unix(-60, 0)), FinishedAt: epoch}}},
		{name: "CRI zero start in another timezone", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: metav1.NewTime(epoch.In(time.FixedZone("offset", 3600))), FinishedAt: now}}},
		{name: "finish before start", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: now, FinishedAt: minuteAgo}}},
		{name: "equal timestamps", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: now, FinishedAt: now}}},
		{name: "valid timestamps", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: minuteAgo, FinishedAt: now}}, want: time.Minute},
		{name: "subsecond duration", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: now, FinishedAt: metav1.NewTime(now.Add(time.Millisecond))}}, want: time.Millisecond},
		{name: "nanosecond after CRI zero", state: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{StartedAt: metav1.NewTime(time.Unix(0, 1)), FinishedAt: metav1.NewTime(time.Unix(0, 2))}}, want: time.Nanosecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, (Summary{ContainerState: tt.state}).age())
		})
	}
}
