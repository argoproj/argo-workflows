//go:build executor
// +build executor

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

const kill2xDuration = 70 * time.Second

// Tests the use of signals to kill containers.
// argoproj/argosay:v2 does not contain sh, so you must use argoproj/argosay:v1.
// Killing often requires SIGKILL, which is issued 30s after SIGTERM. So tests need longer (>30s) timeout.
type SignalsSuite struct {
	fixtures.E2ESuite
}

func (s *SignalsSuite) TestStopBehavior() {
	s.Given().
		Workflow("@functional/stop-terminate.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod, kill2xDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyStop).
		WaitForWorkflow(kill2xDuration).
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, []wfv1.WorkflowPhase{wfv1.WorkflowFailed, wfv1.WorkflowError}, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A")
			if assert.NotNil(t, nodeStatus) {
				assert.Contains(t, []wfv1.NodePhase{wfv1.NodeFailed, wfv1.NodeError}, nodeStatus.Phase)
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
		WaitForWorkflow(fixtures.ToHaveRunningPod, kill2xDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyTerminate).
		WaitForWorkflow(kill2xDuration).
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, []wfv1.WorkflowPhase{wfv1.WorkflowFailed, wfv1.WorkflowError}, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A")
			if assert.NotNil(t, nodeStatus) {
				assert.Contains(t, []wfv1.NodePhase{wfv1.NodeFailed, wfv1.NodeError}, nodeStatus.Phase)
			}
			nodeStatus = status.Nodes.FindByDisplayName("A.onExit")
			assert.Nil(t, nodeStatus)
			nodeStatus = status.Nodes.FindByDisplayName(m.Name + ".onExit")
			assert.Nil(t, nodeStatus)
		})
}

// Tests that new pods are never created once a stop shutdown strategy has been added
func (s *SignalsSuite) TestDoNotCreatePodsUnderStopBehavior() {
	s.Given().
		Workflow("@functional/stop-terminate-2.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod, kill2xDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyStop).
		WaitForWorkflow(kill2xDuration).
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			}
			nodeStatus = status.Nodes.FindByDisplayName("B")
			assert.Nil(t, nodeStatus)
		})
}

func (s *SignalsSuite) TestSidecars() {
	s.Given().
		Workflow("@testdata/sidecar-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, kill2xDuration)
}

// make sure Istio/Anthos and other sidecar injectors will work
func (s *SignalsSuite) TestInjectedSidecar() {
	s.Need(fixtures.None(fixtures.Emissary)) // emissary cannot kill this
	s.Given().
		Workflow("@testdata/sidecar-injected-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, kill2xDuration)
}

func (s *SignalsSuite) TestInjectedSidecarKillAnnotation() {
	s.Given().
		Workflow("@testdata/sidecar-injected-kill-annotation-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, kill2xDuration)
}

func TestSignalsSuite(t *testing.T) {
	suite.Run(t, new(SignalsSuite))
}
