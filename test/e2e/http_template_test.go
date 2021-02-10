// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type HTTPTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *HTTPTemplateSuite) TestHTTPGet() {
	s.Given().
		Workflow("@testdata/http-template/get-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func TestHTTPTemplateSuite(t *testing.T) {
	suite.Run(t, new(HTTPTemplateSuite))
}
