// +build fmea

package fmea

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type FMEASuite struct {
	fixtures.E2ESuite
}

// TODO - is this a valid test?
func (s *FMEASuite) TestCoreDNSDeleted() {
	s.Given().
		Workflow("@testdata/sleepy-workflow.yaml").
		When().
		SubmitWorkflow().
		Exec("kubectl", []string{"-n", "kube-system", "delete", "pod", "-l", "k8s-app=kube-dns"}, fixtures.OutputContains(`pod \"coredns-`)).
		Wait(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *FMEASuite) TestWorkflowControllerDeleted() {
	s.Given().
		Workflow("@testdata/sleepy-workflow.yaml").
		When().
		SubmitWorkflow().
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "-l", "app=workflow-controller"}, fixtures.OutputContains(`pod \"workflow-controller`)).
		WaitForWorkflowToStart(15 * time.Second).
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
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "minio"}, fixtures.NoError).
		WaitForWorkflow(15 * time.Second).
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
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "mysql"}, fixtures.NoError).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *FMEASuite) TestNoAvailableNodes() {
	s.Given().
		Workflow("@testdata/node-selector-workflow.yaml").
		When().
		Exec("kubectl", []string{"label", "node", "--all", "fmea-"}, fixtures.NoError).
		SubmitWorkflow().
		Wait(15*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeRunning, status.Phase)
		}).
		When().
		Exec("kubectl", []string{"label", "node", "--all", "fmea=true"}, fixtures.NoError).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func TestFMEASuite(t *testing.T) {
	suite.Run(t, new(FMEASuite))
}
