// +build e2e

package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type CLIWithServerSuite struct {
	CLISuite
}

func (s *CLIWithServerSuite) BeforeTest(suiteName, testName string) {
	s.CLISuite.BeforeTest(suiteName, testName)
	token, err := s.GetServiceAccountToken()
	s.CheckError(err)
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
	_ = os.Setenv("ARGO_SECURE", "false")
	_ = os.Setenv("ARGO_TOKEN", "Bearer "+token)
	// we should not need this to run any tests
	_ = os.Setenv("KUBECONFIG", "/dev/null")
}

func (s *CLIWithServerSuite) AfterTest(suiteName, testName string) {
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_SECURE")
	_ = os.Unsetenv("ARGO_TOKEN")
	s.CLISuite.AfterTest(suiteName, testName)
}

func (s *CLISuite) TestAuthToken() {
	s.Given().RunCli([]string{"auth", "token"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})
}

func (s *CLIWithServerSuite) TestTokenArg() {
	// we mark this test as skipped because it does not make any sense when only using server
	s.T().SkipNow()
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
		WaitForWorkflow().
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

func (s *CLIWithServerSuite) TestArgoSetOutputs() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-template
  labels:
    argo-e2e: true
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: approve
        template: approve
      - name: approve-no-vars
        template: approve-no-vars
    - - name: release
        template: whalesay
        arguments:
          parameters:
            - name: message
              value: "{{steps.approve.outputs.parameters.message}}"

  - name: approve
    suspend: {}
    outputs:
      parameters:
        - name: message
          valueFrom:
            supplied: {}

  - name: approve-no-vars
    suspend: {}

  - name: whalesay
    inputs:
      parameters:
        - name: message
    container:
      image: argoproj/argosay:v2
      args: ["echo", "{{inputs.parameters.message}}"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart).
		RunCli([]string{"resume", "suspend-template"}, func(t *testing.T, output string, err error) {
			assert.Error(t, err)
			assert.Contains(t, output, "has not been set and does not have a default value")
		}).
		RunCli([]string{"node", "set", "suspend-template", "--output-parameter", "message=\"Hello, World!\"", "--node-field-selector", "displayName=approve"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "workflow values set")
		}).
		RunCli([]string{"node", "set", "suspend-template", "--output-parameter", "message=\"Hello, World!\"", "--node-field-selector", "displayName=approve"}, func(t *testing.T, output string, err error) {
			// Cannot double-set the same parameter
			assert.Error(t, err)
			assert.Contains(t, output, "it was already set")
		}).
		RunCli([]string{"node", "set", "suspend-template", "--output-parameter", "message=\"Hello, World!\"", "--node-field-selector", "displayName=approve-no-vars"}, func(t *testing.T, output string, err error) {
			assert.Error(t, err)
			assert.Contains(t, output, "cannot set output parameters because node is not expecting any raw parameters")
		}).
		RunCli([]string{"node", "set", "suspend-template", "--message", "Test message", "--node-field-selector", "displayName=approve"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "workflow values set")
		}).
		RunCli([]string{"resume", "suspend-template"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "workflow suspend-template resumed")
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("release")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, "Hello, World!", nodeStatus.Inputs.Parameters[0].Value.String())
			}
			nodeStatus = status.Nodes.FindByDisplayName("approve")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, "Test message", nodeStatus.Message)
			}
		})
}

func TestCLIWithServerSuite(t *testing.T) {
	suite.Run(t, new(CLIWithServerSuite))
}
