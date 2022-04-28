//go:build executor
// +build executor

package e2e

import (
	"testing"

	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type HttpArtifactsSuite struct {
	fixtures.E2ESuite
}

func (s *HttpArtifactsSuite) TestInputArtifactHttp() {
	s.Given().
		Workflow("@testdata/input-artifact-http.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *HttpArtifactsSuite) TestOutputArtifactHttp() {
	s.Given().
		Workflow("@testdata/output-artifact-http.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *HttpArtifactsSuite) TestBasicAuthArtifactHttp() {
	s.Given().
		Workflow("@testdata/basic-auth-artifact-http.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *HttpArtifactsSuite) TestOAuthArtifactHttp() {
	s.Given().
		Workflow("@testdata/oauth-artifact-http.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *HttpArtifactsSuite) TestClientCertAuthArtifactHttp() {
	s.Given().
		Workflow("@testdata/clientcert-auth-artifact-http.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestHttpArtifactsSuite(t *testing.T) {
	suite.Run(t, new(HttpArtifactsSuite))
}
