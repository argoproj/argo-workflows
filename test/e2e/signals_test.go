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

const killDuration = 2 * time.Minute

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
		WaitForWorkflow(fixtures.ToHaveRunningPod, killDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyStop).
		WaitForWorkflow(killDuration + 15*time.Second). // this one takes especially long in CI
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

func (s *SignalsSuite) TestStopBehaviorWithDaemon() {
	s.Given().
		Workflow("@functional/stop-terminate-daemon.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod, killDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyStop).
		WaitForWorkflow(killDuration).
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, []wfv1.WorkflowPhase{wfv1.WorkflowFailed, wfv1.WorkflowError}, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("Daemon")
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
		WaitForWorkflow(fixtures.ToHaveRunningPod, killDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyTerminate).
		WaitForWorkflow(killDuration).
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
		WaitForWorkflow(fixtures.ToHaveRunningPod, killDuration).
		ShutdownWorkflow(wfv1.ShutdownStrategyStop).
		WaitForWorkflow(killDuration).
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
		WaitForWorkflow(fixtures.ToBeSucceeded, killDuration)
}

// make sure Istio/Anthos and other sidecar injectors will work
func (s *SignalsSuite) TestInjectedSidecar() {
	s.Given().
		Workflow("@testdata/sidecar-injected-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, killDuration)
}

func (s *SignalsSuite) TestSubProcess() {
	s.Given().
		Workflow("@testdata/subprocess-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow()
}

func (s *SignalsSuite) TestSignaled() {
	s.Given().
		Workflow("@testdata/signaled-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Contains(t, status.Message, "(exit code 143)")
		})
}

func (s *SignalsSuite) TestSignaledContainerSet() {
	s.Given().
		Workflow("@testdata/signaled-container-set-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Contains(t, status.Message, "(exit code 137)")
			one := status.Nodes.FindByDisplayName("one")
			if assert.NotNil(t, one) {
				assert.Equal(t, wfv1.NodeFailed, one.Phase)
				assert.Contains(t, one.Message, "(exit code 137)")
			}
			two := status.Nodes.FindByDisplayName("two")
			if assert.NotNil(t, two) {
				assert.Equal(t, wfv1.NodeFailed, two.Phase)
				assert.Contains(t, two.Message, "(exit code 143)")
			}
		})
}

func TestSignalsSuite(t *testing.T) {
	suite.Run(t, new(SignalsSuite))
}
