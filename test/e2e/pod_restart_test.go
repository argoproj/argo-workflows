//go:build functional

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
)

type PodRestartSuite struct {
	fixtures.E2ESuite
}

// TestEvictedPodRestarts tests that a pod which is evicted before the main container
// starts is automatically restarted and the workflow eventually succeeds.
func (s *PodRestartSuite) TestEvictedPodRestarts() {
	var firstPodName string

	s.Given().
		Workflow(`@testdata/workflow-pod-restart.yaml`).
		When().
		SubmitWorkflow().
		// Wait for the pod to be created and init container to start
		WaitForPod(func(p *corev1.Pod) bool {
			// Wait until pod exists and is running (init container running)
			if p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodPending {
				return false
			}
			firstPodName = p.Name
			return true
		}).
		And(func() {
			// Patch the pod status to simulate an eviction before main container started
			ctx := context.Background()

			// The patch simulates what happens when a pod is evicted:
			// - Phase becomes Failed
			// - Reason is set to "Evicted"
			// - Message describes the eviction cause
			// - Main container never entered Running state (still in Waiting)
			patch := map[string]any{
				"status": map[string]any{
					"phase":   "Failed",
					"reason":  "Evicted",
					"message": "The node had condition: [DiskPressure]",
					"initContainerStatuses": []map[string]any{
						{
							"name":  "init",
							"image": "alpine:latest",
							"state": map[string]any{
								"terminated": map[string]any{
									"exitCode": 0,
									"reason":   "Completed",
								},
							},
							"ready":        true,
							"restartCount": 0,
						},
						{
							"name":  "delay",
							"image": "alpine:latest",
							"state": map[string]any{
								"terminated": map[string]any{
									"exitCode": 137,
									"reason":   "Error",
								},
							},
							"ready":        false,
							"restartCount": 0,
						},
					},
					"containerStatuses": []map[string]any{
						{
							"name":  "main",
							"image": "alpine:latest",
							"state": map[string]any{
								"waiting": map[string]any{
									"reason": "PodInitializing",
								},
							},
							"ready":        false,
							"restartCount": 0,
						},
					},
				},
			}
			patchBytes, err := json.Marshal(patch)
			s.Require().NoError(err)

			_, err = s.KubeClient.CoreV1().Pods(fixtures.Namespace).Patch(
				ctx,
				firstPodName,
				types.MergePatchType,
				patchBytes,
				metav1.PatchOptions{},
				"status",
			)
			s.Require().NoError(err)
			s.T().Logf("Patched pod %s to simulate eviction", firstPodName)
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded, 60*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, pod *corev1.Pod) {
			// Verify that FailedPodRestarts was incremented
			assert.Equal(t, int32(1), n.FailedPodRestarts, "expected FailedPodRestarts to be 1")

			// Verify the main container ran successfully by checking its exit code
			require.NotNil(t, pod, "expected pod to exist")
			var mainContainerFound bool
			for _, c := range pod.Status.ContainerStatuses {
				if c.Name == "main" {
					mainContainerFound = true
					require.NotNil(t, c.State.Terminated, "expected main container to be terminated")
					assert.Equal(t, int32(0), c.State.Terminated.ExitCode, "expected main container to exit with code 0")
				}
			}
			assert.True(t, mainContainerFound, "expected to find main container status")
		})
}

func TestPodRestartSuite(t *testing.T) {
	suite.Run(t, new(PodRestartSuite))
}
