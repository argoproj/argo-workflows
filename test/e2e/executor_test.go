// +build executor

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ExecutorSuite struct {
	fixtures.E2ESuite
}

// executors can should captured the result, but only when `tasks.a.outputs.result` appears in the template
func (s *ExecutorSuite) TestIncludeScriptResult() {
	s.Given().
		Workflow("@testdata/output-result-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, "to be succeeded").
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			if a := status.Nodes.FindByDisplayName("a"); assert.NotNil(t, a) {
				assert.Equal(t, pointer.StringPtr("0"), a.Outputs.ExitCode)
				assert.Equal(t, pointer.StringPtr("foo"), a.Outputs.Result, "only set because 'b' needs it")
			}
			if b := status.Nodes.FindByDisplayName("b"); assert.NotNil(t, b) {
				assert.Equal(t, pointer.StringPtr("0"), b.Outputs.ExitCode)
				assert.Equal(t, wfv1.AnyStringPtr("foo"), b.Inputs.Parameters[0].Value, "set from the result of 'a'")
				assert.Nil(t, b.Outputs.Result, "<nil> because no tasks needs it")
			}
		})
}

func TestExecutorSuite(t *testing.T) {
	suite.Run(t, new(ExecutorSuite))
}
