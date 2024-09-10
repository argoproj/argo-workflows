package resource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
