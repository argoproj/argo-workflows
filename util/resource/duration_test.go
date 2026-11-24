package resource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestDurationForPod(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		pod  *corev1.Pod
		want wfv1.ResourcesDuration
	}{
		{"Empty", &corev1.Pod{}, wfv1.ResourcesDuration{}},
		{"ContainerWithCPURequest", &corev1.Pod{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2000m"),
				},
			}}}},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "main",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								StartedAt:  metav1.Time{Time: now.Add(-1 * time.Minute)},
								FinishedAt: metav1.Time{Time: now},
							},
						},
					},
				},
			},
		}, wfv1.ResourcesDuration{
			corev1.ResourceCPU:    wfv1.NewResourceDuration(2 * time.Minute),
			corev1.ResourceMemory: wfv1.NewResourceDuration(1 * time.Minute),
		}},
		{"ContainerWithGPULimit", &corev1.Pod{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2000m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
				},
			}}}},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "main",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								StartedAt:  metav1.Time{Time: now.Add(-3 * time.Minute)},
								FinishedAt: metav1.Time{Time: now},
							},
						},
					},
				},
			},
		}, wfv1.ResourcesDuration{
			corev1.ResourceCPU:                    wfv1.NewResourceDuration(6 * time.Minute),
			corev1.ResourceMemory:                 wfv1.NewResourceDuration(3 * time.Minute),
			corev1.ResourceName("nvidia.com/gpu"): wfv1.NewResourceDuration(3 * time.Minute),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DurationForPod(tt.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDurationForPodTerminatedContainerTimes(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC))
	minuteAgo := metav1.NewTime(now.Add(-time.Minute))
	epoch := metav1.NewTime(time.Unix(0, 0))
	tests := []struct {
		name       string
		startedAt  metav1.Time
		finishedAt metav1.Time
		seconds    int64
	}{
		{name: "missing timestamps"},
		{name: "start error", finishedAt: now},
		{name: "CRI start error", startedAt: epoch, finishedAt: metav1.NewTime(time.Unix(1763519017, 0))},
		{name: "missing finish", startedAt: minuteAgo},
		{name: "CRI zero finish", startedAt: metav1.NewTime(time.Unix(-60, 0)), finishedAt: epoch},
		{name: "finish before start", startedAt: now, finishedAt: minuteAgo},
		{name: "equal timestamps", startedAt: now, finishedAt: now},
		{name: "valid timestamps", startedAt: minuteAgo, finishedAt: now, seconds: 60},
		{name: "valid timestamps near CRI zero", startedAt: metav1.NewTime(time.Unix(0, 1)), finishedAt: metav1.NewTime(time.Unix(60, 1)), seconds: 60},
		{name: "valid timestamps before CRI zero", startedAt: metav1.NewTime(time.Unix(-120, 0)), finishedAt: metav1.NewTime(time.Unix(-60, 0)), seconds: 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "main", Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
						corev1.ResourceCPU:                    resource.MustParse("3900m"),
						corev1.ResourceMemory:                 resource.MustParse("100Mi"),
						corev1.ResourceEphemeralStorage:       resource.MustParse("10Gi"),
						corev1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
					}}},
					{Name: "sidecar"},
				}},
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
					{Name: "main", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
						Reason:     "StartError",
						StartedAt:  tt.startedAt,
						FinishedAt: tt.finishedAt,
					}}},
					{Name: "sidecar", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
						StartedAt:  metav1.NewTime(now.Add(-10 * time.Second)),
						FinishedAt: now,
					}}},
				}},
			}
			assert.Equal(t, wfv1.ResourcesDuration{
				corev1.ResourceCPU:                    wfv1.ResourceDuration(tt.seconds*39/10 + 1),
				corev1.ResourceMemory:                 wfv1.ResourceDuration(tt.seconds + 10),
				corev1.ResourceEphemeralStorage:       wfv1.ResourceDuration(tt.seconds),
				corev1.ResourceName("nvidia.com/gpu"): wfv1.ResourceDuration(tt.seconds),
			}, DurationForPod(pod))
		})
	}
}
