//go:build cli

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

const (
	GRPC    = "GRPC"
	KUBE    = "KUBE"
	HTTP1   = "HTTP1"
	DEFAULT = HTTP1
	OFFLINE = "OFFLINE"
)

type CLISuite struct {
	fixtures.E2ESuite
}

var kubeConfig = os.Getenv("KUBECONFIG")

func (s *CLISuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	s.setMode(HTTP1)
}

func (s *CLISuite) setMode(mode string) {
	token, err := s.GetServiceAccountToken()
	s.CheckError(err)
	_ = os.Unsetenv("ARGO_INSTANCEID")
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
	_ = os.Unsetenv("ARGO_BASE_HREF")
	_ = os.Setenv("ARGO_SECURE", "false")
	_ = os.Unsetenv("ARGO_INSECURE_SKIP_VERIFY")
	_ = os.Setenv("ARGO_TOKEN", "Bearer "+token)
	_ = os.Setenv("ARGO_NAMESPACE", "argo")
	_ = os.Setenv("KUBECONFIG", "/dev/null")
	switch mode {
	case GRPC:
		_ = os.Unsetenv("ARGO_HTTP1")
	case HTTP1:
		_ = os.Setenv("ARGO_HTTP1", "true")
	case KUBE:
		_ = os.Unsetenv("ARGO_SERVER")
		_ = os.Unsetenv("ARGO_HTTP1")
		_ = os.Unsetenv("ARGO_TOKEN")
		_ = os.Unsetenv("ARGO_NAMESPACE")
		_ = os.Setenv("KUBECONFIG", kubeConfig)
	case OFFLINE:
		_ = os.Unsetenv("KUBECONFIG")
	default:
		panic(mode)
	}
}

func (s *CLISuite) AfterTest(suiteName, testName string) {
	_ = os.Setenv("KUBECONFIG", kubeConfig)
}

func (s *CLISuite) TestCompletion() {
	s.Given().RunCli([]string{"completion", "bash"}, func(t *testing.T, output string, err error) {
		require.NoError(t, err)
		assert.Contains(t, output, "bash completion for argo")
	})
}

func (s *CLISuite) TestLogLevels() {
	s.Run("Verbose", func() {
		s.Given().
			RunCli([]string{"-v", "list"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "CLI version", "comment version header")
			})
	})
	s.Run("LogLevel", func() {
		s.Given().
			RunCli([]string{"--loglevel=debug", "list"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "CLI version", "comment version header")
			})
	})
}

func (s *CLISuite) TestGLogLevels() {
	s.setMode(KUBE)
	expected := "Config loaded from file"
	s.Run("Verbose", func() {
		s.Given().
			RunCli([]string{"-v", "list"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, expected, "glog output")
			})
	})
	s.Run("LogLevel", func() {
		s.Given().
			RunCli([]string{"--loglevel=debug", "list"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, output, expected, "glog output")
			})
	})
	s.Run("GLogLevel", func() {
		s.Given().
			RunCli([]string{"--gloglevel=6", "list"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, expected, "glog output")
			})
	})
}

func (s *CLISuite) TestVersion() {
	// check we can run this without error
	s.Run("NoError", func() {
		s.Given().
			RunCli([]string{"version"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
			})
	})
	s.Run("Default", func() {
		s.Given().
			RunCli([]string{"version"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				lines := strings.Split(output, "\n")
				require.Len(t, lines, 17)
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
				// these are the defaults - we should never see these
				assert.NotContains(t, output, "argo: v0.0.0+unknown")
				assert.NotContains(t, output, "  BuildDate: 1970-01-01T00:00:00Z")
			})
	})
	s.Run("Short", func() {
		s.Given().
			RunCli([]string{"version", "--short"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				lines := strings.Split(output, "\n")
				require.Len(t, lines, 3)
				assert.Contains(t, lines[0], "argo:")
				assert.Contains(t, lines[1], "argo-server:")
			})
	})
}

func (s *CLISuite) TestGRPC() {
	s.setMode(GRPC)
	s.Given().
		RunCli([]string{"list"}, fixtures.NoError)
}

func (s *CLISuite) TestKUBE() {
	s.setMode(KUBE)
	s.Given().
		RunCli([]string{"list"}, fixtures.NoError)
}

func (s *CLISuite) TestSubmitDryRun() {
	s.Given().
		RunCli([]string{"submit", "smoke/basic.yaml", "--dry-run", "-o", "yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "generateName: basic")
			// dry-run should never get a UID
			assert.NotContains(t, output, "uid:")
		})
}

func (s *CLISuite) TestSubmitInvalidWf() {
	s.Given().
		RunCli([]string{"submit", "smoke/basic-invalid.yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.Error(t, err)
			assert.Contains(t, output, "yaml file at index 0 is not valid:")
		})
}

func (s *CLISuite) TestSubmitServerDryRun() {
	s.Given().
		RunCli([]string{"submit", "smoke/basic.yaml", "--server-dry-run", "-o", "yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "name: basic")
			// server-dry-run should get a UID
			assert.Contains(t, output, "uid:")
		})
}

func (s *CLISuite) TestSubmitWorkflowTemplateDryRun() {
	s.Given().
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		When().
		CreateWorkflowTemplates().
		RunCli([]string{"submit", "--dry-run", "--from", "workflowtemplate/workflow-template-whalesay-template", "-o", "yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "generateName: workflow-template-whalesay-template-")
			// dry-run should never get a UID
			assert.NotContains(t, output, "uid:")
		}).
		Then().
		ExpectWorkflowList(metav1.ListOptions{LabelSelector: common.LabelKeyWorkflowTemplate + "=workflow-template-whalesay-template"}, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Empty(t, wfList.Items)
		})
}

func (s *CLISuite) TestSubmitWorkflowTemplateServerDryRun() {
	s.Given().
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		When().
		CreateWorkflowTemplates().
		RunCli([]string{"submit", "--server-dry-run", "--from", "workflowtemplate/workflow-template-whalesay-template", "-o", "yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "generateName: workflow-template-whalesay-template-")
			// server-dry-run should get a UID
			assert.Contains(t, output, "uid:")
		}).
		Then().
		ExpectWorkflowList(metav1.ListOptions{LabelSelector: common.LabelKeyWorkflowTemplate + "=workflow-template-whalesay-template"}, func(t *testing.T, wfList *wfv1.WorkflowList) {
			assert.Empty(t, wfList.Items)
		})
}

func (s *CLISuite) TestTokenArg() {
	if os.Getenv("CI") != "true" {
		s.T().Skip("we only set-up the KUBECONFIG on CI")
	}
	s.setMode(KUBE)
	s.Run("ListWithBadToken", func() {
		s.Given().RunCli([]string{"list", "--user", "fake_token_user", "--token", "badtoken"}, func(t *testing.T, output string, err error) {
			require.Error(t, err)
		})
	})

	var goodToken string
	s.Run("GetSAToken", func() {
		token, err := s.GetServiceAccountToken()
		s.Require().NoError(err)
		goodToken = token
	})
	s.Run("ListWithGoodToken", func() {
		s.Given().RunCli([]string{"list", "--user", "fake_token_user", "--token", goodToken}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		})
	})
}

func (s *CLISuite) TestLogs() {
	var name string
	s.Given().
		Workflow(`@smoke/basic.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.Nodes.FindByDisplayName(wf.Name) != nil, "pod running"
		}), 40*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		})

	s.Run("FollowWorkflowLogs", func() {
		s.Given().
			RunCli([]string{"logs", name, "--follow"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, ":) Hello Argo!")
			})
	})
	s.Run("FollowPodLogs", func() {
		s.Given().
			RunCli([]string{"logs", name, name, "--follow"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, ":) Hello Argo!")
			})
	})
	s.Run("ContainerLogs", func() {
		s.Given().
			RunCli([]string{"logs", name, name, "-c", "wait"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Executor")
			})
	})
	s.Run("Since", func() {
		s.Given().
			RunCli([]string{"logs", name, "--since=1s"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, output, ":) Hello Argo!")
			})
	})
	s.Run("SinceTime", func() {
		s.setMode(KUBE)
		defer s.setMode(DEFAULT)
		s.Given().
			RunCli([]string{"logs", name, "--since-time=" + time.Now().Format(time.RFC3339)}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, output, ":) Hello Argo!")
			})
	})
	s.Run("TailLines", func() {
		s.Given().
			RunCli([]string{"logs", name, "--tail=0"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, output, ":) Hello Argo!")
			})
	})
	s.Run("Grep", func() {
		s.Given().
			RunCli([]string{"logs", name, "--grep=no"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, output, ":) Hello Argo!")
			})
	})
	s.Run("CompletedWorkflow", func() {
		s.Given().
			When().
			WaitForWorkflow().
			Then().
			RunCli([]string{"logs", name, "--tail=10"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, ":) Hello Argo!")
			})
	})
}

func toLines(x string) []string {
	var y []string
	for _, s := range strings.Split(x, "\n") {
		println("s=", s)
		if s != "" && !strings.Contains(s, "argo=true") {
			y = append(y, s)
		}
	}
	return y
}

// this test probably should be in the ArgoServerSuite, but it's just much easier to write the test
// for the CLI
func (s *CLISuite) TestLogProblems() {
	s.Given().
		Workflow(`@testdata/log-problems.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod).
		Then().
		// logs should come in order
		RunCli([]string{"logs", "@latest", "--follow"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			lines := toLines(output)
			require.Len(t, lines, 5)
			assert.Contains(t, lines[0], "one")
			assert.Contains(t, lines[1], "two")
			assert.Contains(t, lines[2], "three")
			assert.Contains(t, lines[3], "four")
			assert.Contains(t, lines[4], "five")
		}).
		When().
		// Next check that all log entries and received and in the correct order.
		WaitForWorkflow().
		Then().
		RunCli([]string{"logs", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			lines := toLines(output)
			require.Len(t, lines, 5)
			assert.Contains(t, lines[0], "one")
			assert.Contains(t, lines[1], "two")
			assert.Contains(t, lines[2], "three")
			assert.Contains(t, lines[3], "four")
			assert.Contains(t, lines[4], "five")
		})
}

func (s *CLISuite) TestParametersFile() {
	err := os.WriteFile("/tmp/parameters-file.yaml", []byte("message: hello"), os.ModePerm)
	s.Require().NoError(err)
	s.Given().
		RunCli([]string{"submit", "testdata/parameters-workflow.yaml", "-l", "workflows.argoproj.io/test=true", "--parameter-file=/tmp/parameters-file.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "message:           hello")
		})
}

func (s *CLISuite) TestRoot() {
	s.Run("Submit", func() {
		s.Given().RunCli([]string{"submit", "testdata/basic-workflow.yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("List", func() {
		s.Run("DefaultOutput", func() {
			s.Given().
				RunCli([]string{"list"}, func(t *testing.T, output string, err error) {
					require.NoError(t, err)
					assert.Contains(t, output, "NAME")
					assert.Contains(t, output, "STATUS")
					assert.Contains(t, output, "AGE")
					assert.Contains(t, output, "DURATION")
					assert.Contains(t, output, "PRIORITY")
				})
		})
		s.Run("NameOutput", func() {
			s.Given().
				RunCli([]string{"list", "-o", "name"}, func(t *testing.T, output string, err error) {
					require.NoError(t, err)
					assert.NotContains(t, output, "NAME")
				})
		})
		s.Run("WideOutput", func() {
			s.Given().
				RunCli([]string{"list", "-o", "wide"}, func(t *testing.T, output string, err error) {
					require.NoError(t, err)
					assert.Contains(t, output, "PARAMETERS")
				})
		})
		s.Run("JSONOutput", func() {
			s.Given().
				RunCli([]string{"list", "-o", "json"}, func(t *testing.T, output string, err error) {
					require.NoError(t, err)
					list := wfv1.Workflows{}
					require.NoError(t, json.Unmarshal([]byte(output), &list))
					assert.Len(t, list, 1)
				})
		})
		s.Run("YAMLOutput", func() {
			s.Given().
				RunCli([]string{"list", "-o", "yaml"}, func(t *testing.T, output string, err error) {
					require.NoError(t, err)
					list := wfv1.Workflows{}
					require.NoError(t, yaml.UnmarshalStrict([]byte(output), &list))
					assert.Len(t, list, 1)
				})
		})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"get", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("Delete", func() {
		s.Given().RunCli([]string{"delete", "@latest"}, fixtures.NoError)
	})
	s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7111")
	s.Run("From", func() {
		s.Given().
			CronWorkflow("@cron/basic.yaml").
			When().
			CreateCronWorkflow().
			RunCli([]string{"submit", "--from", "cronworkflow/test-cron-wf-basic", "--scheduled-time", "2006-01-02T15:04:05-07:00", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Name:                test-cron-wf-basic-")
			}).
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				assert.Equal(t, "2006-01-02T15:04:05-07:00", metadata.Annotations["workflows.argoproj.io/scheduled-time"])
			})
	})
}

func (s *CLISuite) TestSubmitClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		When().
		CreateClusterWorkflowTemplates().
		RunCli([]string{"submit", "--from", "clusterworkflowtemplate/cluster-workflow-template-whalesay-template", "--name", "my-wf", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *CLISuite) TestWorkflowSuspendResume() {
	s.Given().
		Workflow("@testdata/sleep-3s.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart).
		RunCli([]string{"suspend", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "workflow @latest suspended")
		}).
		RunCli([]string{"resume", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "workflow @latest resumed")
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *CLISuite) TestNodeSuspendResume() {
	s.Given().
		Workflow("@testdata/node-suspend.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.AnyActiveSuspendNode(), "suspended node"
		})).
		RunCli([]string{"resume", "@latest", "--node-field-selector", "inputs.parameters.tag.value=suspend1-tag1"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "workflow @latest resumed")
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.AnyActiveSuspendNode(), "suspended node"
		})).
		RunCli([]string{"stop", "@latest", "--node-field-selector", "inputs.parameters.tag.value=suspend2-tag1", "--message", "because"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow node-suspend-.* stopped", output)
		}).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Regexp(t, `child 'node-suspend-.*' failed`, status.Message)
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
			require.NoError(t, err)
			assert.Regexp(t, "Workflow 'basic-.*' deleted", output)
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
			require.NoError(t, err)
			assert.Regexp(t, "Workflow 'basic-.*' deleted", output)
		})
}

func (s *CLISuite) TestWorkflowDeleteDryRun() {
	s.Given().
		When().
		RunCli([]string{"delete", "--dry-run", "basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Workflow 'basic' deleted (dry-run)")
		})
}

func (s *CLISuite) TestWorkflowDeleteNothing() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"delete"}, func(t *testing.T, output string, err error) {
			require.EqualError(t, err, "exit status 1")
			assert.NotContains(t, output, "deleted")
		})
}

func (s *CLISuite) TestWorkflowDeleteNotFound() {
	s.Given().
		When().
		RunCli([]string{"delete", "not-found"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Workflow 'not-found' not found")
		})
}

func (s *CLISuite) TestWorkflowDeleteAll() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Given().
		RunCli([]string{"delete", "--all", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "Workflow 'basic-.*' deleted", output)
		})
}

func (s *CLISuite) TestWorkflowDeleteCompleted() {
	s.Given().
		Workflow("@testdata/sleep-3s.yaml").
		When().
		SubmitWorkflow().
		Given().
		RunCli([]string{"delete", "--completed", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			// nothing should be deleted yet
			assert.NotContains(t, output, "deleted")
		}).
		When().
		WaitForWorkflow().
		Given().
		RunCli([]string{"delete", "--completed", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "deleted")
		})
}

func (s *CLISuite) TestWorkflowDeleteResubmitted() {
	s.Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"resubmit", "--memoized", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		}).
		When().
		Given().
		RunCli([]string{"delete", "--resubmitted", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "deleted")
		})
}

func (s *CLISuite) TestWorkflowDeleteOlder() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"delete", "--older", "1d", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			// nothing over a day should be deleted
			assert.NotContains(t, output, "deleted")
		}).
		RunCli([]string{"delete", "--older", "0s", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "deleted")
		})
}

func (s *CLISuite) TestWorkflowDeleteByPrefix() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Given().
		RunCli([]string{"delete", "--prefix", "missing-prefix", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			// nothing should be deleted
			assert.NotContains(t, output, "deleted")
		}).
		RunCli([]string{"delete", "--prefix", "basic", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "deleted")
		})
}

func (s *CLISuite) TestWorkflowLint() {
	s.Run("LintFile", func() {
		s.Given().RunCli([]string{"lint", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "no linting errors found")
		})
	})
	s.Run("LintFileEmptyParamDAG", func() {
		s.Given().RunCli([]string{"lint", "expectedfailures/empty-parameter-dag.yaml"}, func(t *testing.T, output string, err error) {
			require.EqualError(t, err, "exit status 1")
			assert.Contains(t, output, "templates.abc.tasks.a templates.whalesay inputs.parameters.message was not supplied")
		})
	})
	s.Run("LintFileEmptyParamSteps", func() {
		s.Given().RunCli([]string{"lint", "expectedfailures/empty-parameter-steps.yaml"}, func(t *testing.T, output string, err error) {
			require.EqualError(t, err, "exit status 1")
			assert.Contains(t, output, "templates.abc.steps[0].a templates.whalesay inputs.parameters.message was not supplied")
		})
	})
	s.Run("LintFileWithTemplate", func() {
		s.Given().
			WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
			When().
			CreateWorkflowTemplates().
			Given().
			RunCli([]string{"lint", "smoke/hello-world-workflow-tmpl.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found")
			})
	})
	s.Run("LintDir", func() {
		tmp, err := os.MkdirTemp("", "")
		s.CheckError(err)
		defer func() { _ = os.RemoveAll(tmp) }()
		// Read all content of src to data
		data, err := os.ReadFile("smoke/basic.yaml")
		s.CheckError(err)
		// Write data to dst
		err = os.WriteFile(filepath.Join(tmp, "my-workflow.yaml"), data, 0o600)
		s.CheckError(err)
		s.Given().
			RunCli([]string{"lint", tmp}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found")
			})
	})

	s.Run("Different Kind", func() {
		s.Given().
			RunCli([]string{"lint", "testdata/workflow-template-nested-template.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found")
			})
	})
	s.Run("Lint Only Workflows", func() {
		s.Given().
			RunCli([]string{"lint", "--kinds", "wf", "testdata/workflow-template-nested-template.yaml"}, func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, output, "found nothing to lint in the specified paths, failing...")
				assert.NotContains(t, output, "Usage:")
			})
	})
	s.Run("All Kinds", func() {
		s.Given().
			RunCli([]string{"lint", "testdata/malformed/malformed-workflowtemplate-2.yaml"}, func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, output, "spec.templates[0].name is required")
				assert.Contains(t, output, "1 linting errors found!")
			})
	})
	s.Run("Valid", func() {
		s.Given().
			RunCli([]string{"lint", "testdata/exit-1.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found")
			})
	})
	s.Run("Invalid", func() {
		s.Given().
			RunCli([]string{"lint", "expectedfailures/empty-parameter-dag.yaml"}, func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, output, "1 linting errors found!")
				assert.Contains(t, output, "templates.abc.tasks.a templates.whalesay inputs.parameters.message was not supplied")
			})
	})
	s.Run("Lint Only CronWorkflows", func() {
		s.Given().
			RunCli([]string{"lint", "--kinds", "cronwf", "cron/cron-and-malformed-template.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found")
			})
	})
}

func (s *CLISuite) TestWorkflowOfflineLint() {
	s.setMode(OFFLINE)
	s.Run("LintFile", func() {
		s.Given().RunCli([]string{"lint", "--offline=true", "--kinds=workflows", "smoke/basic.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "no linting errors found")
		})
	})
}

func (s *CLISuite) TestWorkflowRetry() {
	var retryTime metav1.Time

	s.Given().
		Workflow("@testdata/retry-test.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.AnyActiveSuspendNode(), "suspended node"
		}), time.Second*90).
		RunCli([]string{"terminate", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow retry-test-.* terminated", output)
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			retryTime = wf.Status.FinishedAt
			return wf.Status.Phase == wfv1.WorkflowFailed, "is terminated"
		})).
		Wait(3*time.Second).
		RunCli([]string{"retry", "@latest", "--restart-successful", "--node-field-selector", "templateName==steps-inner"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err, output)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
		}).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.AnyActiveSuspendNode(), "suspended node"
		}), time.Second*90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			outerStepsPodNode := status.Nodes.FindByDisplayName("steps-outer-step1")
			innerStepsPodNode := status.Nodes.FindByDisplayName("steps-inner-step1")

			assert.True(t, outerStepsPodNode.FinishedAt.Before(&retryTime))
			assert.True(t, retryTime.Before(&innerStepsPodNode.FinishedAt))
		})
}

func (s *CLISuite) TestWorkflowRetryFailedWorkflow() {
	failFirstPassSecondWorkflowWhen := s.Given().
		Workflow("@testdata/fail-first-pass-second.yaml").
		When()

	failFirstPassSecondWorkflowWhen.
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		RunCli([]string{"retry", "-l", "workflows.argoproj.io/workflow=fail-first-pass-second-workflow", "--namespace=argo"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err, output)
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *CLISuite) TestWorkflowRetryNestedDag() {
	s.Given().
		Workflow("@testdata/retry-nested-dag-test.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		RunCli([]string{"retry", "retry-nested-dag", "--restart-successful", "--node-field-selector", "name=retry-nested-dag.dag1-step2.dag2-step1.dag3-step1"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err, output)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
		}).
		When().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("dag3-step3").Phase)
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("dag3-step2").Phase)
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("dag3-step1").Phase)
		})
}

func (s *CLISuite) TestWorkflowRetryWithRecreatedPVC() {
	s.Given().
		Workflow("@testdata/retry-with-recreated-pvc-test.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		RunCli([]string{"retry", "retry-with-recreated-pvc"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err, output)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
		}).
		When().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Nodes.FindByDisplayName("print").Phase)
			// This step is failed intentionally to allow retry. The error message is not related to PVC that is deleted
			// previously since it is re-created during retry.
			assert.Equal(t, "main: Error (exit code 1)", status.Nodes.FindByDisplayName("print").Message)
		})
}

func (s *CLISuite) TestRetryWorkflowWithContinueOn() {
	var workflowName string
	s.Given().
		Workflow(`@testdata/retry-workflow-with-continueon.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			workflowName = metadata.Name
			assert.Len(t, status.Nodes, 7)
		}).
		RunCli([]string{"retry", workflowName}, func(t *testing.T, output string, err error) {
			require.NoError(t, err, output)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
		})

	s.Given().
		When().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			workflowName = metadata.Name
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Len(t, status.Nodes, 7)
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return strings.Contains(status.Name, ".success")
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Len(t, status.Children, 2)
		})
}

func (s *CLISuite) TestRetryWorkflowWithFailedExitHandler() {
	var workflowName string
	s.Given().
		Workflow(`@testdata/retry-workflow-with-failed-exit-handler.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			workflowName = metadata.Name
			assert.Len(t, status.Nodes, 2)
		}).
		RunCli([]string{"retry", workflowName}, func(t *testing.T, output string, err error) {
			if assert.NoError(t, err, output) {
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
			}
		})

	s.Given().
		When().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			workflowName = metadata.Name
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Len(t, status.Nodes, 2)
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return status.Name == workflowName
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return strings.Contains(status.Name, ".onExit")
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			assert.Contains(t, status.Message, "exit code 1")
		})
}

func (s *CLISuite) TestWorkflowStop() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		RunCli([]string{"stop", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic-.* stopped", output)
		})
}

func (s *CLISuite) TestWorkflowStopDryRun() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"stop", "--dry-run", "basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic stopped \\(dry-run\\)", output)
		})
}

func (s *CLISuite) TestWorkflowStopBySelector() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"stop", "--selector", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic-.* stopped", output)
		})
}

func (s *CLISuite) TestWorkflowStopByFieldSelector() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"stop", "--field-selector", "metadata.namespace=argo"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic-.* stopped", output)
		})
}

func (s *CLISuite) TestWorkflowTerminate() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		RunCli([]string{"terminate", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic-.* terminated", output)
		})
}

func (s *CLISuite) TestWorkflowTerminateDryRun() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"terminate", "--dry-run", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow @latest terminated \\(dry-run\\)", output)
		})
}

func (s *CLISuite) TestWorkflowTerminateBySelector() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"terminate", "--selector", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic-.* terminated", output)
		})
}

func (s *CLISuite) TestWorkflowTerminateByFieldSelector() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		RunCli([]string{"terminate", "--field-selector", "metadata.namespace=argo"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Regexp(t, "workflow basic-.* terminated", output)
		})
}

func (s *CLISuite) TestWorkflowWait() {
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
			require.NoError(t, err)
			assert.Regexp(t, "basic-.* Succeeded", output)
		})
}

func (s *CLISuite) TestWorkflowWatch() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		Then().
		RunCli([]string{"watch", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name: ")
		})
}

func (s *CLISuite) TestTemplateCommands() {
	s.Run("LintWithoutArgs", func() {
		s.Given().RunCli([]string{"template", "lint"}, func(t *testing.T, output string, err error) {
			require.Error(t, err)
			assert.Contains(t, output, "Error: requires at least 1 arg(s), only received 0")
			assert.Contains(t, output, "Usage:")
		})
	})

	s.Run("Lint", func() {
		s.Given().RunCli([]string{"template", "lint", "testdata/basic-workflowtemplate.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "no linting errors found!")
		})
	})
	s.Run("DirLintWithInvalidWFT", func() {
		s.Given().RunCli([]string{"template", "lint", "testdata/workflow-templates"}, func(t *testing.T, output string, err error) {
			require.Error(t, err)
			assert.Contains(t, output, "invalid-workflowtemplate.yaml")
			assert.Contains(t, output, `unknown field "spec.entrypoints"`)
			assert.Contains(t, output, "linting errors found!")
		})
	})

	s.Run("Create", func() {
		s.Given().RunCli([]string{"template", "create", "testdata/basic-workflowtemplate.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("List", func() {
		s.Given().RunCli([]string{"template", "list"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "NAME")
		})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"template", "get", "not-found"}, func(t *testing.T, output string, err error) {
			require.EqualError(t, err, "exit status 1")
			assert.Contains(t, output, `"not-found" not found`)
		}).RunCli([]string{"template", "get", "basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("Submit", func() {
		s.Given().
			RunCli([]string{"submit", "--from", "workflowtemplate/basic"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			})
	})
	s.Run("Update", func() {
		s.Given().
			RunCli([]string{"template", "update", "testdata/basic-workflowtemplate-update.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"template", "get", "basic", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "entrypoint: main-2")
			assert.Contains(t, output, "name: main-2")
		})
	})
	s.Run("Update", func() {
		s.Given().
			RunCli([]string{"template", "update", "testdata/basic-workflowtemplate.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "Created:")
			})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"template", "get", "basic", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "entrypoint: main")
			assert.Contains(t, output, "name: main")
			assert.NotContains(t, output, "entrypoint: main-2")
			assert.NotContains(t, output, "name: main-2")
		})
	})
	s.Run("Delete", func() {
		s.Given().RunCli([]string{"template", "delete", "basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
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
		RunCli([]string{"resubmit", "--memoized", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		})
}

func (s *CLISuite) TestWorkflowResubmitDAGWithDependencies() {
	var wfString string
	s.Given().
		Workflow("@testdata/resubmit-dag-with-dependencies.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		RunCli([]string{"resubmit", "--memoized", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		}).
		RunCli([]string{"get", "@latest", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			wfString = output
		})

	s.Given().
		Workflow(wfString).
		When().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			assert.Len(t, status.Nodes, 5)
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return strings.Contains(status.Name, ".A")
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Equal(t, wfv1.NodeTypeSkipped, status.Type)
			assert.Contains(t, status.Message, "original pod")
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return strings.Contains(status.Name, ".B")
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			assert.Contains(t, status.Message, "exit code 1")
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return strings.Contains(status.Name, ".C")
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Equal(t, wfv1.NodeTypeSkipped, status.Type)
			assert.Contains(t, status.Message, "omitted: depends condition not met")
		}).
		ExpectWorkflowNode(func(status wfv1.NodeStatus) bool {
			return strings.Contains(status.Name, ".D")
		}, func(t *testing.T, status *wfv1.NodeStatus, pod *corev1.Pod) {
			assert.Equal(t, wfv1.NodeTypeSkipped, status.Type)
			assert.Contains(t, status.Message, "omitted: depends condition not met")
		})
}

func (s *CLISuite) TestWorkflowResubmitByLabelSelector() {
	s.Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"resubmit", "--selector", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		})
}

func (s *CLISuite) TestWorkflowResubmitByFieldSelector() {
	s.Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"resubmit", "--field-selector", "metadata.namespace=argo"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "ServiceAccount:")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Created:")
		})
}

func (s *CLISuite) TestCronCommands() {
	s.Run("Lint", func() {
		s.Given().RunCli([]string{"cron", "lint", "cron/basic.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "no linting errors found!")
		})
	})
	s.Run("Lint All Kinds", func() {
		s.Given().RunCli([]string{"lint", "cron/basic.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "no linting errors found!")
		})
	})
	s.Run("Different Kind", func() {
		s.Given().
			RunCli([]string{"cron", "lint", "testdata/workflow-template-nested-template.yaml"}, func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, output, "found nothing to lint in the specified paths, failing...")
			})
	})
	// Ignore other malformed kinds
	s.Run("IgnoreOtherKinds", func() {
		s.Given().
			RunCli([]string{"cron", "lint", "cron/cron-and-malformed-template.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found!")
			})
	})

	// All files in this directory are CronWorkflows, expect success
	s.Run("AllCron", func() {
		s.Given().
			RunCli([]string{"cron", "lint", "cron"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "no linting errors found!")
			})
	})

	s.Run("Create", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/basic.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
			assert.Contains(t, output, "Schedules:")
			assert.Contains(t, output, "Suspended:")
			assert.Contains(t, output, "StartingDeadlineSeconds:")
			assert.Contains(t, output, "ConcurrencyPolicy:")
		})
	})

	s.Run("Delete", func() {
		s.Given().RunCli([]string{"cron", "delete", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		})
	})

	s.Run("Create Schedule Override", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/basic.yaml", "--schedule", "1 2 3 * *", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Schedules:                     1 2 3 * *")
		})
	})
	s.Run("Update", func() {
		s.Given().
			RunCli([]string{"cron", "update", "cron/basic-update.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Schedules:                     5 5 5 * *")
			})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"cron", "get", "test-cron-wf-basic", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "5 5 5 * *")
			assert.Contains(t, output, "Replace")
			assert.Contains(t, output, "startingDeadlineSeconds: 10")
			assert.Contains(t, output, "successfulJobsHistoryLimit: 5")
			assert.Contains(t, output, "failedJobsHistoryLimit: 3")
			assert.Contains(t, output, "argosay")
		})
	})
	s.Run("Update", func() {
		s.Given().
			RunCli([]string{"cron", "update", "cron/basic-update-template.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Schedules:                     5 5 5 * *")
			})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"cron", "get", "test-cron-wf-basic", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "5 5 5 * *")
			assert.Contains(t, output, "Replace")
			assert.Contains(t, output, "whalesay")
			assert.NotContains(t, output, "argosay")
		})
	})
	s.Run("Create Parameter Override", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/param.yaml", "-p", "message=\"bar test passed\"", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "bar test passed")
		})
	})

	s.Run("Create Name Override", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/basic.yaml", "--name", "basic-cron-wf-overridden-name", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, strings.Replace(output, " ", "", -1), "Name:basic-cron-wf-overridden-name")
		})
	})

	s.Run("Create GenerateName Override", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/basic.yaml", "--generate-name", "basic-cron-wf-overridden-generate-name-", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, strings.Replace(output, " ", "", -1), "Name:basic-cron-wf-overridden-generate-name-")
		})
	})

	s.Run("List", func() {
		s.Given().RunCli([]string{"cron", "list"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "NAME")
			assert.Contains(t, output, "AGE")
			assert.Contains(t, output, "LAST RUN")
			assert.Contains(t, output, "SCHEDULES")
			assert.Contains(t, output, "SUSPENDED")
		})
	})

	s.Run("List with labels", func() {
		// First create cron workflow with labels
		s.Given().RunCli([]string{"cron", "create", "cron/cron-workflow-with-labels.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		}).
			// Then create cron workflow without labels
			RunCli([]string{"cron", "create", "cron/basic.yaml", "--name", "cron-wf-without-labels"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
			}).
			// Then check to make sure only cron workflow with labels shows up from 'argo cron list...'
			RunCli([]string{"cron", "list", "-l client=importantclient"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "NAME")
				assert.Contains(t, output, "AGE")
				assert.Contains(t, output, "LAST RUN")
				assert.Contains(t, output, "SCHEDULES")
				assert.Contains(t, output, "SUSPENDED")
				assert.Contains(t, output, "test-cwf-with-labels")
				assert.NotContains(t, output, "cron-wf-without-labels")
			})
	})

	s.Run("Suspend", func() {
		s.Given().RunCli([]string{"cron", "suspend", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "CronWorkflow 'test-cron-wf-basic' suspended")
		})
	})
	s.Run("Resume", func() {
		s.Given().RunCli([]string{"cron", "resume", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "CronWorkflow 'test-cron-wf-basic' resumed")
		})
	})
	s.Run("Get", func() {
		s.Given().RunCli([]string{"cron", "get", "not-found"}, func(t *testing.T, output string, err error) {
			require.EqualError(t, err, "exit status 1")
			assert.Contains(t, output, `"not-found" not found`)
		}).RunCli([]string{"cron", "get", "test-cron-wf-basic"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
			assert.Contains(t, output, "Schedules:")
			assert.Contains(t, output, "Suspended:")
			assert.Contains(t, output, "StartingDeadlineSeconds:")
			assert.Contains(t, output, "ConcurrencyPolicy:")
		})
	})
	s.Run("Create Multiple Schedules", func() {
		s.Given().RunCli([]string{"cron", "create", "cron/multiple-schedules.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Schedules:                     * * * * *,*/2 * * * *")
		})
	})
}

func (s *CLISuite) TestCronBackfillCommands() {
	s.Run("Backfill", func() {
		s.Given().RunCli([]string{"template", "create", "cron/cron-backfill-template.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		})
		s.Given().RunCli([]string{"cron", "create", "cron/cron-daily-job.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Schedules:                     0 2 * * *")
		})
		s.Given().RunCli([]string{"cron", "backfill", "daily-job", "--start", "Wed, 21 Oct 2024 15:28:00 GMT", "--end", "Wed, 21 Oct 2024 16:28:00 GMT", "--argname", "date"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "There is no suitable scheduling time.")
		})
		s.Given().RunCli([]string{"cron", "backfill", "daily-job", "--start", "Wed, 21 Oct 2024 15:28:00 GMT", "--end", "Sat, 24 Oct 2024 15:28:00 GMT", "--argname", "date", "--parallel", "true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Backfill task for Cronworkflow daily-job")
			assert.Contains(t, output, "Backfill Period :")
			assert.Contains(t, output, "Start Time : Wed, 21 Oct 2024 15:28:00 GMT")
			assert.Contains(t, output, "End Time : Sat, 24 Oct 2024 15:28:00 GMT")
			assert.Contains(t, output, "Total Backfill Schedule: 3")
		})
		s.Given().RunCli([]string{"watch", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "2024-10-22 02:00:00 +0000 GMT")
			assert.Contains(t, output, "2024-10-23 02:00:00 +0000 GMT")
			assert.Contains(t, output, "2024-10-24 02:00:00 +0000 GMT")
			require.Contains(t, output, "Status:              Succeeded")
		})
		s.Given().RunCli([]string{"cron", "delete", "daily-job"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		})
		s.Given().RunCli([]string{"template", "delete", "job"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		})
	})
}

func (s *CLISuite) TestClusterTemplateCommands() {
	s.Run("Create", func() {
		s.Given().
			RunCli([]string{"cluster-template", "create", "smoke/cluster-workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
			})
	})
	s.Run("Get", func() {
		s.Given().
			RunCli([]string{"cluster-template", "get", "cluster-workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
			})
	})
	s.Run("Update", func() {
		s.Given().
			RunCli([]string{"cluster-template", "update", "smoke/cluster-workflow-template-whalesay-template-update.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
			})
	})
	s.Run("Get", func() {
		s.Given().
			RunCli([]string{"cluster-template", "get", "cluster-workflow-template-whalesay-template", "-o", "yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
				assert.Contains(t, output, "Updated: {{inputs.parameters.message}}")
			})
	})
	s.Run("Update", func() {
		s.Given().
			RunCli([]string{"cluster-template", "update", "smoke/cluster-workflow-template-whalesay-template-update-parameters.yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
			})
	})
	s.Run("Get", func() {
		s.Given().
			RunCli([]string{"cluster-template", "get", "cluster-workflow-template-whalesay-template", "-o", "yaml"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
				assert.Contains(t, output, "thanks")
			})
	})
	s.Run("list", func() {
		s.Given().
			RunCli([]string{"cluster-template", "list"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
			})
	})
	s.Run("Delete", func() {
		s.Given().
			RunCli([]string{"cluster-template", "delete", "cluster-workflow-template-whalesay-template"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
			})
	})
}

func (s *CLISuite) TestWorkflowTemplateRefSubmit() {
	s.Run("CreateWFT", func() {
		s.Given().RunCli([]string{"template", "create", "smoke/workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("CreateWF", func() {
		s.Given().RunCli([]string{"submit", "testdata/workflow-template-ref.yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("CreateCWFT", func() {
		s.Given().RunCli([]string{"cluster-template", "create", "smoke/cluster-workflow-template-whalesay-template.yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Created:")
		})
	})
	s.Run("CreateWFWithCWFTRef", func() {
		s.Given().RunCli([]string{"submit", "testdata/cluster-workflow-template-ref.yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Name:")
			assert.Contains(t, output, "Namespace:")
			assert.Contains(t, output, "Created:")
		})
	})
}

func (s *CLISuite) TestWorkflowCopyArtifact() {
	s.workflowCopyArtifactTests("basic-artifact-workflow.yaml")
}

func (s *CLISuite) workflowCopyArtifactTests(workflowFileName string) {
	s.Given().
		Workflow(fmt.Sprintf("@testdata/%s", workflowFileName)).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"cp", "@latest", "outputDir"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Created \"main.log\"")
			assert.Contains(t, output, "Created \"hello_world.tgz\"")
			assert.Contains(t, output, "Created \"bye_world.tgz\"")
		})
	os.RemoveAll("outputDir")

	s.Given().
		Workflow(fmt.Sprintf("@testdata/%s", workflowFileName)).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"cp", "@latest", "outputDir", "--template-name", "bye"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "Created \"main.log\"")
			assert.Contains(t, output, "Created \"bye_world.tgz\"")
			assert.NotContains(t, output, "Created \"hello_world.tgz\"")
		})
	os.RemoveAll("outputDir")

	s.Given().
		Workflow(fmt.Sprintf("@testdata/%s", workflowFileName)).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"cp", "@latest", "outputDir", "--artifact-name", "hello_world"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.NotContains(t, output, "Created \"main.log\"")
			assert.NotContains(t, output, "Created \"bye_world.tgz\"")
			assert.Contains(t, output, "Created \"hello_world.tgz\"")
		})
	os.RemoveAll("outputDir")

	s.Given().
		Workflow(fmt.Sprintf("@testdata/%s", workflowFileName)).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Given().
		RunCli([]string{"cp", "@latest", ".", "--path", "/{templateName}/{artifactName}/"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			//Assert everything was stored
			assert.Contains(t, output, "Created \"main.log\"")
			assert.Contains(t, output, "Created \"bye_world.tgz\"")
			assert.Contains(t, output, "Created \"hello_world.tgz\"")
			//Assert filepaths are correct
			statStrip := func(f os.FileInfo, err error) error {
				return err
			}
			require.NoError(t, statStrip(os.Stat("bye/bye_world/bye_world.tgz")))
			require.NoError(t, statStrip(os.Stat("hello/hello_world/hello_world.tgz")))
		})
	os.RemoveAll("bye")
	os.RemoveAll("hello")
}

func (s *CLISuite) TestRetryOmit() {
	s.Given().
		Workflow("@testdata/retry-omit.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeOmitted, status.Nodes.FindByDisplayName("O").Phase)
		}).
		RunCli([]string{"retry", "@latest"}, fixtures.NoError)
}

func (s *CLISuite) TestResourceTemplateStopAndTerminate() {
	s.Run("ResourceTemplateStop", func() {
		s.Given().
			WorkflowName("resource-tmpl-wf").
			When().
			RunCli([]string{"submit", "functional/resource-template.yaml", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Pending")
			}).
			WaitForWorkflow(fixtures.ToHaveRunningPod).
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
			RunCli([]string{"submit", "functional/resource-template.yaml", "--name", "resource-tmpl-wf-1", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
				assert.Contains(t, output, "Pending")
			}).
			WaitForWorkflow(fixtures.ToHaveRunningPod).
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

func (s *CLISuite) TestAuthToken() {
	s.Given().RunCli([]string{"auth", "token"}, func(t *testing.T, output string, err error) {
		require.NoError(t, err)
		assert.NotEmpty(t, output)
	})
}

func (s *CLISuite) TestArchive() {
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.Run("List", func() {
		s.Given().
			RunCli([]string{"archive", "list", "--chunk-size", "1"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				lines := strings.Split(output, "\n")
				assert.Contains(t, lines[0], "NAMESPACE")
				assert.Contains(t, lines[0], "NAME")
				assert.Contains(t, lines[0], "STATUS")
				assert.Contains(t, lines[0], "UID")
				assert.Contains(t, lines[1], "argo")
				assert.Contains(t, lines[1], "basic")
				assert.Contains(t, lines[1], "Succeeded")
			})
	})
	s.Run("Get", func() {
		s.Given().
			RunCli([]string{"archive", "get", string(uid)}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Name:")
				assert.Contains(t, output, "Namespace:")
				assert.Contains(t, output, "ServiceAccount:")
				assert.Contains(t, output, "Status:")
				assert.Contains(t, output, "Created:")
				assert.Contains(t, output, "Started:")
				assert.Contains(t, output, "Finished:")
				assert.Contains(t, output, "Duration:")
			})
	})
	s.Run("Delete", func() {
		s.Given().
			RunCli([]string{"archive", "delete", string(uid)}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Archived workflow")
				assert.Contains(t, output, "deleted")
			})
	})
}

func (s *CLISuite) TestArchiveLabel() {
	s.Given().
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		When().
		CreateWorkflowTemplates().
		RunCli([]string{"submit", "--from", "workflowtemplate/workflow-template-whalesay-template", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
	s.Run("ListKeys", func() {
		s.Given().
			RunCli([]string{"archive", "list-label-keys"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				lines := strings.Split(output, "\n")
				assert.Contains(t, lines, "workflows.argoproj.io/test")
			})
	})
	s.Run("ListValues", func() {
		s.Given().
			RunCli([]string{"archive", "list-label-values", "-l", "workflows.argoproj.io/test"}, func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				lines := strings.Split(output, "\n")
				assert.Contains(t, lines[0], "true")
			})
	})
}

func (s *CLISuite) TestArgoSetOutputs() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: suspend-template-
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
		RunCli([]string{"resume", "@latest"}, func(t *testing.T, output string, err error) {
			require.Error(t, err)
			assert.Contains(t, output, "has not been set and does not have a default value")
		}).
		RunCli([]string{"node", "set", "@latest", "--output-parameter", "message=\"Hello, World!\"", "--node-field-selector", "displayName=approve"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "workflow values set")
		}).
		RunCli([]string{"node", "set", "@latest", "--output-parameter", "message=\"Hello, World!\"", "--node-field-selector", "displayName=approve"}, func(t *testing.T, output string, err error) {
			// Cannot double-set the same parameter
			require.Error(t, err)
			assert.Contains(t, output, "it was already set")
		}).
		RunCli([]string{"node", "set", "@latest", "--output-parameter", "message=\"Hello, World!\"", "--node-field-selector", "displayName=approve-no-vars"}, func(t *testing.T, output string, err error) {
			require.Error(t, err)
			assert.Contains(t, output, "cannot set output parameters because node is not expecting any raw parameters")
		}).
		RunCli([]string{"node", "set", "@latest", "--message", "Test message", "--node-field-selector", "displayName=approve"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "workflow values set")
		}).
		RunCli([]string{"resume", "@latest"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			assert.Contains(t, output, "workflow @latest resumed")
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("release")
			require.NotNil(t, nodeStatus)
			assert.Equal(t, "Hello, World!", nodeStatus.Inputs.Parameters[0].Value.String())

			nodeStatus = status.Nodes.FindByDisplayName("approve")
			require.NotNil(t, nodeStatus)
			assert.Equal(t, "Test message; Resumed by: map[User:system:serviceaccount:argo:argo-server]", nodeStatus.Message)
		})
}

func (s *CLISuite) TestPluginStruct() {
	s.setMode(GRPC)
	s.Given().
		Workflow("@testdata/plugins/executor/template-executor-workflow.yaml").
		When().
		SubmitWorkflow().
		Then().
		RunCli([]string{"get", "@latest", "-o", "yaml"}, func(t *testing.T, output string, err error) {
			require.NoError(t, err)
			wf := wfv1.Workflow{}
			require.NoError(t, yaml.UnmarshalStrict([]byte(output), &wf))
			assert.NotNil(t, wf.Spec.Templates[0].Plugin)
		})
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
