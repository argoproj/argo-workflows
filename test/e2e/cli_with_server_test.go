// +build e2e

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
	_ = os.Setenv("ARGO_SECURE", "true")
	_ = os.Setenv("ARGO_INSECURE_SKIP_VERIFY", "true")
	_ = os.Setenv("ARGO_TOKEN", token)
}

func (s *CLIWithServerSuite) AfterTest(suiteName, testName string) {
	s.CLISuite.AfterTest(suiteName, testName)
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_SECURE")
	_ = os.Unsetenv("ARGO_INSECURE_SKIP_VERIFY")
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

func (s *CLIWithServerSuite) TestVersion() {
	s.Run("Default", func() {
		s.Given().
			RunCli([]string{"version"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					lines := strings.Split(output, "\n")
					if assert.Len(t, lines, 17) {
						assert.Contains(t, lines[0], "argo:")
						assert.Contains(t, lines[1], "BuildDate:")
						assert.Contains(t, lines[2], "GitCommit:")
						assert.Contains(t, lines[3], "GitTreeState:")
						assert.Contains(t, lines[4], "GitTag:")
						assert.Contains(t, lines[5], "GoVersion:")
						assert.Contains(t, lines[6], "Compiler:")
						assert.Contains(t, lines[7], "Platform:")
						assert.Contains(t, lines[8], "argo-server:")
						assert.Contains(t, lines[9], "BuildDate:")
						assert.Contains(t, lines[10], "GitCommit:")
						assert.Contains(t, lines[11], "GitTreeState:")
						assert.Contains(t, lines[12], "GitTag:")
						assert.Contains(t, lines[13], "GoVersion:")
						assert.Contains(t, lines[14], "Compiler:")
						assert.Contains(t, lines[15], "Platform:")
					}
					// these are the defaults - we should never see these
					assert.NotContains(t, output, "argo: v0.0.0+unknown")
					assert.NotContains(t, output, "  BuildDate: 1970-01-01T00:00:00Z")
				}
			})
	})
	s.Run("Short", func() {
		s.Given().
			RunCli([]string{"version", "--short"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					lines := strings.Split(output, "\n")
					if assert.Len(t, lines, 3) {
						assert.Contains(t, lines[0], "argo:")
						assert.Contains(t, lines[1], "argo-server:")
					}
				}
			})
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
			RunCli([]string{"archive", "list", "--chunk-size", "1"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					lines := strings.Split(output, "\n")
					assert.Contains(t, lines[0], "NAMESPACE")
					assert.Contains(t, lines[0], "NAME")
					assert.Contains(t, lines[0], "STATUS")
					assert.Contains(t, lines[1], "argo")
					assert.Contains(t, lines[1], "basic")
					assert.Contains(t, lines[1], "Succeeded")
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
