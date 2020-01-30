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
	s.Given(s.T()).RunCli([]string{"completion", "bash"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "bash completion for argo")
	})
}
func (s *CLISuite) TestSubmitDryRun() {
	s.Given(s.T()).
		RunCli([]string{"submit", "smoke/basic.yaml", "--dry-run", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "name: basic")
				// dry-run should never get a UID
				assert.NotContains(t, output, "uid:")
			}
		})
}

func (s *CLISuite) TestSubmitServerDryRun() {
	s.Given(s.T()).
		RunCli([]string{"submit", "smoke/basic.yaml", "--server-dry-run", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "name: basic")
				// server-dry-run should get a UID
				assert.Contains(t, output, "uid:")
			}
		})
}

func (s *CLISuite) TestTokenArg() {
	if os.Getenv("CI") != "true" {
		s.T().SkipNow()
	}
	s.Given().RunCli([]string{"list", "--user", "fake_token_user", "--token", "badtoken"}, func(t *testing.T, output string, err error) {
		assert.Error(t, err)
	})

	var goodToken string
	s.Run("GetSAToken", func(t *testing.T) {
		token, err := s.GetServiceAccountToken()
		assert.NoError(t, err)
		goodToken = token
	})

	s.Given().RunCli([]string{"list", "--user", "fake_token_user", "--token", goodToken}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "STATUS")
	})
}

func (s *CLISuite) TestRoot() {
	s.Run("Submit", func(t *testing.T) {
		s.Given(t).RunCli([]string{"submit", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("List", func(t *testing.T) {
		s.Given(t).RunCli([]string{"list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAME")
				assert.Contains(t, output, "STATUS")
				assert.Contains(t, output, "AGE")
				assert.Contains(t, output, "DURATION")
				assert.Contains(t, output, "PRIORITY")
			}
		})
	})
	s.Run("Get", func(t *testing.T) {
		s.Given(t).RunCli([]string{"get", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("Delete", func(t *testing.T) {
		s.Given(t).RunCli([]string{"delete", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "deleted")
			}
		})
	})
}

func (s *CLISuite) TestTemplate() {
	s.Run("Lint", func(t *testing.T) {
		s.Given(t).RunCli([]string{"template", "lint", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "validated")
			}
		})

	})
	s.Run("Create", func(t *testing.T) {
		s.Given(t).RunCli([]string{"template", "create", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("List", func(t *testing.T) {
		s.Given(t).RunCli([]string{"template", "list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAME")
			}
		})
	})
	s.Run("Get", func(t *testing.T) {
		s.Given(t).RunCli([]string{"template", "get", "not-found"}, func(t *testing.T, output string, err error) {
			if assert.Error(t, err, "exit status 1") {
				assert.Contains(t, output, `"not-found" not found`)

			}
		}).RunCli([]string{"template", "get", "workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("Delete", func(t *testing.T) {
		s.Given(t).RunCli([]string{"template", "delete", "workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		})
	})
}

func (s *CLISuite) TestCron() {
	s.Run("Create", func(t *testing.T) {
		s.Given(t).RunCli([]string{"cron", "create", "testdata/basic.yaml"}, func(t *testing.T, output string, err error) {
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
	})
	s.Run("List", func(t *testing.T) {
		s.Given(t).RunCli([]string{"cron", "list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAME")
				assert.Contains(t, output, "AGE")
				assert.Contains(t, output, "LAST RUN")
				assert.Contains(t, output, "SCHEDULE")
				assert.Contains(t, output, "SUSPENDED")
			}
		})
	})
	s.Run("Get", func(t *testing.T) {
		s.Given(t).RunCli([]string{"cron", "get", "not-found"}, func(t *testing.T, output string, err error) {
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
	})
	s.Run("Delete", func(t *testing.T) {
		s.Given(t).RunCli([]string{"cron", "delete", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		})
	})
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
