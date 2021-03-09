// +build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type SemaphoreSuite struct {
	fixtures.E2ESuite
}

func (s *SemaphoreSuite) TestSynchronizationWfLevelMutex() {
	s.Given().
		Workflow("@functional/synchronization-mutex-wf-level-1.yaml").
		When().
		SubmitWorkflow().
		Given().
		Workflow("@functional/synchronization-mutex-wf-level.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeWaitingOnAMutex).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *SemaphoreSuite) TestTemplateLevelMutex() {
	s.Given().
		Workflow("@functional/synchronization-mutex-tmpl-level.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeWaitingOnAMutex).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *SemaphoreSuite) TestWorkflowLevelSemaphore() {
	s.Given().
		Workflow("@testdata/semaphore-wf-level.yaml").
		When().
		CreateConfigMap("my-config", map[string]string{"workflow": "1"}).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHavePhase(wfv1.WorkflowUnknown)).
		WaitForWorkflow().
		DeleteConfigMap("my-config").
		Then().
		When().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *SemaphoreSuite) TestTemplateLevelSemaphore() {
	s.Given().
		Workflow("@testdata/semaphore-tmpl-level.yaml").
		When().
		CreateConfigMap("my-config", map[string]string{"template": "1"}).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.True(t, status.Nodes.Any(func(n wfv1.NodeStatus) bool {
				return strings.Contains(n.Message, "Waiting for")
			}))
		}).
		When().
		WaitForWorkflow()
}

func TestSemaphoreSuite(t *testing.T) {
	suite.Run(t, new(SemaphoreSuite))
}
