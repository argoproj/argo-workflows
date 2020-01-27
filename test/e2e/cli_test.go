package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo/test/e2e/fixtures"
)

type CLISuite struct {
	fixtures.E2ESuite
}

func (s *CLISuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_TOKEN")
}

func (s *CLISuite) TestCompletion() {
	s.Given().RunCli([]string{"completion", "bash"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "bash completion for argo")
	})
}

func (s *CLISuite) TestRoot() {
	s.Given().RunCli([]string{"submit", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "ServiceAccount:")
		assert.Contains(t, output, "Status:")
		assert.Contains(t, output, "Created:")
	})
	s.Given().RunCli([]string{"list"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "STATUS")
		assert.Contains(t, output, "AGE")
		assert.Contains(t, output, "DURATION")
		assert.Contains(t, output, "PRIORITY")
	})
	s.Given().RunCli([]string{"get", "basic"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "ServiceAccount:")
		assert.Contains(t, output, "Status:")
		assert.Contains(t, output, "Created:")
	})
	s.Given().RunCli([]string{"delete", "basic"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "deleted")
	})
}

func (s *CLISuite) TestTemplate() {

	s.Given().RunCli([]string{"template", "lint", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "validated")
	})

	s.Given().RunCli([]string{"template", "create", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "Name:")
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "Created:")
	})

	s.Given().RunCli([]string{"template", "list"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "NAME")
	})

	s.Given().RunCli([]string{"template", "get", "not-found"}, func(t *testing.T, output string, err error) {
		assert.Error(t, err, "exit status 1")
		assert.Contains(t, output, `"not-found" not found`)
	}).RunCli([]string{"template", "get", "workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
		if assert.NoError(t, err) {
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		}
	})

	s.Given().RunCli([]string{"template", "delete", "workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
	})
}

func (s *CLISuite) TestCron() {

	s.Given().RunCli([]string{"cron", "create", "testdata/basic.yaml"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "Name:")
		assert.Contains(t, output, "Namespace:")
		assert.Contains(t, output, "Created:")
		assert.Contains(t, output, "Schedule:")
		assert.Contains(t, output, "Suspended:")
		assert.Contains(t, output, "StartingDeadlineSeconds:")
		assert.Contains(t, output, "ConcurrencyPolicy:")
	})

	s.Given().RunCli([]string{"cron", "list"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "AGE")
		assert.Contains(t, output, "LAST RUN")
		assert.Contains(t, output, "SCHEDULE")
		assert.Contains(t, output, "SUSPENDED")
	})

	s.Given().RunCli([]string{"cron", "get", "not-found"}, func(t *testing.T, output string, err error) {
		assert.Error(t, err, "exit status 1")
		assert.Contains(t, output, `"not-found" not found`)
	}).RunCli([]string{"cron", "get", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
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

	s.Given().RunCli([]string{"cron", "delete", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
	})
}

func TestCliSuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
