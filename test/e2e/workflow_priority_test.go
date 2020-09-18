// +build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type WorkflowPrioritySuite struct {
	fixtures.E2ESuite
}

func (s *WorkflowPrioritySuite) TestPriority() {
	// the first two workflows will start, because there is
	// nothing running
	// then the priority 3 workflows will run
	// then the 2
	// then the 1
	// then sleepy (priority = 0)
	s.Given().
		Workflow(`@testdata/priority-1-workflow.yaml`).
		When().
		SubmitWorkflow().
		Given().
		Workflow(`@testdata/priority-2-workflow.yaml`).
		When().
		SubmitWorkflow().
		Given().
		Workflow(`@testdata/priority-3-workflow.yaml`).
		When().
		SubmitWorkflow().
		Given().
		Workflow(`@testdata/priority-1-workflow.yaml`).
		When().
		SubmitWorkflow().
		Given().
		Workflow(`@testdata/priority-2-workflow.yaml`).
		When().
		SubmitWorkflow().
		Given().
		Workflow(`@testdata/priority-3-workflow.yaml`).
		When().
		SubmitWorkflow().
		Given().
		// this will run and finish after all others
		Workflow(`@testdata/sleep-3s.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(1 * time.Minute).
		Then().
		ExpectWorkflows(func(t *testing.T, wfs wfv1.Workflows) {
			if assert.Len(t, wfs, 7) {
				assert.Empty(t, wfs[0].Spec.Priority) // sleep
				assert.Equal(t, 2, int(*wfs[1].Spec.Priority))
				assert.Equal(t, 1, int(*wfs[2].Spec.Priority))
				assert.Equal(t, 3, int(*wfs[3].Spec.Priority))
				assert.Equal(t, 3, int(*wfs[4].Spec.Priority))
				assert.Equal(t, 2, int(*wfs[5].Spec.Priority))
				assert.Equal(t, 1, int(*wfs[6].Spec.Priority))
			}
		})
}

func TestPrioritySuite(t *testing.T) {
	suite.Run(t, new(WorkflowPrioritySuite))
}
