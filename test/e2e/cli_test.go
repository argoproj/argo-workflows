// +build cli

package e2e

import (
	"fmt"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type CLISuite struct {
	fixtures.E2ESuite
}

var kubeConfig = os.Getenv("KUBECONFIG")

func (s *CLISuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	mode := os.Getenv("E2E_MODE")
	println("mode:", mode)
	token, err := s.GetServiceAccountToken()
	s.CheckError(err)
	switch mode {
	default:
		_ = os.Unsetenv("ARGO_INSTANCEID")
		_ = os.Unsetenv("ARGO_SERVER")
		_ = os.Unsetenv("ARGO_BASE_HREF")
		_ = os.Unsetenv("ARGO_HTTP1")
		_ = os.Unsetenv("ARGO_TOKEN")
		_ = os.Unsetenv("ARGO_NAMESPACE")
		_ = os.Setenv("KUBECONFIG", kubeConfig)
	case "GRPC":
		_ = os.Unsetenv("ARGO_INSTANCEID")
		_ = os.Setenv("ARGO_SERVER", "localhost:2746")
		_ = os.Unsetenv("ARGO_BASE_HREF")
		_ = os.Unsetenv("ARGO_SECURE")
		_ = os.Unsetenv("ARGO_INSECURE_SKIP_VERIFY")
		_ = os.Unsetenv("ARGO_HTTP1")
		_ = os.Setenv("ARGO_TOKEN", "Bearer "+token)
		_ = os.Setenv("ARGO_NAMESPACE", "argo")
		_ = os.Setenv("KUBECONFIG", "/dev/null")
	case "HTTP1":
		_ = os.Unsetenv("ARGO_INSTANCEID")
		_ = os.Setenv("ARGO_SERVER", "localhost:2746")
		_ = os.Unsetenv("ARGO_BASE_HREF")
		_ = os.Unsetenv("ARGO_SECURE")
		_ = os.Unsetenv("ARGO_INSECURE_SKIP_VERIFY")
		_ = os.Setenv("ARGO_HTTP1", "true")
		_ = os.Setenv("ARGO_TOKEN", "Bearer "+token)
		_ = os.Setenv("ARGO_NAMESPACE", "argo")
		_ = os.Setenv("KUBECONFIG", "/dev/null")
	}
}

func (s *CLISuite) AfterTest(suiteName, testName string) {
	_ = os.Setenv("KUBECONFIG", kubeConfig)
}

func (s *CLISuite) needsServer() {
	if os.Getenv("ARGO_SERVER") == "" {
		s.T().Skip("test needs server")
	}
}
func (s *CLISuite) skipIfServer() {
	if os.Getenv("ARGO_SERVER") != "" {
		s.T().Skip("test must not run with server")
	}
}

func (s *CLISuite) skipIfHTTP() {
	if os.Getenv("ARGO_HTTP1") != "" {
		s.T().Skip("test must not run with HTTP")
	}
}

func (s *CLISuite) NeedsOffloading() {
	s.E2ESuite.NeedsOffloading()
	s.needsServer()
}

func (s *CLISuite) TestCompletion() {
	s.Given().RunCli([]string{"completion", "bash"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.Contains(t, output, "bash completion for argo")
	})
}

func (s *CLISuite) TestLogLevels() {
	s.Run("Verbose", func() {
		s.Given().
			RunCli([]string{"-v", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "CLI version", "comment version header")
				}
			})
	})
	s.Run("LogLevel", func() {
		s.Given().
			RunCli([]string{"--loglevel=debug", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "CLI version", "comment version header")
				}
			})
	})
}

func (s *CLISuite) TestGLogLevels() {
	s.skipIfServer()
	s.Run("Verbose", func() {
		s.Given().
			RunCli([]string{"-v", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Config loaded from file", "glog output")
				}
			})
	})
	s.Run("LogLevel", func() {
		s.Given().
			RunCli([]string{"--loglevel=debug", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, "Config loaded from file", "glog output")
				}
			})
	})
	s.Run("GLogLevel", func() {
		s.Given().
			RunCli([]string{"--gloglevel=6", "list"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Config loaded from file", "glog output")
				}
			})
	})
}

func (s *CLISuite) TestVersion() {
	// check we can run this without error
	s.Run("NoError", func() {
		s.Given().
			RunCli([]string{"version"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
			})

	})
	s.Run("Default", func() {
		s.needsServer()
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
		s.needsServer()
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

func (s *CLISuite) TestSubmitDryRun() {
	s.Given().
		RunCli([]string{"submit", "smoke/basic.yaml", "--dry-run", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "generateName: basic")
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
	s.skipIfServer()
	s.NeedsCI()
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
		})
	})
}

func (s *CLISuite) TestLogs() {
	var name string
	s.Given().
		Workflow(`@smoke/basic.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart, "to start").
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Nodes.FindByDisplayName(wf.Name) != nil
		}), "pod running", 10*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		})

	s.Run("FollowWorkflowLogs", func() {
		s.Given().
			RunCli([]string{"logs", name, "--follow"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("FollowPodLogs", func() {
		s.Given().
			RunCli([]string{"logs", name, name, "--follow"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("ContainerLogs", func() {
		s.Given().
			RunCli([]string{"logs", name, name, "-c", "wait"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.Contains(t, output, "Executor")
				}
			})
	})
	s.Run("Since", func() {
		s.Given().
			RunCli([]string{"logs", name, "--since=1s"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("SinceTime", func() {
		s.skipIfHTTP() // this test errors with `field type *v1.Time is not supported in query parameters`
		s.Given().
			RunCli([]string{"logs", name, "--since-time=" + time.Now().Format(time.RFC3339)}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("TailLines", func() {
		s.Given().
			RunCli([]string{"logs", name, "--tail=0"}, func(t *testing.T, output string, err error) {
				if assert.NoError(t, err) {
					assert.NotContains(t, output, ":) Hello Argo!")
				}
			})
	})
	s.Run("CompletedWorkflow", func() {
		s.Given().
			When().
			WaitForWorkflow().
			Then().
			RunCli([]string{"logs", name, "--tail=10"}, func(t *testing.T, output string, err error) {
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
		WaitForWorkflow(fixtures.ToStart, "to start").
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
		WaitForWorkflow().
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
		s.Given().RunCli([]string{"submit", "testdata/basic-workflow.yaml"}, func(t *testing.T, output string, err error) {
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
		s.NeedsOffloading()
		for i := 0; i < 3; i++ {
			s.Given().
				Workflow("@smoke/basic-generate-name.yaml").
				When().
				SubmitWorkflow().
				WaitForWorkflow()
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
		s.NeedsOffloading()
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
			WaitForWorkflow(createdWorkflowName).
			Then().
			ExpectWorkflowName(createdWorkflowName, func(t *testing.T, metadata *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			})
	})
}

func (s *CLISuite) TestWorkflowSuspendResume() {
	s.NeedsOffloading()
	s.Given().
		Workflow("@testdata/sleep-3s.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart, "to start").
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
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *CLISuite) TestNodeSuspendResume() {
	s.NeedsOffloading()
	s.Given().
		Workflow("@testdata/node-suspend.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}), "suspended node").
		RunCli([]string{"resume", "node-suspend", "--node-field-selector", "inputs.parameters.tag.value=suspend1-tag1"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow node-suspend resumed")
			}
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}), "suspended node").
		RunCli([]string{"stop", "node-suspend", "--node-field-selector", "inputs.parameters.tag.value=suspend2-tag1", "--message", "because"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow node-suspend stopped")
			}
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Phase == wfv1.NodeFailed
		}), "suspended node").
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
	var name string
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		}).
		RunCli([]string{"delete", name}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "Workflow 'basic-.*' deleted", output)
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteByFieldSelector() {
	var name string
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		}).
		RunCli([]string{"delete", "--field-selector", fmt.Sprintf("metadata.name=%s", name)}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "Workflow 'basic-.*' deleted", output)
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
		RunCli([]string{"delete"}, func(t *testing.T, output string, err error) {
			if assert.EqualError(t, err, "exit status 1") {
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
		Given().
		RunCli([]string{"delete", "--all", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "Workflow 'basic-.*' deleted", output)
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
		WaitForWorkflow().
		Given().
		RunCli([]string{"delete", "--completed", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "deleted")
			}
		})
}

func (s *CLISuite) TestWorkflowDeleteResubmitted() {
	s.Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"resubmit", "--memoized", "exit-1"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
			}
		}).
		When().
		Given().
		RunCli([]string{"delete", "--resubmitted", "-l", "argo-e2e"}, func(t *testing.T, output string, err error) {
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
		WaitForWorkflow().
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
}

func (s *CLISuite) TestWorkflowRetry() {
	s.NeedsOffloading()
	var retryTime corev1.Time

	s.Given().
		Workflow("@testdata/retry-test.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart, "to start").
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}), "suspended node").
		RunCli([]string{"terminate", "retry-test"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "workflow retry-test terminated")
			}
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			retryTime = wf.Status.FinishedAt
			return wf.Status.Phase == wfv1.NodeFailed
		}), "is terminated", 20*time.Second).
		Wait(3*time.Second).
		RunCli([]string{"retry", "retry-test", "--restart-successful", "--node-field-selector", "templateName==steps-inner"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err, output) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
			}
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.AnyActiveSuspendNode()
		}), "suspended node").
		Then().
		ExpectWorkflow(func(t *testing.T, _ *corev1.ObjectMeta, status *wfv1.WorkflowStatus) {
			outerStepsPodNode := status.Nodes.FindByDisplayName("steps-outer-step1")
			innerStepsPodNode := status.Nodes.FindByDisplayName("steps-inner-step1")

			assert.True(t, outerStepsPodNode.FinishedAt.Before(&retryTime))
			assert.True(t, retryTime.Before(&innerStepsPodNode.FinishedAt))
		})
}

func (s *CLISuite) TestWorkflowTerminate() {
	var name string
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		}).
		RunCli([]string{"terminate", name}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "workflow basic-.* terminated", output)
			}
		})
}

func (s *CLISuite) TestWorkflowTerminateDryRun() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"terminate", "--dry-run", "basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "workflow basic terminated \\(dry-run\\)", output)
			}
		})
}

func (s *CLISuite) TestWorkflowTerminateBySelector() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"terminate", "--selector", "argo-e2e=true"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "workflow basic terminated", output)
			}
		})
}

func (s *CLISuite) TestWorkflowTerminateByFieldSelector() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"terminate", "--field-selector", "metadata.name=basic"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "workflow basic terminated", output)
			}
		})
}

func (s *CLISuite) TestWorkflowWait() {
	s.needsServer()
	var name string
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		}).
		RunCli([]string{"wait", name}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Regexp(t, "basic-.* Succeeded", output)
			}
		})
}

func (s *CLISuite) TestWorkflowWatch() {
	s.needsServer()
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		RunCli([]string{"watch", "@latest"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "Name: ")
			}
		})
}

func (s *CLISuite) TestTemplate() {
	s.Run("LintWithoutArgs", func() {
		s.Given().RunCli([]string{"template", "lint"}, func(t *testing.T, output string, err error) {
			if assert.Error(t, err) {
				assert.Contains(t, output, "Usage:")
			}
		})
	})

	s.Run("Lint", func() {
		s.Given().RunCli([]string{"template", "lint", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "validated")
			}
		})
	})
	s.Run("DirLintWithInvalidWFT", func() {
		s.Given().RunCli([]string{"template", "lint", "testdata/workflow-templates"}, func(t *testing.T, output string, err error) {
			assert.Error(t, err)
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
		s.NeedsOffloading()
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
			WaitForWorkflow().
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
		WaitForWorkflow().
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

	s.Run("Create Parameter Override", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/param.yaml", "-p", "message=\"bar test passed\""}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "bar test passed")
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

func (s *CLISuite) TestWorkflowLevelSemaphore() {
	semaphoreData := map[string]string{
		"workflow": "1",
	}
	s.NeedsOffloading()
	s.Given().
		Workflow("@testdata/semaphore-wf-level.yaml").
		When().
		CreateConfigMap("my-config", semaphoreData).
		RunCli([]string{"submit", "testdata/semaphore-wf-level-1.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "semaphore-wf-level-1")
			}
		}).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Phase == ""
		}), "Workflow is waiting for lock").
		WaitForWorkflow().
		DeleteConfigMap("my-config").
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *CLISuite) TestTemplateLevelSemaphore() {
	semaphoreData := map[string]string{
		"template": "1",
	}

	s.NeedsOffloading()
	s.Given().
		Workflow("@testdata/semaphore-tmpl-level.yaml").
		When().
		CreateConfigMap("my-config", semaphoreData).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Phase == wfv1.NodeRunning
		}), "waiting for Workflow to run", 10*time.Second).
		RunCli([]string{"get", "semaphore-tmpl-level"}, func(t *testing.T, output string, err error) {
			assert.Contains(t, output, "Waiting for")
		}).
		WaitForWorkflow()
}

func (s *CLISuite) TestRetryOmit() {
	s.NeedsOffloading()
	s.Given().
		Workflow("@testdata/retry-omit.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
				return node.Phase == wfv1.NodeOmitted
			})
		}), "any node omitted").
		WaitForWorkflow(10*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			node := status.Nodes.FindByDisplayName("should-not-execute")
			if assert.NotNil(t, node) {
				assert.Equal(t, wfv1.NodeOmitted, node.Phase)
			}
		}).
		RunCli([]string{"retry", "dag-diamond-8q7vp"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "Status:              Running")
		}).When().
		WaitForWorkflow()
}

func (s *CLISuite) TestSynchronizationWfLevelMutex() {
	s.NeedsOffloading()
	s.Given().
		Workflow("@functional/synchronization-mutex-wf-level.yaml").
		When().
		RunCli([]string{"submit", "functional/synchronization-mutex-wf-level-1.yaml"}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err) {
				assert.Contains(t, output, "synchronization-wf-level-mutex")
			}
		}).
		SubmitWorkflow().
		Wait(1*time.Second).
		RunCli([]string{"get", "synchronization-wf-level-mutex"}, func(t *testing.T, output string, err error) {
			assert.Contains(t, output, "Pending")
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *CLISuite) TestTemplateLevelMutex() {
	s.NeedsOffloading()
	s.Given().
		Workflow("@functional/synchronization-mutex-tmpl-level.yaml").
		When().
		SubmitWorkflow().
		Wait(3*time.Second).
		RunCli([]string{"get", "synchronization-tmpl-level-mutex"}, func(t *testing.T, output string, err error) {
			assert.Contains(t, output, "Waiting for")
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *CLISuite) TestResourceTemplateStopAndTerminate() {
	s.NeedsOffloading()
	s.Run("ResourceTemplateStop", func() {
		s.Given().
			WorkflowName("resource-tmpl-wf").
			When().
			RunCli([]string{"submit", "functional/resource-template.yaml"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Pending")
			}).
			RunCli([]string{"get", "resource-tmpl-wf"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Running")
			}).
			RunCli([]string{"stop", "resource-tmpl-wf"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "workflow resource-tmpl-wf stopped")
			}).
			WaitForWorkflow().
			RunCli([]string{"get", "resource-tmpl-wf"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Stopped with strategy 'Stop'")
			}).
			RunCli([]string{"delete", "resource-tmpl-wf"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "deleted")
			})

	})
	s.Run("ResourceTemplateTerminate", func() {
		s.Given().
			WorkflowName("resource-tmpl-wf-1").
			When().
			RunCli([]string{"submit", "functional/resource-template.yaml", "--name", "resource-tmpl-wf-1"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Pending")
			}).
			WaitForWorkflow(fixtures.ToBeRunning).
			RunCli([]string{"get", "resource-tmpl-wf-1"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Running")
			}).
			RunCli([]string{"terminate", "resource-tmpl-wf-1"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "workflow resource-tmpl-wf-1 terminated")
			}).
			WaitForWorkflow().
			RunCli([]string{"get", "resource-tmpl-wf-1"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Stopped with strategy 'Terminate'")
			})

	})
}

func (s *CLISuite) TestMetaDataNamespace() {
	s.needsServer()
	s.Given().
		Exec("../../dist/argo", []string{"cron", "create", "testdata/wf-default-ns.yaml"}, func(t *testing.T, output string, err error) {
			if assert.Error(t, err) {
				assert.Contains(t, output, "PermissionDenied")
				assert.Contains(t, output, `in the namespace "default"`)
			}
		}).
		Exec("../../dist/argo", []string{"cron", "get", "test-cron-wf-basic", "-n", "default"}, func(t *testing.T, output string, err error) {
			if assert.Error(t, err) {
				assert.Contains(t, output, "PermissionDenied")
				assert.Contains(t, output, `in the namespace \"default\"`)
			}
		}).
		Exec("../../dist/argo", []string{"cron", "delete", "test-cron-wf-basic", "-n", "default"}, func(t *testing.T, output string, err error) {
			if assert.Error(t, err) {
				assert.Contains(t, output, "PermissionDenied")
				assert.Contains(t, output, `in the namespace \"default\"`)
			}
		})
}

func (s *CLISuite) TestAuthToken() {
	s.NeedsOffloading()
	s.Given().RunCli([]string{"auth", "token"}, func(t *testing.T, output string, err error) {
		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})
}

func (s *CLISuite) TestArchive() {
	s.NeedsOffloading()
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived, "to be archived").
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

func (s *CLISuite) TestArgoSetOutputs() {
	s.NeedsOffloading()
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
		WaitForWorkflow(fixtures.ToStart, "to start").
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

func TestCLISuite(t *testing.T) {
	_ = os.Setenv("E2E_MODE", "KUBE")
	suite.Run(t, new(CLISuite))
}

func TestCLIWithServerOverGRPCSuite(t *testing.T) {
	_ = os.Setenv("E2E_MODE", "GRPC")
	suite.Run(t, new(CLISuite))
}

func TestCLIWithServerOverHTTP1Suite(t *testing.T) {
	_ = os.Setenv("E2E_MODE", "HTTP1")
	suite.Run(t, new(CLISuite))
}
