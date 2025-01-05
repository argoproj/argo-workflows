//go:build functional
// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type TestSuspendSitue struct {
	fixtures.E2ESuite
}

func (s *TestSuspendSitue) TestSuspendNodeTimeoutWithoutDefaultValue() {
	s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-node-timeout-without-default-value
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: approve
        template: approve
    - - name: release
        template: whalesay
        arguments:
          parameters:
            - name: message
              value: "{{steps.approve.outputs.parameters.message}}"
  - name: approve
    suspend:
      duration: 5s
    outputs:
      parameters:
        - name: message
          valueFrom:
            supplied: {}
  - name: whalesay
    inputs:
      parameters:
        - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Contains(t, "raw output parameter 'message' has not been set and does not have a default value", status.Message)
		})
}

func (s *TestSuspendSitue) TestSuspendNodeTimeoutWithDefaultValue() {
	s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-node-timeout-with-default-value
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: approve
        template: approve
    - - name: release
        template: whalesay
        arguments:
          parameters:
            - name: message
              value: "{{steps.approve.outputs.parameters.message}}"
  - name: approve
    suspend:
      duration: 5s
    outputs:
      parameters:
        - name: message
          valueFrom:
            default: default message
            supplied: {}
  - name: whalesay
    inputs:
      parameters:
        - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, wfv1.Progress("2/2"))
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return status.Name == "suspend-node-timeout-with-default-value[0].approve"
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Len(t, status.Outputs.Parameters, 1)
			assert.Equal(t, "message", status.Outputs.Parameters[0].Name)
			assert.Equal(t, wfv1.AnyStringPtr("default message"), status.Outputs.Parameters[0].Value)
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return status.Name == "suspend-node-timeout-with-default-value[1].release"
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Len(t, status.Inputs.Parameters, 1)
			assert.Equal(t, "message", status.Inputs.Parameters[0].Name)
			assert.Equal(t, wfv1.AnyStringPtr("default message"), status.Inputs.Parameters[0].Value)
		})
}
