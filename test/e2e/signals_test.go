// +build executor

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/v3/test/e2e/fixtures"
)

type SignalsSuite struct {
	fixtures.E2ESuite
}

func (s *SignalsSuite) SetupSuite() {
	s.E2ESuite.SetupSuite()
	// Because k8ssapi and kubelet execute `sh -c 'kill 15 1'` to they do not work.
	s.Need(fixtures.None(fixtures.K8SAPI, fixtures.Kubelet))
}

func (s *SignalsSuite) TestStopBehavior() {
	s.Given().
		Workflow("@functional/stop-terminate.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart, "to start").
		RunCli([]string{"stop", "@latest"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Regexp(t, "workflow stop-terminate-.* stopped", output)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			}
			nodeStatus = status.Nodes.FindByDisplayName("A.onExit")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			}
			nodeStatus = status.Nodes.FindByDisplayName(m.Name + ".onExit")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			}
		})
}

func (s *SignalsSuite) TestTerminateBehavior() {
	s.Given().
		Workflow("@functional/stop-terminate.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart, "to start").
		RunCli([]string{"terminate", "@latest"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Regexp(t, "workflow stop-terminate-.* terminated", output)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			}
			nodeStatus = status.Nodes.FindByDisplayName("A.onExit")
			assert.Nil(t, nodeStatus)
			nodeStatus = status.Nodes.FindByDisplayName(m.Name + ".onExit")
			assert.Nil(t, nodeStatus)
		})
}

func (s *SignalsSuite) TestPropagateMaxDuration() {
	s.Need(fixtures.None(fixtures.PNS)) // does not work on PNS on CI for some reason
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-backoff-2
  labels:
    argo-e2e: true
spec:
  entrypoint: retry-backoff
  templates:
  - name: retry-backoff
    retryStrategy:
      limit: 10
      backoff:
        duration: "1"
        factor: 1
        maxDuration: "10"
    container:
      image: argoproj/argosay:v1
      command: [sh, -c]
      args: ["sleep $(( {{retries}} * 40 )); exit 1"]

`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(45 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Len(t, status.Nodes, 3)
			node := status.Nodes.FindByDisplayName("retry-backoff-2(1)")
			if assert.NotNil(t, node) {
				assert.Equal(t, wfv1.NodeFailed, node.Phase)
			}
		})
}

func (s *SignalsSuite) TestSidecars() {
	s.Given().
		Workflow("@testdata/sidecar-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func TestSignalsSuite(t *testing.T) {
	suite.Run(t, new(SignalsSuite))
}
