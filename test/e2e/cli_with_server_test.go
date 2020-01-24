package e2e

import (
	"os"
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
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("ARGO_TOKEN", token)
}

func (s *CLIWithServerSuite) AfterTest(suiteName, testName string) {
	s.CLISuite.AfterTest(suiteName, testName)
	_ = os.Unsetenv("ARGO_TOKEN")
}

func (s *CLIWithServerSuite) TestToken() {
	s.Given().
		RunCli([]string{"token"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			token, err := s.GetServiceAccountToken()
			assert.NoError(t, err)
			assert.Equal(t, token, output)
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
		WaitForWorkflow(30*time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		}).
		RunCli([]string{"archive", "list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAMESPACE NAME")
				assert.Contains(t, output, "argo basic")
			}
		}).
		RunCli([]string{"archive", "get", string(uid)}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Succeeded")
			}
		}).
		RunCli([]string{"archive", "delete", string(uid)}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Archived workflow")
				assert.Contains(t, output, "deleted")
			}
		})
}

func TestCLIWithServerSuite(t *testing.T) {
	suite.Run(t, new(CLIWithServerSuite))
}
