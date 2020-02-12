package usage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestEstimatePodUsage(t *testing.T) {
	now := time.Now()
	zero := now.Add(-time.Since(now))

	tests := []struct {
		name string
		pod  *corev1.Pod
		want wfv1.Usage
	}{
		{"Empty", &corev1.Pod{}, wfv1.Usage{}},
		{"RunningContainerWithCPURequest", &corev1.Pod{
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
							Running: &corev1.ContainerStateRunning{
								StartedAt: metav1.Time{
									Time: zero.Add(-1 * time.Minute),
								},
							},
						},
					},
				},
			},
		}, wfv1.Usage{
			corev1.ResourceCPU:    2 * time.Minute,
			corev1.ResourceMemory: 1 * time.Minute,
		}},
		{"TerminatedContainerWithCPURequest", &corev1.Pod{
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
							Running: &corev1.ContainerStateRunning{
								StartedAt: metav1.Time{
									Time: zero.Add(-2 * time.Minute),
								},
							},
						},
					},
				},
			},
		}, wfv1.Usage{
			corev1.ResourceCPU:    4 * time.Minute,
			corev1.ResourceMemory: 2 * time.Minute,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimatePodUsage(tt.pod, zero)
			assert.Equal(t, tt.want, got)
		})
	}
}
