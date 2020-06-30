// +build e2e

package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

func (s *CLISuite) testNeedsOffloading() {
	skip := s.Persistence.IsEnabled() && os.Getenv("ARGO_SERVER") == ""
	if skip {
		s.T().Skip("test needs offloading, but not Argo Server available")
	}
}

func (s *CLISuite) TestCompletion() {
	s.Given().RunCli([]string{"completion", "bash"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "bash completion for argo")
	})
}

func (s *CLISuite) TestVersion() {
	// check we can run this without error
	s.Given().
		RunCli([]string{"version"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
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
	s.Given().
		Workflow(`@smoke/basic.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Nodes.FindByDisplayName("basic") != nil
		}, "pod running", 10*time.Second)

	s.Run("FollowWorkflowLogs", func() {
		s.Given().
			RunCli([]string{"logs", "basic", "--follow"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("FollowPodLogs", func() {
		s.Given().
			RunCli([]string{"logs", "basic", "basic", "--follow"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("ContainerLogs", func() {
		s.Given().
			RunCli([]string{"logs", "basic", "basic", "-c", "wait"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Executor")
				}
			})
	})
	s.Run("Since", func() {
		s.Given().
			RunCli([]string{"logs", "basic", "--since=1s"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("SinceTime", func() {
		s.Given().
			RunCli([]string{"logs", "basic", "--since-time=" + time.Now().Format(time.RFC3339)}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("TailLines", func() {
		s.Given().
			RunCli([]string{"logs", "basic", "--tail=0"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("CompletedWorkflow", func() {
		s.Given().
			WorkflowName("basic").
			When().
			WaitForWorkflow(10*time.Second).
			Then().
			RunCli([]string{"logs", "basic", "--tail=10"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Argo!")
				}
			})
	})
}

// this test probably should be in the ArgoServerSuite, but it's just much easier to write the test
// for the CLI
func (s *CLISuite) TestLogProblems() {
	s.Given().
		Workflow(`@testdata/log-problems.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		Then().
		// logs should come in order
		RunCli([]string{"logs", "log-problems", "--follow"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				lines := strings.Split(output, "\n")
				if assert.Len(t, lines, 6) {
					assert.Contains(t, lines[0], "one")
					assert.Contains(t, lines[1], "two")
					assert.Contains(t, lines[2], "three")
					assert.Contains(t, lines[3], "four")
					assert.Contains(t, lines[4], "five")
				}
			}
		}).
		When().
		// Next check that all log entries and received and in the correct order.
		WaitForWorkflow(30*time.Second).
		Then().
		RunCli([]string{"logs", "log-problems"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				lines := strings.Split(output, "\n")
				if assert.Len(t, lines, 6) {
					assert.Contains(t, lines[0], "one")
					assert.Contains(t, lines[1], "two")
					assert.Contains(t, lines[2], "three")
					assert.Contains(t, lines[3], "four")
					assert.Contains(t, lines[4], "five")
				}
			}
		})
}

func (s *CLISuite) TestRoot() {
	s.Run("Submit", func() {
		s.Given().RunCli([]string{"submit", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("List", func() {
		s.testNeedsOffloading()
		for i := 0; i < 3; i++ {
			s.Given().
				Workflow("@smoke/basic-generate-name.yaml").
				When().
				SubmitWorkflow().
				WaitForWorkflow(20 * time.Second)
		}
		s.Given().RunCli([]string{"list", "--chunk-size", "1"}, func(t *testing.T, output string, err error) {
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
		s.testNeedsOffloading()
		s.Given().RunCli([]string{"get", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
	})

	var createdWorkflowName string
	s.Run("From", func() {
		s.Given().CronWorkflow("@cron/basic.yaml").
			When().
			CreateCronWorkflow().
			RunCli([]string{"submit", "--from", "cronwf/test-cron-wf-basic", "-l", "argo-e2e=true"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "Name:                test-cron-wf-basic-")
				r := regexp.MustCompile(`Name:\s+?(test-cron-wf-basic-[a-z0-9]+)`)
				res := r.FindStringSubmatch(output)
				if len(res) != 2 {
					assert.Fail(t, "Internal test error, please report a bug")
				}
				createdWorkflowName = res[1]
			}).
			WaitForWorkflowName(createdWorkflowName, 30*time.Second).
			Then().
			ExpectWorkflowName(createdWorkflowName, func(t *testing.T, metadata *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			})
	})
}

func (s *CLISuite) TestWorkflowSuspendResume() {
	s.testNeedsOffloading()
	s.Given().
		Workflow("@testdata/sleep-3s.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(10*time.Second).
		RunCli([]string{"suspend", "sleep-3s"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow sleep-3s suspended")
			}
		}).
		RunCli([]string{"resume", "sleep-3s"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow sleep-3s resumed")
			}
		}).
		WaitForWorkflow(20 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *CLISuite) TestNodeSuspendResume() {
	s.testNeedsOffloading()
	s.Given().
		Workflow("@testdata/node-suspend.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}, "suspended node", 30*time.Second).
		RunCli([]string{"resume", "node-suspend", "--node-field-selector", "inputs.parameters.tag.value=suspend1-tag1"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow node-suspend resumed")
			}
		}).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}, "suspended node", 10*time.Second).
		RunCli([]string{"stop", "node-suspend", "--node-field-selector", "inputs.parameters.tag.value=suspend2-tag1", "--message", "because"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow node-suspend stopped")
			}
		}).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Phase == wfv1.NodeFailed
		}, "suspended node", 10*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
			if assert.Equal(t, wfv1.NodeFailed, status.Phase) {
				r := regexp.MustCompile(`child '(node-suspend-[0-9]+)' failed`)
				res := r.FindStringSubmatch(status.Message)
				assert.Equal(t, len(res), 2)
				assert.Equal(t, status.Nodes[res[1]].Message, "because")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteByName() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		RunCli([]string{"delete", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Workflow 'basic' deleted")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteDryRun() {
	s.Given().
		When().
		RunCli([]string{"delete", "--dry-run", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Workflow 'basic' deleted (dry-run)")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteNothing() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		RunCli([]string{"delete"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.NotContains(t, output, "deleted")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteNotFound() {
	s.Given().
		When().
		RunCli([]string{"delete", "not-found"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Workflow 'not-found' not found")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteAll() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		Given().
		RunCli([]string{"delete", "--all", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Workflow 'basic' deleted")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteCompleted() {
	s.Given().
		Workflow("@testdata/sleep-3s.yaml").
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
		WaitForWorkflow(30*time.Second).
		Given().
		RunCli([]string{"delete", "--completed", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "deleted")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteOlder() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		Given().
		RunCli([]string{"delete", "--older", "1d", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				// nothing over a day should be deleted
				assert.NotContains(t, output, "deleted")
			}
		}).
		RunCli([]string{"delete", "--older", "0s", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "deleted")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteByPrefix() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		Given().
		RunCli([]string{"delete", "--prefix", "missing-prefix", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				// nothing should be deleted
				assert.NotContains(t, output, "deleted")
			}
		}).
		RunCli([]string{"delete", "--prefix", "basic", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "deleted")
			}
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
	s.Run("LintFileEmptyParamDAG", func() {
		s.Given().RunCli([]string{"lint", "expectedfailures/empty-parameter-dag.yaml"}, func(t *testing.T, output string, err error) {
			if assert.EqualError(t, err, "exit status 1") {
				assert.Contains(t, output, "templates.abc.tasks.a templates.whalesay inputs.parameters.message was not supplied")
			}
		})
	})
	s.Run("LintFileEmptyParamSteps", func() {
		s.Given().RunCli([]string{"lint", "expectedfailures/empty-parameter-steps.yaml"}, func(t *testing.T, output string, err error) {
			if assert.EqualError(t, err, "exit status 1") {
				assert.Contains(t, output, "templates.abc.steps[0].a templates.whalesay inputs.parameters.message was not supplied")
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
		s.CheckError(err)
		defer func() { _ = os.RemoveAll(tmp) }()
		// Read all content of src to data
		data, err := ioutil.ReadFile("smoke/basic.yaml")
		s.CheckError(err)
		// Write data to dst
		err = ioutil.WriteFile(filepath.Join(tmp, "my-workflow.yaml"), data, 0644)
		s.CheckError(err)
		s.Given().
			RunCli([]string{"lint", tmp}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "my-workflow.yaml is valid")
				}
			})
	})

	s.Run("Different Kind", func() {
		s.Given().
			RunCli([]string{"lint", "testdata/workflow-template-nested-template.yaml"}, func(t *testing.T, output string, err error) {
				if assert.Error(t, err) {
					assert.Contains(t, output, "WorkflowTemplate 'workflow-template-nested-template' is not of kind Workflow. Ignoring...")
					assert.Contains(t, output, "Error in file testdata/workflow-template-nested-template.yaml: there was nothing to validate")
				}
			})
	})
	s.Run("Valid", func() {
		s.Given().
			RunCli([]string{"lint", "testdata/exit-1.yaml"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "exit-1.yaml is valid")
				}
			})
	})
	s.Run("Invalid", func() {
		s.Given().
			RunCli([]string{"lint", "expectedfailures/empty-parameter-dag.yaml"}, func(t *testing.T, output string, err error) {
				if assert.Error(t, err) {
					assert.Contains(t, output, "Error in file expectedfailures/empty-parameter-dag.yaml:")
				}
			})
	})
	// Not all files in this directory are Workflows, expect failure
	s.Run("NotAllWorkflows", func() {
		s.Given().
			RunCli([]string{"lint", "testdata"}, func(t *testing.T, output string, err error) {
				if assert.Error(t, err) {
					assert.Contains(t, output, "WorkflowTemplate 'workflow-template-nested-template' is not of kind Workflow. Ignoring...")
					assert.Contains(t, output, "Error in file testdata/workflow-template-nested-template.yaml: there was nothing to validate")
				}
			})
	})

	// All files in this directory are Workflows, expect success
	s.Run("AllWorkflows", func() {
		s.Given().
			RunCli([]string{"lint", "stress"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
			})
	})
}

func (s *CLISuite) TestWorkflowRetry() {
	s.testNeedsOffloading()
	var retryTime corev1.Time

	s.Given().
		Workflow("@testdata/retry-test.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}, "suspended node", 30*time.Second).
		RunCli([]string{"terminate", "retry-test"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow retry-test terminated")
			}
		}).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			retryTime = wf.Status.FinishedAt
			return wf.Status.Phase == wfv1.NodeFailed
		}, "terminated", 20*time.Second).
		RunCli([]string{"retry", "retry-test", "--restart-successful", "--node-field-selector", "templateName==steps-inner"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
			}
		}).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}, "suspended node", 20*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
			outerStepsPodNode := status.Nodes.FindByDisplayName("steps-outer-step1")
			innerStepsPodNode := status.Nodes.FindByDisplayName("steps-inner-step1")

			assert.True(t, outerStepsPodNode.FinishedAt.Before(&retryTime))
			assert.True(t, retryTime.Before(&innerStepsPodNode.FinishedAt))
		})
}

func (s *CLISuite) TestWorkflowTerminate() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Given().
		RunCli([]string{"terminate", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow basic terminated")
			}
		})
}

func (s *CLISuite) TestWorkflowWait() {
	s.testNeedsOffloading()
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Given().
		RunCli([]string{"wait", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "basic Succeeded")
			}
		})
}

func (s *CLISuite) TestWorkflowWatch() {
	s.testNeedsOffloading()
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Given().
		RunCli([]string{"watch", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
			}
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
			if assert.EqualError(t, err, "exit status 1") {
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
	s.Run("Submittable-Template", func() {
		s.testNeedsOffloading()
		s.Given().RunCli([]string{"submit", "--from", "workflowtemplate/workflow-template-whalesay-template", "-l", "argo-e2e=true"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
		var workflowName string
		s.Given().RunCli([]string{"list"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				r := regexp.MustCompile(`\s+?(workflow-template-whalesay-template-[a-z0-9]+)`)
				res := r.FindStringSubmatch(output)
				if len(res) != 2 {
					assert.Fail(t, "Internal test error, please report a bug")
				}
				workflowName = res[1]
			}
		})
		s.Given().
			WorkflowName(workflowName).
			When().
			WaitForWorkflow(30*time.Second).
			RunCli([]string{"get", workflowName}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, workflowName)
					assert.Contains(t, output, "Succeeded")
				}
			})
	})
	s.Run("Delete", func() {
		s.Given().RunCli([]string{"template", "delete", "workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		})
	})
}

func (s *CLISuite) TestWorkflowResubmit() {
	s.Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		Given().
		RunCli([]string{"resubmit", "--memoized", "exit-1"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		})
}

func (s *CLISuite) TestCron() {
	s.Run("Lint", func() {
		s.Given().RunCli([]string{"cron", "lint", "cron/basic.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "cron/basic.yaml is valid")
				assert.Contains(t, output, "Cron workflow manifests validated")
			}
		})
	})
	s.Run("Different Kind", func() {
		s.Given().
			RunCli([]string{"cron", "lint", "testdata/workflow-template-nested-template.yaml"}, func(t *testing.T, output string, err error) {
				if assert.Error(t, err) {
					assert.Contains(t, output, "WorkflowTemplate 'workflow-template-nested-template' is not of kind CronWorkflow. Ignoring...")
					assert.Contains(t, output, "Error in file testdata/workflow-template-nested-template.yaml: there was nothing to validate")
				}
			})
	})
	// Not all files in this directory are CronWorkflows, expect failure
	s.Run("NotAllWorkflows", func() {
		s.Given().
			RunCli([]string{"cron", "lint", "testdata"}, func(t *testing.T, output string, err error) {
				if assert.Error(t, err) {
					assert.Contains(t, output, "WorkflowTemplate 'workflow-template-nested-template' is not of kind CronWorkflow. Ignoring...")
					assert.Contains(t, output, "Error in file testdata/workflow-template-nested-template.yaml: there was nothing to validate")
				}
			})
	})

	// All files in this directory are CronWorkflows, expect success
	s.Run("AllCron", func() {
		s.Given().
			RunCli([]string{"cron", "lint", "cron"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
			})
	})

	s.Run("Create", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/basic.yaml"}, func(t *testing.T, output string, err error) {
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

	s.Run("Create Schedule Override", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/basic.yaml", "--schedule", "1 2 3 * *"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Schedule:                      1 2 3 * *")
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
	s.Run("Suspend", func() {
		s.Given().RunCli([]string{"cron", "suspend", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "CronWorkflow 'test-cron-wf-basic' suspended")
			}
		})
	})
	s.Run("Resume", func() {
		s.Given().RunCli([]string{"cron", "resume", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "CronWorkflow 'test-cron-wf-basic' resumed")
			}
		})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"cron", "get", "not-found"}, func(t *testing.T, output string, err error) {
			if assert.EqualError(t, err, "exit status 1") {
				assert.Contains(t, output, `\"not-found\" not found`)

			}
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
}

func (s *CLISuite) TestClusterTemplateCommands() {
	s.Run("Create", func() {
		s.Given().
			RunCli([]string{"cluster-template", "create", "smoke/cluster-workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
				}
			})
	})
	s.Run("Get", func() {
		s.Given().
			RunCli([]string{"cluster-template", "get", "cluster-workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
				}
			})
	})
	s.Run("list", func() {
		s.Given().
			RunCli([]string{"cluster-template", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
				}
			})
	})
	s.Run("Delete", func() {
		s.Given().
			RunCli([]string{"cluster-template", "delete", "cluster-workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
				}
			})
	})
}

func (s *CLISuite) TestWorkflowTemplateRefSubmit() {
	s.Run("CreateWFT", func() {
		s.Given().RunCli([]string{"template", "create", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("CreateWF", func() {
		s.Given().RunCli([]string{"submit", "testdata/workflow-template-ref.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("CreateCWFT", func() {
		s.Given().RunCli([]string{"cluster-template", "create", "smoke/cluster-workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
	s.Run("CreateWFWithCWFTRef", func() {
		s.Given().RunCli([]string{"submit", "testdata/cluster-workflow-template-ref.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			}
		})
	})
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
