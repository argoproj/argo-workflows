//go:build functional

package e2e

// import (
// 	"testing"

// 	"github.com/stretchr/testify/suite"

// 	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
// )

// type HttpArtifactsSuite struct {
// 	fixtures.E2ESuite
// }

// func (s *HttpArtifactsSuite) TestInputArtifactHttp() {
// 	s.Given().
// 		Workflow("@testdata/http/input-artifact-http.yaml").
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeSucceeded)
// }

// func (s *HttpArtifactsSuite) TestOutputArtifactHttp() {
// 	s.Given().
// 		Workflow("@testdata/http/output-artifact-http.yaml").
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeSucceeded)
// }

// func (s *HttpArtifactsSuite) TestBasicAuthArtifactHttp() {
// 	s.Given().
// 		Workflow("@testdata/http/basic-auth-artifact-http.yaml").
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeSucceeded)
// }

// func (s *HttpArtifactsSuite) TestOAuthArtifactHttp() {
// 	s.Given().
// 		Workflow("@testdata/http/oauth-artifact-http.yaml").
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeSucceeded)
// }

// func (s *HttpArtifactsSuite) TestClientCertAuthArtifactHttp() {
// 	s.Given().
// 		Workflow("@testdata/http/clientcert-auth-artifact-http.yaml").
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeSucceeded)
// }

// func (s *HttpArtifactsSuite) TestArtifactoryArtifacts() {
// 	s.Given().
// 		Workflow("@testdata/http/artifactory-artifact.yaml").
// 		When().
// 		SubmitWorkflow().
// 		WaitForWorkflow(fixtures.ToBeSucceeded)
// }

// func TestHttpArtifactsSuite(t *testing.T) {
// 	suite.Run(t, new(HttpArtifactsSuite))
// }
