package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestMainContainerNeverStarted(t *testing.T) {
	tests := []struct {
		name     string
		pod      *apiv1.Pod
		tmpl     *wfv1.Template
		expected bool
	}{
		{
			name: "pod with no container statuses (never scheduled)",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase:             apiv1.PodFailed,
					ContainerStatuses: []apiv1.ContainerStatus{},
				},
			},
			tmpl:     nil,
			expected: true,
		},
		{
			name: "main container in waiting state",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodFailed,
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Waiting: &apiv1.ContainerStateWaiting{
									Reason:  "ContainerCreating",
									Message: "Container is creating",
								},
							},
						},
					},
				},
			},
			tmpl:     nil,
			expected: true,
		},
		{
			name: "main container ran and terminated",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodFailed,
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Terminated: &apiv1.ContainerStateTerminated{
									ExitCode:   1,
									StartedAt:  metav1.Now(),
									FinishedAt: metav1.Now(),
								},
							},
						},
					},
				},
			},
			tmpl:     nil,
			expected: false,
		},
		{
			name: "main container was running",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodFailed,
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Running: &apiv1.ContainerStateRunning{
									StartedAt: metav1.Now(),
								},
							},
						},
					},
				},
			},
			tmpl:     nil,
			expected: false,
		},
		{
			name: "main container waiting for pod initializing",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodFailed,
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Waiting: &apiv1.ContainerStateWaiting{
									Reason: "PodInitializing",
								},
							},
						},
					},
				},
			},
			tmpl:     nil,
			expected: true,
		},
		{
			name: "main container terminated but never had startedAt",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodFailed,
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Terminated: &apiv1.ContainerStateTerminated{
									ExitCode: 137,
									Reason:   "OOMKilled",
									// No StartedAt - container was killed before starting
								},
							},
						},
					},
				},
			},
			tmpl:     nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mainContainerNeverStarted(tt.pod, tt.tmpl)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRestartableReason(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected bool
	}{
		{
			name:     "Evicted",
			reason:   "Evicted",
			expected: true,
		},
		{
			name:     "NodeShutdown",
			reason:   "NodeShutdown",
			expected: true,
		},
		{
			name:     "NodeAffinity",
			reason:   "NodeAffinity",
			expected: true,
		},
		{
			name:     "UnexpectedAdmissionError",
			reason:   "UnexpectedAdmissionError",
			expected: true,
		},
		{
			name:     "OOMKilled is not restartable",
			reason:   "OOMKilled",
			expected: false,
		},
		{
			name:     "Error is not restartable",
			reason:   "Error",
			expected: false,
		},
		{
			name:     "empty reason",
			reason:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRestartableReason(tt.reason)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzePodForRestart(t *testing.T) {
	tests := []struct {
		name     string
		pod      *apiv1.Pod
		tmpl     *wfv1.Template
		expected bool
	}{
		{
			name: "running pod should not restart",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodRunning,
				},
			},
			expected: false,
		},
		{
			name: "succeeded pod should not restart",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase: apiv1.PodSucceeded,
				},
			},
			expected: false,
		},
		{
			name: "evicted pod that never started should restart",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase:   apiv1.PodFailed,
					Reason:  "Evicted",
					Message: "The node had condition: [DiskPressure]",
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Waiting: &apiv1.ContainerStateWaiting{
									Reason: "ContainerCreating",
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "evicted pod that ran should not restart",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase:   apiv1.PodFailed,
					Reason:  "Evicted",
					Message: "The node had condition: [DiskPressure]",
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Terminated: &apiv1.ContainerStateTerminated{
									ExitCode:   137,
									StartedAt:  metav1.Now(),
									FinishedAt: metav1.Now(),
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "failed pod with non-restartable reason should not restart",
			pod: &apiv1.Pod{
				Status: apiv1.PodStatus{
					Phase:  apiv1.PodFailed,
					Reason: "OOMKilled",
					ContainerStatuses: []apiv1.ContainerStatus{
						{
							Name: common.MainContainerName,
							State: apiv1.ContainerState{
								Waiting: &apiv1.ContainerStateWaiting{},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzePodForRestart(tt.pod, tt.tmpl)
			assert.Equal(t, tt.expected, result)
		})
	}
}
