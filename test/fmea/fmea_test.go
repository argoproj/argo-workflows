// +build fmea

package fmea

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

// Failure Mode Effect Analysis (FMEA)
type FMEASuite struct {
	fixtures.E2ESuite
}

func (s *FMEASuite) BeforeTest(suiteName, testName string) {
	s.resetTestSystem()
	s.E2ESuite.BeforeTest(suiteName, testName)
}

func (s *FMEASuite) AfterTest(suiteName, testName string) {
	s.resetTestSystem()
	s.E2ESuite.AfterTest(suiteName, testName)
}

func (s *FMEASuite) resetTestSystem() {
	_, err := fixtures.Exec("kubectl", "-n", "argo", "scale", "deploy/minio", "--replicas", "1")
	assert.NoError(s.T(), err)
	_, err = fixtures.Exec("kubectl", "-n", "argo", "scale", "deploy/mysql", "--replicas", "1")
	assert.NoError(s.T(), err)
	_, err = fixtures.Exec("kubectl", "label", "node", "--all", "fmea-")
	assert.NoError(s.T(), err)
}

func (s *FMEASuite) TestWorkflowControllerDeleted() {
	s.Given().
		Workflow("@testdata/sleepy-workflow.yaml").
		When().
		SubmitWorkflow().
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "-l", "app=workflow-controller"}, fixtures.OutputContains(`pod "workflow-controller`)).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *FMEASuite) TestDeletingWorkflowPod() {
	s.Given().
		Workflow("@testdata/sleepy-workflow.yaml").
		When().
		SubmitWorkflow().
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "-l", "workflows.argoproj.io/workflow"}, fixtures.OutputContains(`pod "sleepy" deleted`)).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Equal(t, "pod deleted", status.Message)
		})
}

func (s *FMEASuite) TestArtifactStorageFailure() {
	s.Given().
		Workflow("@testdata/ok-workflow.yaml").
		When().
		SubmitWorkflow().
		Exec("kubectl", []string{"-n", "argo", "scale", "deploy/minio", "--replicas", "0"}, fixtures.NoError).
		WaitForWorkflow(60 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Equal(t, "failed to save outputs: timed out waiting for the condition", status.Message)
		})
}

func (s *FMEASuite) TestDatabaseLost() {
	s.Given().
		Workflow("@testdata/ok-workflow.yaml").
		When().
		SubmitWorkflow().
		Exec("kubectl", []string{"-n", "argo", "scale", "deploy/mysql", "--replicas", "0"}, fixtures.NoError).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
	s.Given().
		Exec("kubectl", []string{"-n", "argo", "scale", "deploy/mysql", "--replicas", "1"}, fixtures.NoError).
		When().
		Wait(15 * time.Second).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			_, err := s.Persistence.WorkflowArchive.GetWorkflow(string(metadata.UID))
			assert.NoError(t, err)
		})
}

func (s *FMEASuite) TestNoAvailableNodes() {
	s.Given().
		Workflow("@testdata/node-selector-workflow.yaml").
		When().
		SubmitWorkflow().
		Wait(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeRunning, status.Phase)
			assert.True(t, status.Nodes.Any(func(node wfv1.NodeStatus) bool {
				return strings.HasPrefix(node.Message, "Unschedulable")
			}))
		})
	s.Given().
		Exec("kubectl", []string{"label", "node", "--all", "fmea=true"}, fixtures.NoError).
		When().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *FMEASuite) TestNoResourcePodsQuota() {
	s.Given().
		Workflow("@testdata/ok-workflow.yaml").
		When().
		PodsQuota(-1).
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.True(t, status.Nodes.Any(func(node wfv1.NodeStatus) bool {
				return node.Phase == wfv1.NodeError && strings.Contains(node.Message, "is forbidden: exceeded quota")
			}))
		})
}

func TestFMEASuite(t *testing.T) {
	suite.Run(t, new(FMEASuite))
}
