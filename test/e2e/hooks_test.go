//go:build functional

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type HooksSuite struct {
	fixtures.E2ESuite
}

func (s *HooksSuite) TestWorkflowLevelHooksSuccessVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestWorkflowLevelHooksFailVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.failed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksStepSuccessVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook-tmpl-level.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
	// TODO: Temporarily comment out this assertion since it's flaky:
	// 	  The running hook is occasionally not triggered. Possibly because the step finishes too quickly
	//	  while the controller did not get a chance to trigger this hook.
	//.ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
	//	return strings.Contains(status.Name, "step-2.hooks.running")
	//}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
	//	assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	//})
}

func (s *HooksSuite) TestTemplateLevelHooksStepFailVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook-tmpl-level.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.failed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksDagSuccessVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook-tmpl-level.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		// TODO: Temporarily comment out this assertion since it's flaky:
		// 	  The running hook is occasionally not triggered. Possibly because the step finishes too quickly
		//	  while the controller did not get a chance to trigger this hook.
		//assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksDagFailVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook-tmpl-level.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.failed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksDagHasDependencyVersion() {
	s.Given().
		Workflow("@functional/lifecycle-hook-tmpl-level.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
			// Make sure unnecessary hooks are not triggered
			assert.Equal(t, status.Progress, v1alpha1.Progress("1/2"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "A.hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "B")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeOmitted, status.Phase)
		})
}

func (s *HooksSuite) TestWorkflowLevelHooksWaitForTriggeredHook() {
	s.Given().
		Workflow("@functional/lifecycle-hook.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("2/2"))
			assert.Equal(t, 1, int(status.Progress.N()/status.Progress.M()))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *HooksSuite) TestTemplateLevelHooksWaitForTriggeredHook() {
	s.Given().
		Workflow("@functional/example-steps.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("2/2"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "job.hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

// Ref: https://github.com/argoproj/argo-workflows/issues/11117
func (s *HooksSuite) TestTemplateLevelHooksWaitForTriggeredHookAndRespectSynchronization() {
	s.Given().
		Workflow("@functional/example-steps-simple-mutex.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("3/3"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "job.hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "job.hooks.succeed")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *HooksSuite) TestWorkflowLevelHooksWithRetry() {
	s.Given().
		Workflow("@functional/test-workflow-level-hooks-with-retry.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("2/4"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry.hooks.running"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
			assert.True(t, status.NodeFlag.Hooked)
			assert.False(t, status.NodeFlag.Retried)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry.hooks.failed"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
			assert.True(t, status.NodeFlag.Hooked)
			assert.False(t, status.NodeFlag.Retried)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.Equal(t, v1alpha1.NodeTypeRetry, status.Type)
			assert.Nil(t, status.NodeFlag)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry(0)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.False(t, status.NodeFlag.Hooked)
			assert.True(t, status.NodeFlag.Retried)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry(1)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.False(t, status.NodeFlag.Hooked)
			assert.True(t, status.NodeFlag.Retried)
		})
}

func (s *HooksSuite) TestTemplateLevelHooksWithRetry() {
	var children []string
	(s.Given().
		Workflow("@functional/retries-with-hooks-and-artifact.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.True(t, status.Fulfilled())
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			for _, node := range status.Nodes {
				if node.Type == v1alpha1.NodeTypeRetry {
					assert.Equal(t, v1alpha1.NodeSucceeded, node.Phase)
					children = node.Children
				}
			}
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[0].build(0)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Contains(t, children, status.ID)
			assert.False(t, status.NodeFlag.Hooked)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[0].build.hooks.started"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Contains(t, children, status.ID)
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[0].build.hooks.success"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Contains(t, children, status.ID)
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[1].print"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *HooksSuite) TestExitHandlerWithWorkflowLevelDeadline() {
	var onExitNodeName string
	(s.Given().
		Workflow("@functional/exit-handler-with-workflow-level-deadline.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted, 2*time.Minute).
		WaitForWorkflow(fixtures.Condition(func(wf *v1alpha1.Workflow) (bool, string) {
			onExitNodeName = common.GenerateOnExitNodeName(wf.ObjectMeta.Name)
			onExitNode := wf.Status.Nodes.FindByDisplayName(onExitNodeName)
			return onExitNode.Completed(), "exit handler completed"
		})).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.DisplayName == onExitNodeName
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}))
}

func (s *HooksSuite) TestHttpExitHandlerWithWorkflowLevelDeadline() {
	var onExitNodeName string
	(s.Given().
		Workflow("@functional/http-exit-handler-with-workflow-level-deadline.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		WaitForWorkflow(fixtures.Condition(func(wf *v1alpha1.Workflow) (bool, string) {
			onExitNodeName = common.GenerateOnExitNodeName(wf.ObjectMeta.Name)
			onExitNode := wf.Status.Nodes.FindByDisplayName(onExitNodeName)
			return onExitNode.Completed(), "exit handler completed"
		})).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.DisplayName == onExitNodeName
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}))
}

func TestHooksSuite(t *testing.T) {
	suite.Run(t, new(HooksSuite))
}
