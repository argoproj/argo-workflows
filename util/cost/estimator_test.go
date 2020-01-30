package cost

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEstimateCost(t *testing.T) {
	now := time.Now()
	zero := now.Add(-time.Since(now))

	tests := []struct {
		name string
		pod  *v1.Pod
		want int64
	}{
		{"Empty", &v1.Pod{}, 0},
		{"WaitingContainerWithDefaults", &v1.Pod{
			Spec: v1.PodSpec{Containers: []v1.Container{{Name: "main"}}},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{{Name: "main"}},
			},
		}, 0},
		{"RunningContainerWithDefaults", &v1.Pod{
			Spec: v1.PodSpec{Containers: []v1.Container{{Name: "main"}}},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name: "main",
						State: v1.ContainerState{
							Running: &v1.ContainerStateRunning{
								StartedAt: metav1.Time{
									Time: zero.Add(-1 * time.Minute),
								},
							},
						},
					},
				},
			},
		}, 4},
		{"TerminatedContainerWithDefaults", &v1.Pod{
			Spec: v1.PodSpec{Containers: []v1.Container{{Name: "main"}}},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name: "main",
						State: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								StartedAt:  metav1.Time{Time: zero},
								FinishedAt: metav1.Time{Time: zero.Add(2 * time.Minute)},
							},
						},
					},
				},
			},
		}, 8},
		{"TerminatedContainerWithCPURequest", &v1.Pod{
			Spec: v1.PodSpec{Containers: []v1.Container{{Name: "main", Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("2000m"),
				},
			}}}},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name: "main",
						State: v1.ContainerState{
							Running: &v1.ContainerStateRunning{
								StartedAt: metav1.Time{
									Time: zero.Add(-1 * time.Minute),
								},
							},
						},
					},
				},
			},
		}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EstimateCost(tt.pod, zero); got != tt.want {
				t.Errorf("EstimateCost() = %v, want %v", got, tt.want)
			}
		})
	}
}
