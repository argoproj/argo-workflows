package e2e

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

func (suite FunctionalSuite) TestFunctional() {
	t := suite.T()
	files, err := ioutil.ReadDir("functional")
	if assert.NoError(t, err) {
		for _, file := range files {
			suite.Run(file.Name(), func() {
				suite.Given().
					Workflow("@functional/" + file.Name()).
					When().
					SubmitWorkflow().
					WaitForWorkflow().
					Then().
					Expect(func(t *testing.T, wf *v1alpha1.WorkflowStatus) {
						assert.Equal(t, v1alpha1.NodeSucceeded, wf.Phase)
					})
			})
		}
	}
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
