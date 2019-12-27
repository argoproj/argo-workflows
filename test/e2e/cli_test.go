package e2e

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type CLISuite struct {
	fixtures.E2ESuite
}

func (s *CLISuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
}

func (s *CLISuite) AfterTest(suiteName, testName string) {
	s.E2ESuite.AfterTest(suiteName, testName)
	_ = os.Unsetenv("ARGO_SERVER")
}

func argo(args ...string) (string, error) {
	output, err := exec.Command("../../dist/argo", args...).CombinedOutput()
	return string(output), err
}

func (s *CLISuite) TestCompletion() {
	output, err := argo("completion", "bash")
	s.Assert().NoError(err)
	s.Assert().Contains(output, "bash completion for argo")
}

func (s *CLISuite) TestCore() {
	s.T().Run("Submit", func(t *testing.T) {
		output, err := argo("submit", "smoke/basic.yaml", "--wait")
		assert.NoError(t, err)
		assert.Contains(t, output, "Succeeded")
	})
	s.T().Run("Get", func(t *testing.T) {
		output, err := argo("get", "basic")
		assert.NoError(t, err)
		assert.Contains(t, output, "Succeeded")
	})
}

func (s *CLISuite) TestHistory() {
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.T().Run("List", func(t *testing.T) {
		output, err := argo("history", "list")
		assert.NoError(t, err)
		assert.Contains(t, output, "NAMESPACE NAME")
		assert.Contains(t, output, "argo basic")
	})
	s.T().Run("Get", func(t *testing.T) {
		output, err := argo("history", "get", fixtures.Namespace, string(uid))
		assert.NoError(t, err)
		assert.Contains(t, output, "Succeeded")
	})
	s.T().Run("Resubmit", func(t *testing.T) {
		output, err := argo("history", "resubmit", fixtures.Namespace, string(uid))
		assert.NoError(t, err)
		assert.Contains(t, output, "Workflow")
		assert.Contains(t, output, "basic")
		assert.Contains(t, output, "resubmitted")
	})
	s.T().Run("Delete", func(t *testing.T) {
		output, err := argo("history", "delete", fixtures.Namespace, string(uid))
		assert.NoError(t, err)
		assert.Contains(t, output, "Workflow")
		assert.Contains(t, output, "deleted")
	})
}

func TestCliSuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
