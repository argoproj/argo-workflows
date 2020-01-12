package e2e

import (
	"os"
	"os/exec"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type CLISuite struct {
	fixtures.E2ESuite
	lastOutput string
	lastErr    error
}

func (s *CLISuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
}

func (s *CLISuite) AfterTest(suiteName, testName string) {
	s.E2ESuite.AfterTest(suiteName, testName)
	_ = os.Unsetenv("ARGO_SERVER")
	if s.T().Failed() {
		log.WithFields(log.Fields{"lastOutput": s.lastOutput, "lastError": s.lastErr}).Info("Last CLI output and error")
	}
}

func (s *CLISuite) argo(args ...string) (string, error) {
	args = append([]string{"-n", fixtures.Namespace}, args...)
	output, err := exec.Command("../../dist/argo", args...).CombinedOutput()
	s.lastOutput = string(output)
	s.lastErr = err
	return s.lastOutput, s.lastErr
}

func (s *CLISuite) TestCompletion() {
	output, err := s.argo("completion", "bash")
	s.Assert().NoError(err)
	s.Assert().Contains(output, "bash completion for argo")
}

func (s *CLISuite) TestToken() {
	output, err := s.argo("token")
	s.Assert().NoError(err)
	s.Assert().NotEmpty(output)
}

func (s *CLISuite) TestRoot() {
	s.Run("Submit", func(t *testing.T) {
		// TODO - with --wait we get an error - need to investigate
		output, err := s.argo("submit", "smoke/basic.yaml")
		assert.NoError(t, err)
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "ServiceAccount:")
		assert.Contains(t, output, "Status:")
		assert.Contains(t, output, "Created:")
	})
	s.Run("List", func(t *testing.T) {
		output, err := s.argo("list")
		assert.NoError(t, err)
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "STATUS")
		assert.Contains(t, output, "AGE")
		assert.Contains(t, output, "DURATION")
		assert.Contains(t, output, "PRIORITY")
	})
	s.Run("Get", func(t *testing.T) {
		output, err := s.argo("get", "basic")
		assert.NoError(t, err)
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "ServiceAccount:")
		assert.Contains(t, output, "Status:")
		assert.Contains(t, output, "Created:")
		assert.Contains(t, output, "Started:")
		assert.Contains(t, output, "Duration:")
	})
}

func (s *CLISuite) TestCron() {

	s.Run("Create", func(t *testing.T) {
		output, err := s.argo("cron", "create", "cron/testdata/basic.yaml")
		assert.NoError(t, err)
		assert.Contains(t, output, "Name:")
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "Created:")
		assert.Contains(t, output, "Schedule:")
		assert.Contains(t, output, "Suspended:")
		assert.Contains(t, output, "StartingDeadlineSeconds:")
		assert.Contains(t, output, "ConcurrencyPolicy:")
	})

	s.Run("List", func(t *testing.T) {
		output, err := s.argo("cron", "list")
		assert.NoError(t, err)
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "AGE")
		assert.Contains(t, output, "LAST RUN")
		assert.Contains(t, output, "SCHEDULE")
		assert.Contains(t, output, "SUSPENDED")
	})

	s.Run("Get", func(t *testing.T) {
		output, err := s.argo("cron", "get", "not-found")
		assert.EqualError(t, err, "exit status 1")
		assert.Contains(t, output, `"not-found" not found`)

		output, err = s.argo("cron", "get", "test-cron-wf-basic")
		if assert.NoError(t, err) {
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
			assert.Contains(t, output, "Schedule:")
			assert.Contains(t, output, "Suspended:")
			assert.Contains(t, output, "StartingDeadlineSeconds:")
			assert.Contains(t, output, "ConcurrencyPolicy:")
		}
	})

	s.Run("Delete", func(t *testing.T) {
		_, err := s.argo("cron", "delete", "test-cron-wf-basic")
		assert.NoError(t, err)
	})
}

func (s *CLISuite) TestArchive() {
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
	s.Run("List", func(t *testing.T) {
		output, err := s.argo("archive", "list")
		assert.NoError(t, err)
		assert.Contains(t, output, "NAMESPACE NAME")
		assert.Contains(t, output, "argo basic")
	})
	s.Run("Get", func(t *testing.T) {
		output, err := s.argo("archive", "get", string(uid))
		assert.NoError(t, err)
		assert.Contains(t, output, "Succeeded")
	})
	s.Run("Delete", func(t *testing.T) {
		output, err := s.argo("archive", "delete", string(uid))
		assert.NoError(t, err)
		assert.Contains(t, output, "Archived workflow")
		assert.Contains(t, output, "deleted")
	})
}

func TestCliSuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
