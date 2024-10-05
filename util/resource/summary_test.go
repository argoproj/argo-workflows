package resource

import (
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

func TestSummaries_Duration(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	finishTime := time.Now()

	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}

	summaries := Summaries{
		"container1": {
			ResourceList: resourceList,
			ContainerState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					StartedAt:  metav1.NewTime(startTime),
					FinishedAt: metav1.NewTime(finishTime),
				},
			},
		},
	}

	expectedDuration := wfv1.ResourcesDuration{}
	expectedDuration = expectedDuration.Add(wfv1.ResourcesDuration{
		corev1.ResourceCPU:    wfv1.NewResourceDuration(3600 * time.Second),
		corev1.ResourceMemory: wfv1.NewResourceDuration(36864 * time.Second),
	})

	assert.Equal(t, expectedDuration, summaries.Duration())
}

func TestSummaries_Duration_StartedAtIsEpoch(t *testing.T) {
	startedTime, err := time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	require.NoError(t, err)
	finishTime := time.Now()

	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}

	summaries := Summaries{
		"container1": {
			ResourceList: resourceList,
			ContainerState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					StartedAt:  metav1.NewTime(startedTime),
					FinishedAt: metav1.NewTime(finishTime),
				},
			},
		},
	}

	expectedDuration := wfv1.ResourcesDuration{}
	expectedDuration = expectedDuration.Add(wfv1.ResourcesDuration{
		corev1.ResourceCPU:    wfv1.NewResourceDuration(0 * time.Second),
		corev1.ResourceMemory: wfv1.NewResourceDuration(0 * time.Second),
	})
	assert.Equal(t, expectedDuration, summaries.Duration())
}
