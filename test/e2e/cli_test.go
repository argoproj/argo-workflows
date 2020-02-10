package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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
func (s *CLISuite) TestSubmitDryRun() {
	s.Given().
		RunCli([]string{"submit", "smoke/basic.yaml", "--dry-run", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "name: basic")
				// dry-run should never get a UID
				assert.NotContains(t, output, "uid:")
			}
		})
}

func (s *CLISuite) TestSubmitServerDryRun() {
	s.Given().
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
	s.Run("ListWithBadToken", func() {
		s.Given().RunCli([]string{"list", "--user", "fake_token_user", "--token", "badtoken"}, func(t *testing.T, output string, err error) {
			assert.Error(t, err)
		})
	})

	var goodToken string
	s.Run("GetSAToken", func() {
		token, err := s.GetServiceAccountToken()
		assert.NoError(s.T(), err)
		goodToken = token
	})
	s.Run("ListWithGoodToken", func() {
		s.Given().RunCli([]string{"list", "--user", "fake_token_user", "--token", goodToken}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "NAME")
			assert.Contains(t, output, "STATUS")
		})
	})
}

func (s *CLISuite) TestLogs() {
	// we use our own YAML as we do not want to use PodGC for this test

	s.Run("SetUp", func() {
		s.Given().
			Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wf
  labels:
    argo-e2e: true
spec:
  entrypoint: my-template
  templates:
    - name: my-template
      container:
        image: cowsay:v1
        command: [cowsay, ":) Hello Logs!"]
        imagePullPolicy: IfNotPresent`).
			When().
			SubmitWorkflow().
			WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
				return wf.Status.Nodes.FindByDisplayName("my-wf") != nil
			}, "pod running", 10*time.Second)
	})
	s.Run("FollowPodLogs", func() {
		s.Given().
			RunCli([]string{"logs", "my-wf", "my-wf", "--follow"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Logs!")
				}
			})
	})
	s.Run("ContainerLogs", func() {
		s.Given().
			RunCli([]string{"logs", "my-wf", "my-wf", "-c", "wait"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Executor")
				}
			})
	})
	s.Run("Since", func() {
		s.Given().
			RunCli([]string{"logs", "my-wf", "--since=0s"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Logs!")
				}
			})
	})
	s.Run("SinceTime", func() {
		s.Given().
			RunCli([]string{"logs", "my-wf", "--since-time=" + time.Now().Format(time.RFC3339)}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Logs!")
				}
			})
	})
	s.Run("TailLines", func() {
		s.Given().
			RunCli([]string{"logs", "my-wf", "--tail=0"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Logs!")
				}
			})
	})
	s.Run("CompletedWorkflow", func() {
		s.Given().
			WorkflowName("my-wf").
			When().
			WaitForWorkflow(10*time.Second).
			Then().
			RunCli([]string{"logs", "my-wf", "my-wf", "--tail=10"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Logs!")
				}
			})
	})
}

func (s *CLISuite) TestRoot() {
	s.Run("Submit", func() {
		s.Given().RunCli([]string{"submit", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("List", func() {
		s.Given().RunCli([]string{"list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAME")
				assert.Contains(t, output, "STATUS")
				assert.Contains(t, output, "AGE")
				assert.Contains(t, output, "DURATION")
				assert.Contains(t, output, "PRIORITY")
			}
		})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"get", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
	})

	var createdWorkflowName string
	s.Run("From", func() {
		s.Given().CronWorkflow("@testdata/basic.yaml").
			When().
			CreateCronWorkflow().
			RunCli([]string{"submit", "--from", "cronwf/test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "Name:                test-cron-wf-basic-")
				r := regexp.MustCompile(`Name:\s+?(test-cron-wf-basic-[a-z0-9]+)`)
				res := r.FindStringSubmatch(output)
				if len(res) != 2 {
					assert.Fail(t, "Internal test error, please report a bug")
				}
				createdWorkflowName = res[1]
			}).
			WaitForWorkflowName(createdWorkflowName, 15*time.Second).
			Then().
			ExpectWorkflowName(createdWorkflowName, func(t *testing.T, metadata *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			})
	})
}

func (s *CLISuite) TestWorkflowDelete() {
	s.Run("DeleteByName", func() {
		s.Given().
			Workflow("@smoke/basic.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(15*time.Second).
			Given().
			RunCli([]string{"delete", "basic"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Workflow 'basic' deleted")
				}
			})
	})
	s.Run("DeleteAll", func() {
		s.Given().
			Workflow("@smoke/basic.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(15*time.Second).
			Given().
			RunCli([]string{"delete", "--all", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Workflow 'basic' deleted")
				}
			})
	})
	s.Run("DeleteCompleted", func() {
		s.Given().
			Workflow("@smoke/basic.yaml").
			When().
			SubmitWorkflow().
			Given().
			RunCli([]string{"delete", "--completed", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					// nothing should be deleted yet
					assert.NotContains(t, output, "deleted")
				}
			}).
			When().
			WaitForWorkflow(15*time.Second).
			Given().
			RunCli([]string{"delete", "--completed", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Workflow 'basic' deleted")
				}
			})
	})
	s.Run("DeleteOlder", func() {
		s.Given().
			Workflow("@smoke/basic.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(15*time.Second).
			Given().
			RunCli([]string{"delete", "--older", "1d", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					// nothing over a day should be deleted
					assert.NotContains(t, output, "deleted")
				}
			})
	})
}
func (s *CLISuite) TestWorkflowLint() {
	s.Run("LintFile", func() {
		s.Given().RunCli([]string{"lint", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "smoke/basic.yaml is valid")
			}
		})
	})
	s.Run("LintFileWithTemplate", func() {
		s.Given().
			WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
			When().
			CreateWorkflowTemplates().
			Given().
			RunCli([]string{"lint", "smoke/hello-world-workflow-tmpl.yaml"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "smoke/hello-world-workflow-tmpl.yaml is valid")
				}
			})
	})
	s.Run("LintDir", func() {
		tmp, err := ioutil.TempDir("", "")
		if err != nil {
			s.T().Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		// Read all content of src to data
		data, err := ioutil.ReadFile("smoke/basic.yaml")
		if err != nil {
			s.T().Fatal(err)
		}
		// Write data to dst
		err = ioutil.WriteFile(filepath.Join(tmp, "my-workflow.yaml"), data, 0644)
		if err != nil {
			s.T().Fatal(err)
		}
		s.Given().
			RunCli([]string{"lint", tmp}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "my-workflow.yaml is valid")
				}
			})
	})
}

func (s *CLISuite) TestTemplate() {
	s.Run("Lint", func() {
		s.Given().RunCli([]string{"template", "lint", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "validated")
			}
		})
	})
	s.Run("Create", func() {
		s.Given().RunCli([]string{"template", "create", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("List", func() {
		s.Given().RunCli([]string{"template", "list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAME")
			}
		})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"template", "get", "not-found"}, func(t *testing.T, output string, err error) {
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
	s.Run("Delete", func() {
		s.Given().RunCli([]string{"template", "delete", "workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		})
	})
}

func (s *CLISuite) TestCron() {
	s.Run("Create", func() {
		s.Given().RunCli([]string{"cron", "create", "testdata/basic.yaml"}, func(t *testing.T, output string, err error) {
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
	s.Run("List", func() {
		s.Given().RunCli([]string{"cron", "list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "NAME")
				assert.Contains(t, output, "AGE")
				assert.Contains(t, output, "LAST RUN")
				assert.Contains(t, output, "SCHEDULE")
				assert.Contains(t, output, "SUSPENDED")
			}
		})
	})
	s.Run("Get", func() {
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
	})
	s.Run("Delete", func() {
		s.Given().RunCli([]string{"cron", "delete", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		})
	})
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
