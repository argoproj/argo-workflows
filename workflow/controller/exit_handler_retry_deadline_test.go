package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const (
	retryDeadlineMessage = "retry exceeded workflow deadline"
	stepDeadlineMessage  = "Step exceeded its deadline"
)

func nodeSummary(wf *wfv1.Workflow) string {
	names := make([]string, 0, len(wf.Status.Nodes))
	byName := make(map[string]wfv1.NodeStatus, len(wf.Status.Nodes))
	for _, n := range wf.Status.Nodes {
		names = append(names, n.Name)
		byName[n.Name] = n
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		n := byName[name]
		fmt.Fprintf(&b, "%s type=%s phase=%s boundary=%q display=%q msg=%q\n",
			n.Name, n.Type, n.Phase, n.BoundaryID, n.DisplayName, n.Message)
	}
	return b.String()
}

// exitDeadlineWorkflow is the #17265 workflow: activeDeadlineSeconds plus an
// onExit (or lifecycle exit hook) whose template gets a retryStrategy from
// templateDefaults. exitField is either `onExit: exit-handler` or a hooks.exit
// block; the controller runs both as "<wf>.onExit".
func exitDeadlineWorkflow(name, exitField string) *wfv1.Workflow {
	return wfv1.MustUnmarshalWorkflow(fmt.Sprintf(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: %s
  namespace: default
spec:
  entrypoint: main
  activeDeadlineSeconds: 30
  %s
  templateDefaults:
    retryStrategy:
      limit: 1
  templates:
  - name: main
    container:
      image: busybox
      command: [echo, main]
  - name: exit-handler
    container:
      image: busybox
      command: [echo, exit]
`, name, exitField))
}

// reachExitAttempt runs the entrypoint, fails it after activeDeadlineSeconds
// has passed, and reconciles the operate that starts the exit handler. The
// exit attempt is left Pending: nodes are failed for the deadline at the
// start of operate, before the handler is created.
func reachExitAttempt(ctx context.Context, t *testing.T, wf *wfv1.Workflow) (*wfOperationCtx, *WorkflowController) {
	t.Helper()
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if strings.Contains(n.Name, ".onExit") {
			return ""
		}
		return apiv1.PodFailed
	})
	next := woc.wf.DeepCopy()
	next.Status.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
	woc = newWorkflowOperationCtx(ctx, next, controller)
	woc.operate(ctx)

	attemptName := wf.Name + ".onExit(0)"
	attempt, err := woc.wf.GetNodeByName(attemptName)
	require.NoError(t, err, "exit attempt was not created:\n%s", nodeSummary(woc.wf))
	require.Equal(t, wfv1.NodePending, attempt.Phase, "precondition: attempt still pending\n%s", nodeSummary(woc.wf))
	require.Empty(t, attempt.BoundaryID, "a workflow-level retry attempt keeps an empty boundary\n%s", nodeSummary(woc.wf))
	return woc, controller
}

func TestExitHandlerRetrySurvivesWorkflowDeadline(t *testing.T) {
	for _, tc := range []struct {
		name      string
		exitField string
	}{
		{name: "OnExit", exitField: "onExit: exit-handler"},
		{name: "HooksExit", exitField: "hooks:\n    exit:\n      template: exit-handler"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wfName := "exit-deadline-" + strings.ToLower(tc.name)
			wf := exitDeadlineWorkflow(wfName, tc.exitField)
			woc, controller := reachExitAttempt(ctx, t, wf)

			entry, err := woc.wf.GetNodeByName(wfName)
			require.NoError(t, err, nodeSummary(woc.wf))
			assert.Contains(t, entry.Message, retryDeadlineMessage, "a non-exit retry still stops at the workflow deadline\n%s", nodeSummary(woc.wf))

			woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
			woc.operate(ctx)
			summary := nodeSummary(woc.wf)

			attempt, err := woc.wf.GetNodeByName(wfName + ".onExit(0)")
			require.NoError(t, err, summary)
			assert.NotEqual(t, stepDeadlineMessage, attempt.Message, "pending exit attempt must not be failed for the workflow deadline\n%s", summary)
			assert.NotEqual(t, wfv1.NodeFailed, attempt.Phase, summary)
			assert.True(t, attempt.IsPartOfExitHandler(ctx, woc.wf.Status.Nodes), summary)

			exitNode, err := woc.wf.GetNodeByName(wfName + ".onExit")
			require.NoError(t, err, summary)
			assert.NotContains(t, exitNode.Message, retryDeadlineMessage, "exit retry must outlive the workflow deadline\n%s", summary)
			assert.NotEqual(t, wfv1.NodeFailed, exitNode.Phase, summary)

			entry, err = woc.wf.GetNodeByName(wfName)
			require.NoError(t, err, summary)
			assert.Equal(t, wfv1.NodeFailed, entry.Phase, summary)
			assert.Contains(t, entry.Message, retryDeadlineMessage, summary)
		})
	}
}

// TestExitHandlerRetryAfterDeadline retries a genuinely failed exit attempt
// once activeDeadlineSeconds is already past. The attempt is Failed before
// the reconcile, so fail-after-deadline does not touch it; processNodeRetries
// is what used to refuse the next attempt.
func TestExitHandlerRetryAfterDeadline(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := exitDeadlineWorkflow("exit-retry", "onExit: exit-handler")
	woc, controller := reachExitAttempt(ctx, t, wf)

	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.Name == "exit-retry.onExit(0)" {
			return apiv1.PodFailed
		}
		return ""
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	summary := nodeSummary(woc.wf)

	exitNode, err := woc.wf.GetNodeByName("exit-retry.onExit")
	require.NoError(t, err, summary)
	assert.NotContains(t, exitNode.Message, retryDeadlineMessage, summary)
	assert.NotEqual(t, wfv1.NodeFailed, exitNode.Phase, summary)

	next, err := woc.wf.GetNodeByName("exit-retry.onExit(1)")
	require.NoError(t, err, "retryStrategy must start attempt 1 after the workflow deadline\n%s", summary)
	assert.Equal(t, wfv1.NodePending, next.Phase, summary)
	assert.True(t, next.IsPartOfExitHandler(ctx, woc.wf.Status.Nodes), summary)

	entry, err := woc.wf.GetNodeByName("exit-retry")
	require.NoError(t, err, summary)
	assert.Contains(t, entry.Message, retryDeadlineMessage, "a non-exit retry still stops at the workflow deadline\n%s", summary)
}

// TestStepExitRetrySurvivesWorkflowDeadline is a step hooks.exit whose
// template has its own retryStrategy. The attempt's boundary is the steps
// node, so BoundaryID alone does not lead to the ".onExit" node.
func TestStepExitRetrySurvivesWorkflowDeadline(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: step-exit
  namespace: default
spec:
  entrypoint: main
  activeDeadlineSeconds: 30
  templates:
  - name: main
    steps:
    - - name: a
        template: work
        hooks:
          exit:
            template: exit-handler
  - name: work
    container:
      image: busybox
      command: [echo, work]
  - name: exit-handler
    retryStrategy:
      limit: 1
    container:
      image: busybox
      command: [echo, exit]
`)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	setPodPhases(ctx, woc, func(*wfv1.NodeStatus) apiv1.PodPhase { return apiv1.PodSucceeded })
	next := woc.wf.DeepCopy()
	next.Status.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
	woc = newWorkflowOperationCtx(ctx, next, controller)
	woc.operate(ctx)

	attempt, err := woc.wf.GetNodeByName("step-exit[0].a.onExit(0)")
	require.NoError(t, err, "step exit attempt was not created:\n%s", nodeSummary(woc.wf))
	require.Equal(t, wfv1.NodePending, attempt.Phase, nodeSummary(woc.wf))

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	summary := nodeSummary(woc.wf)

	attempt, err = woc.wf.GetNodeByName("step-exit[0].a.onExit(0)")
	require.NoError(t, err, summary)
	assert.NotEqual(t, stepDeadlineMessage, attempt.Message, summary)
	assert.NotEqual(t, wfv1.NodeFailed, attempt.Phase, summary)
	assert.True(t, attempt.IsPartOfExitHandler(ctx, woc.wf.Status.Nodes), summary)

	exitNode, err := woc.wf.GetNodeByName("step-exit[0].a.onExit")
	require.NoError(t, err, summary)
	assert.NotContains(t, exitNode.Message, retryDeadlineMessage, summary)
	assert.NotEqual(t, wfv1.NodeFailed, exitNode.Phase, summary)
}

// TestProcessNodeRetriesExitHandlerIgnoresWorkflowDeadline is the retry-node
// half of #17265: once an exit attempt has failed, processNodeRetries must
// not stop the Retry node because activeDeadlineSeconds is already past.
// A retry that is not an exit handler still stops.
func TestProcessNodeRetriesExitHandlerIgnoresWorkflowDeadline(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf
  namespace: default
spec:
  entrypoint: main
  activeDeadlineSeconds: 30
  templates:
  - name: main
    container:
      image: busybox
      command: [echo]
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	wf.Status.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.workflowDeadline = woc.getWorkflowDeadline()
	require.NotNil(t, woc.workflowDeadline)
	require.True(t, time.Now().UTC().After(*woc.workflowDeadline))

	retries := wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("1")}

	t.Run("ExitHandler", func(t *testing.T) {
		nodeName := "wf.onExit"
		woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Hooked: true}, true)
		woc.initializeNode(ctx, nodeName+"(0)", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Hooked: true, Retried: true}, true)
		woc.addChildNode(ctx, nodeName, nodeName+"(0)")

		node, err := woc.wf.GetNodeByName(nodeName)
		require.NoError(t, err)
		node, _, err = woc.processNodeRetries(ctx, node, retries, &executeTemplateOpts{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeRunning, node.Phase)
		assert.NotContains(t, node.Message, retryDeadlineMessage)
	})

	t.Run("NotExitHandler", func(t *testing.T) {
		nodeName := "wf"
		woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
		woc.initializeNode(ctx, nodeName+"(0)", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, nodeName+"(0)")

		node, err := woc.wf.GetNodeByName(nodeName)
		require.NoError(t, err)
		node, _, err = woc.processNodeRetries(ctx, node, retries, &executeTemplateOpts{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, retryDeadlineMessage)
	})
}

// TestDagDownstreamTaskAfterRetriedExitHookHonorsWorkflowDeadline verifies that
// when task a is a Retry node whose last child is its retried exit hook
// ("<wf>.a.onExit"), getOutboundNodes links dependent task b under that exit
// Retry node, yet b and b(0) are NOT classified as part of the exit handler and
// still stop at activeDeadlineSeconds.
func TestDagDownstreamTaskAfterRetriedExitHookHonorsWorkflowDeadline(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-exit-dep
  namespace: default
spec:
  entrypoint: main
  activeDeadlineSeconds: 30
  templateDefaults:
    retryStrategy:
      limit: 1
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: work
        hooks:
          exit:
            template: exit-handler
      - name: b
        depends: a
        template: work
  - name: work
    container:
      image: busybox
      command: [echo, work]
  - name: exit-handler
    container:
      image: busybox
      command: [echo, exit]
`)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Succeed a(0) so a.onExit(0) starts.
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.Name == "dag-exit-dep(0).a(0)" {
			return apiv1.PodSucceeded
		}
		return ""
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Succeed a.onExit(0) so dependent task b and b(0) are created (pending).
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.Name == "dag-exit-dep(0).a.onExit(0)" {
			return apiv1.PodSucceeded
		}
		return ""
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aExit, err := woc.wf.GetNodeByName("dag-exit-dep(0).a.onExit")
	require.NoError(t, err, nodeSummary(woc.wf))
	bRetry, err := woc.wf.GetNodeByName("dag-exit-dep(0).b")
	require.NoError(t, err, nodeSummary(woc.wf))
	assert.Contains(t, aExit.Children, bRetry.ID, "precondition: b is linked as an outbound child of a.onExit")
	bAttempt, err := woc.wf.GetNodeByName("dag-exit-dep(0).b(0)")
	require.NoError(t, err, nodeSummary(woc.wf))
	require.Equal(t, wfv1.NodePending, bAttempt.Phase, nodeSummary(woc.wf))
	assert.False(t, bRetry.IsPartOfExitHandler(ctx, woc.wf.Status.Nodes), nodeSummary(woc.wf))
	assert.False(t, bAttempt.IsPartOfExitHandler(ctx, woc.wf.Status.Nodes), nodeSummary(woc.wf))

	// Advance past activeDeadlineSeconds: pending b(0) must fail for the
	// deadline, and b's Retry node must stop with retry exceeded workflow deadline.
	expired := woc.wf.DeepCopy()
	expired.Status.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
	woc = newWorkflowOperationCtx(ctx, expired, controller)
	woc.operate(ctx)
	summary := nodeSummary(woc.wf)

	bAttempt, err = woc.wf.GetNodeByName("dag-exit-dep(0).b(0)")
	require.NoError(t, err, summary)
	assert.Equal(t, wfv1.NodeFailed, bAttempt.Phase, summary)
	assert.Equal(t, stepDeadlineMessage, bAttempt.Message, summary)

	bRetry, err = woc.wf.GetNodeByName("dag-exit-dep(0).b")
	require.NoError(t, err, summary)
	assert.Equal(t, wfv1.NodeFailed, bRetry.Phase, summary)
	assert.Contains(t, bRetry.Message, retryDeadlineMessage, summary)
}
