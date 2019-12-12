package e2e

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type AutoSuite struct {
	fixtures.E2ESuite
}

func (s AutoSuite) SetupSuite() {
	s.T().SkipNow()
}

func (s AutoSuite) TestFunctional() {
	files, err := ioutil.ReadDir("functional")
	if assert.NoError(s.T(), err) {
		for _, file := range files {
			s.Run(file.Name(), func() {
				s.Given().
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

func (s AutoSuite) TestExpectedFailures() {
	files, err := ioutil.ReadDir("expectedfailures")
	if assert.NoError(s.T(), err) {
		for _, file := range files {
			s.Run(file.Name(), func() {
				s.Given().
					Workflow("@expectedfailures/" + file.Name()).
					When().
					SubmitWorkflow().
					WaitForWorkflow().
					Then().
					Expect(func(t *testing.T, wf *v1alpha1.WorkflowStatus) {
						assert.Equal(t, v1alpha1.NodeFailed, wf.Phase)
					})
			})
		}
	}
}
func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(AutoSuite))
}
