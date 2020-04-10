package e2e

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type CLIWithServerSuite struct {
	CLISuite
}

func (s *CLIWithServerSuite) BeforeTest(suiteName, testName string) {
	s.CLISuite.BeforeTest(suiteName, testName)
	token, err := s.GetServiceAccountToken()
	s.CheckError(err)
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
	_ = os.Setenv("ARGO_TOKEN", token)
}

func (s *CLIWithServerSuite) AfterTest(suiteName, testName string) {
	s.CLISuite.AfterTest(suiteName, testName)
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_TOKEN")
}

func (s *CLISuite) TestAuthToken() {
	s.Given().RunCli([]string{"auth", "token"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		var authString, token string
		token = s.GetBasicAuthToken()
		if token == "" {
			token, err = s.GetServiceAccountToken()
			assert.NoError(t, err)
			authString = "Bearer " + token
		} else {
			authString = "Basic " + token
		}
		assert.Equal(t, authString, strings.TrimSpace(output))
	})
}

func (s *CLIWithServerSuite) TestArchive() {
	if !s.Persistence.IsEnabled() {
		s.T().SkipNow()
	}
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.Run("List", func() {
		s.Given().
			RunCli([]string{"archive", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "NAMESPACE NAME")
					assert.Contains(t, output, "argo basic")
				}
			})
	})
	s.Run("Get", func() {
		s.Given().
			RunCli([]string{"archive", "get", string(uid)}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Name:")
					assert.Contains(t, output, "Namespace:")
					assert.Contains(t, output, "ServiceAccount:")
					assert.Contains(t, output, "Status:")
					assert.Contains(t, output, "Created:")
					assert.Contains(t, output, "Started:")
					assert.Contains(t, output, "Finished:")
					assert.Contains(t, output, "Duration:")
				}
			})
	})
	s.Run("Delete", func() {
		s.Given().
			RunCli([]string{"archive", "delete", string(uid)}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Archived workflow")
					assert.Contains(t, output, "deleted")
				}
			})
	})
}

func TestCLIWithServerSuite(t *testing.T) {
	suite.Run(t, new(CLIWithServerSuite))
}
