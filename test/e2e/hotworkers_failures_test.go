//go:build hotworkers

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// The tests in this file end pooled tasks in each way a pod can end, and in
// the ways only pools have. A worker that is told to stop hands its task
// back once its terminationGracePeriodSeconds less its
// releaseReserveSeconds have passed, 10 seconds in the profile, so a task
// that must be released runs for longer than that (20 seconds). Each
// workflow names its steps, so their nodes can be found by display name.

// pooledNodeIs is met once the node displayName is pooled and ok holds for
// its ledger.
func pooledNodeIs(displayName, what string, ok func(*wfv1.PooledTaskStatus) bool) fixtures.Condition {
	return func(wf *wfv1.Workflow) (bool, string) {
		n := wf.Status.Nodes.FindByDisplayName(displayName)
		return n != nil && n.PooledTask != nil && ok(n.PooledTask), displayName + " to be " + what
	}
}

// toBeClaimed is met once a worker holds the claim on the node displayName.
func toBeClaimed(displayName string) fixtures.Condition {
	return pooledNodeIs(displayName, "claimed", func(l *wfv1.PooledTaskStatus) bool { return l.State() == wfv1.PooledTaskClaimed })
}

// toBeReleased is met once the node displayName has been released.
func toBeReleased(displayName string) fixtures.Condition {
	return pooledNodeIs(displayName, "released", func(l *wfv1.PooledTaskStatus) bool { return l.Releases > 0 })
}

// toHaveNodeMessage is met once the node displayName's message contains
// message.
func toHaveNodeMessage(displayName, message string) fixtures.Condition {
	return func(wf *wfv1.Workflow) (bool, string) {
		n := wf.Status.Nodes.FindByDisplayName(displayName)
		return n != nil && strings.Contains(n.Message, message), fmt.Sprintf("%s to have message %q", displayName, message)
	}
}

// node returns the workflow's node displayName.
func node(t *testing.T, status *wfv1.WorkflowStatus, displayName string) *wfv1.NodeStatus {
	t.Helper()
	n := status.Nodes.FindByDisplayName(displayName)
	require.NotNil(t, n, "no node %s", displayName)
	return n
}

// podNodes returns the workflow's pod-type nodes, pooled or not.
func podNodes(status *wfv1.WorkflowStatus) []wfv1.NodeStatus {
	var nodes []wfv1.NodeStatus
	for _, n := range status.Nodes {
		if n.Type == wfv1.NodeTypePod {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// workerOf returns the worker pod that claimed the node displayName.
func workerOf(t *testing.T, wf *wfv1.Workflow, displayName string) string {
	t.Helper()
	n := node(t, &wf.Status, displayName)
	require.NotNil(t, n.PooledTask)
	require.NotEmpty(t, n.PooledTask.WorkerPodName)
	return n.PooledTask.WorkerPodName
}

// deletePod deletes the pod with the grace period given, or with its own
// when nil.
func (s *HotWorkersSuite) deletePod(name string, gracePeriodSeconds *int64) {
	s.T().Helper()
	ctx := logging.TestContext(s.T().Context())
	s.CheckError(s.KubeClient.CoreV1().Pods(fixtures.Namespace).Delete(ctx, name, metav1.DeleteOptions{GracePeriodSeconds: gracePeriodSeconds}))
}

// poolDeployments lists the Deployments of the pools named poolName.
func (s *HotWorkersSuite) poolDeployments(poolName string) []appsv1.Deployment {
	s.T().Helper()
	ctx := logging.TestContext(s.T().Context())
	list, err := s.KubeClient.AppsV1().Deployments(fixtures.Namespace).List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyPoolName + "=" + poolName})
	s.CheckError(err)
	return list.Items
}

// scalePool sets the replicas of every Deployment of the pools named
// poolName.
func (s *HotWorkersSuite) scalePool(poolName string, replicas int32) {
	s.T().Helper()
	ctx := logging.TestContext(s.T().Context())
	deployments := s.poolDeployments(poolName)
	s.Require().NotEmpty(deployments, "pool %s has no Deployment", poolName)
	patch, err := json.Marshal(map[string]any{"spec": map[string]any{"replicas": replicas}})
	s.CheckError(err)
	for _, d := range deployments {
		_, err := s.KubeClient.AppsV1().Deployments(fixtures.Namespace).Patch(ctx, d.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		s.CheckError(err)
	}
}

// expectNoPooledTaskSets checks the workflow left no pooled WorkflowTaskSet.
func (s *HotWorkersSuite) expectNoPooledTaskSets(wfName string) {
	s.T().Helper()
	ctx := logging.TestContext(s.T().Context())
	taskSets := versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTaskSets(fixtures.Namespace)
	list, err := taskSets.List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + wfName + "," + common.LabelKeyPoolHash})
	s.CheckError(err)
	s.Empty(list.Items, "pooled TaskSets left by %s", wfName)
}

// sleeper is a workflow whose one step sleeps on a one-worker pool for the
// seconds given. The seconds are a per-task arg, so every sleeper on a pool
// name shares the pool.
func sleeper(poolName string, seconds int) string {
	return fmt.Sprintf(`
metadata:
  generateName: pool-sleep-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: sleep
    - name: sleep
      pool:
        name: %s
        retentionSeconds: 300
      container:
        image: argoproj/argosay:v2
        command: [sleep, "%d"]
`, poolName, seconds)
}

// warmPool runs a short sleeper on the pool, creating it if it doesn't
// exist, so a test whose timeout or deadline counts from its task's start
// doesn't spend it waiting for the pool's pod to start.
func (s *HotWorkersSuite) warmPool(poolName string) {
	s.T().Helper()
	s.Given().
		Workflow(sleeper(poolName, 1)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 2*time.Minute)
}

// expectWorkerFree checks the pool's worker is free to run another task, so
// the task it ran before was stopped: a short sleeper on the pool succeeds,
// run by the same worker.
func (s *HotWorkersSuite) expectWorkerFree(poolName, worker string) {
	s.T().Helper()
	s.Given().
		Workflow(sleeper(poolName, 1)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 45*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, worker, workerOf(t, &wfv1.Workflow{Status: *status}, "task"))
		})
}

// TestForceDeleteWorker force-deletes the worker pod running a task, which
// then can't hand the task back: the attempt fails with Error, and
// retryPolicy OnError retries it.
func (s *HotWorkersSuite) TestForceDeleteWorker() {
	w := s.Given().
		Workflow(`
metadata:
  generateName: pool-force-delete-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: task
    - name: task
      pool:
        name: e2e-force-delete
        retentionSeconds: 300
      retryStrategy:
        limit: 1
        retryPolicy: OnError
      script:
        image: argoproj/argosay:v2
        command: [sh]
        source: |
          [ {{retries}} = 0 ] && sleep 300
          echo done
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("task(0)"), time.Minute)
	worker := workerOf(s.T(), w.GetWorkflow(), "task(0)")
	s.deletePod(worker, new(int64(0)))
	w.WaitForWorkflow(fixtures.ToBeSucceeded, 2*time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			first := node(t, status, "task(0)")
			assert.Equal(t, wfv1.NodeError, first.Phase)
			assert.Contains(t, first.Message, "worker pod "+worker+" deleted")
			assert.Equal(t, wfv1.NodeSucceeded, node(t, status, "task(1)").Phase)
		})
	s.expectNoPooledTaskSets(w.GetWorkflow().Name)
}

// expectReleasedOnce checks the workflow's one task succeeded on a worker
// other than worker after one release, as its only attempt.
func expectReleasedOnce(worker string) func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
	return func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
		nodes := podNodes(status)
		require.Len(t, nodes, 1, "a release uses no retry")
		n := nodes[0]
		assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
		require.NotNil(t, n.PooledTask)
		assert.Equal(t, int32(1), n.PooledTask.Releases)
		assert.NotEqual(t, worker, n.PooledTask.WorkerPodName, "run again by another worker")
	}
}

// TestGracefulDeleteWorker deletes the worker pod running a task with its
// grace period: the worker hands the task back, and the pod that replaces
// it runs the task again without using a retry.
func (s *HotWorkersSuite) TestGracefulDeleteWorker() {
	w := s.Given().
		Workflow(sleeper("e2e-graceful-delete", 20)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("task"), time.Minute)
	worker := workerOf(s.T(), w.GetWorkflow(), "task")
	s.deletePod(worker, nil)
	w.WaitForWorkflow(fixtures.ToBeSucceeded, 2*time.Minute).
		Then().
		ExpectWorkflow(expectReleasedOnce(worker))
}

// TestScaleDown scales a pool to zero while its worker runs a task: the
// worker hands the task back, which waits for a worker, and once the pool
// is scaled up again runs without using a retry.
func (s *HotWorkersSuite) TestScaleDown() {
	const poolName = "e2e-scale-down"
	w := s.Given().
		Workflow(sleeper(poolName, 20)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("task"), time.Minute)
	worker := workerOf(s.T(), w.GetWorkflow(), "task")
	s.scalePool(poolName, 0)
	w.WaitForWorkflow(toBeReleased("task"), time.Minute)
	s.scalePool(poolName, 1)
	w.WaitForWorkflow(fixtures.ToBeSucceeded, 2*time.Minute).
		Then().
		ExpectWorkflow(expectReleasedOnce(worker))
}

// TestOOMKilled runs a task that its main container's memory limit kills:
// the attempt fails with Failed, as a pod's does, and retryPolicy
// OnFailure retries it.
func (s *HotWorkersSuite) TestOOMKilled() {
	s.Given().
		Workflow(`
metadata:
  generateName: pool-oom-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: task
    - name: task
      pool:
        name: e2e-oom
        retentionSeconds: 300
      retryStrategy:
        limit: 1
        retryPolicy: OnFailure
      script:
        image: argoproj/argosay:v2
        command: [sh]
        resources:
          requests:
            memory: 96Mi
          limits:
            memory: 96Mi
        source: |
          # tail holds a line of /dev/zero, which never ends, in memory.
          [ {{retries}} = 0 ] && tail /dev/zero
          echo done
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 2*time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			first := node(t, status, "task(0)")
			assert.Equal(t, wfv1.NodeFailed, first.Phase, first.Message)
			// The kubelet kills all of main's container, and the worker
			// can't read why, so the message isn't the pod's
			// OOMKilled (exit code 137), and there is no exit code. What
			// main logged before it died is archived.
			assert.Equal(t, "main: the container exited before the task ended", first.Message)
			require.NotNil(t, first.Outputs)
			assert.Nil(t, first.Outputs.ExitCode)
			assert.NotNil(t, first.Outputs.GetArtifactByName("main-logs"))
			assert.Equal(t, wfv1.NodeSucceeded, node(t, status, "task(1)").Phase)
		})
}

// TestStopWithBacklog stops a workflow while one of its tasks runs and the
// others wait for the pool's only worker: the workflow completes, the
// running task is stopped and no pooled TaskSet is left.
func (s *HotWorkersSuite) TestStopWithBacklog() {
	const poolName = "e2e-stop"
	w := s.Given().
		Workflow(fmt.Sprintf(`
metadata:
  generateName: pool-stop-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: sleep
            withSequence:
              count: "3"
    - name: sleep
      pool:
        name: %s
        retentionSeconds: 300
      container:
        image: argoproj/argosay:v2
        command: [sleep, "300"]
`, poolName)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			claimed := wf.Status.Nodes.Any(func(n wfv1.NodeStatus) bool {
				return n.PooledTask != nil && n.PooledTask.State() == wfv1.PooledTaskClaimed
			})
			return claimed && len(podNodes(&wf.Status)) == 3, "to run one task with two waiting"
		}), time.Minute)
	var worker string
	for _, n := range podNodes(&w.GetWorkflow().Status) {
		if n.PooledTask != nil && n.PooledTask.WorkerPodName != "" {
			worker = n.PooledTask.WorkerPodName
		}
	}
	w.ShutdownWorkflow(wfv1.ShutdownStrategyStop).
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			for _, n := range podNodes(status) {
				assert.Equal(t, wfv1.NodeFailed, n.Phase, n.DisplayName)
				assert.True(t, n.TaskResultSynced != nil && *n.TaskResultSynced, n.DisplayName)
			}
		})
	s.expectNoPooledTaskSets(w.GetWorkflow().Name)
	s.expectWorkerFree(poolName, worker)
}

// TestTerminate terminates a workflow while its task runs: the task is
// killed, freeing its worker, and the workflow completes.
func (s *HotWorkersSuite) TestTerminate() {
	const poolName = "e2e-terminate"
	w := s.Given().
		Workflow(sleeper(poolName, 300)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("task"), time.Minute)
	worker := workerOf(s.T(), w.GetWorkflow(), "task")
	w.ShutdownWorkflow(wfv1.ShutdownStrategyTerminate).
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, node(t, status, "task").Phase)
		})
	s.expectNoPooledTaskSets(w.GetWorkflow().Name)
	s.expectWorkerFree(poolName, worker)
}

// unschedulable is a pooled template whose pool's pods can never be
// scheduled, so its tasks stay queued. Its pool goes as soon as no task
// uses it.
const unschedulable = `
    - name: unschedulable
      pool:
        name: e2e-unschedulable
        retentionSeconds: 0
      nodeSelector:
        e2e-hot-workers: never
      container:
        image: argoproj/argosay:v2
        command: [sleep, "1"]
`

// TestTimeoutRunning runs a task past its template's timeout: the worker
// kills it and it fails with the absolute deadline's message.
func (s *HotWorkersSuite) TestTimeoutRunning() {
	s.warmPool("e2e-timeout")
	w := s.Given().
		Workflow(`
metadata:
  generateName: pool-timeout-running-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: sleep
    - name: sleep
      timeout: 15s
      pool:
        name: e2e-timeout
        retentionSeconds: 300
      container:
        image: argoproj/argosay:v2
        command: [sleep, "300"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("task"), time.Minute)
	worker := workerOf(s.T(), w.GetWorkflow(), "task")
	w.WaitForWorkflow(fixtures.ToBeFailed, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			n := node(t, status, "task")
			assert.Equal(t, wfv1.NodeFailed, n.Phase)
			assert.Contains(t, n.Message, "Step exceeded its deadline")
		})
	s.expectWorkerFree("e2e-timeout", worker)
}

// TestTimeoutQueued leaves a task queued past its template's timeout: it
// fails without a worker ever claiming it.
func (s *HotWorkersSuite) TestTimeoutQueued() {
	s.Given().
		Workflow(`
metadata:
  generateName: pool-timeout-queued-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: unschedulable
`+strings.Replace(unschedulable, "      pool:", "      timeout: 10s\n      pool:", 1)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			n := node(t, status, "task")
			assert.Equal(t, wfv1.NodeFailed, n.Phase)
			assert.Contains(t, n.Message, "timeout")
			require.NotNil(t, n.PooledTask)
			assert.Empty(t, n.PooledTask.WorkerPodName)
		})
}

// TestTimeoutReleased releases a running task with no worker to take it
// back, past its template's timeout: it fails with the absolute deadline's
// message.
func (s *HotWorkersSuite) TestTimeoutReleased() {
	const poolName = "e2e-timeout-released"
	s.warmPool(poolName)
	w := s.Given().
		Workflow(fmt.Sprintf(`
metadata:
  generateName: pool-timeout-released-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: sleep
    - name: sleep
      timeout: 40s
      pool:
        name: %s
        retentionSeconds: 0
      container:
        image: argoproj/argosay:v2
        command: [sleep, "300"]
`, poolName)).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("task"), time.Minute)
	s.scalePool(poolName, 0)
	w.WaitForWorkflow(toBeReleased("task"), time.Minute).
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			n := node(t, status, "task")
			assert.Equal(t, wfv1.NodeFailed, n.Phase)
			assert.Contains(t, n.Message, "Step exceeded its deadline")
		})
}

// TestWorkflowDeadline passes the workflow's deadline with one task running
// and one queued: both fail, and the workflow completes.
func (s *HotWorkersSuite) TestWorkflowDeadline() {
	s.warmPool("e2e-deadline")
	w := s.Given().
		Workflow(`
metadata:
  generateName: pool-deadline-
spec:
  entrypoint: main
  activeDeadlineSeconds: 20
  templates:
    - name: main
      dag:
        tasks:
          - name: running
            template: sleep
          - name: queued
            template: unschedulable
    - name: sleep
      pool:
        name: e2e-deadline
        retentionSeconds: 300
      container:
        image: argoproj/argosay:v2
        command: [sleep, "300"]
`+unschedulable).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toBeClaimed("running"), time.Minute)
	worker := workerOf(s.T(), w.GetWorkflow(), "running")
	w.WaitForWorkflow(fixtures.ToBeFailed, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			for _, name := range []string{"running", "queued"} {
				n := node(t, status, name)
				assert.Equal(t, wfv1.NodeFailed, n.Phase, name)
				assert.Contains(t, n.Message, "deadline", name)
			}
		})
	s.expectNoPooledTaskSets(w.GetWorkflow().Name)
	s.expectWorkerFree("e2e-deadline", worker)
}

// TestRetention lets a pool sit idle past its retentionSeconds: it is
// deleted, and the next task creates it afresh.
func (s *HotWorkersSuite) TestRetention() {
	const poolName = "e2e-retention"
	workflow := fmt.Sprintf(`
metadata:
  generateName: pool-retention-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: echo
    - name: echo
      pool:
        name: %s
        retentionSeconds: 5
      container:
        image: argoproj/argosay:v2
        command: [echo, hello]
`, poolName)
	run := func() {
		s.Given().
			Workflow(workflow).
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute)
	}
	run()
	first := s.poolDeployments(poolName)
	s.Require().Len(first, 1)
	s.Eventually(func() bool { return len(s.poolDeployments(poolName)) == 0 }, time.Minute, time.Second, "the idle pool is deleted")
	run()
	second := s.poolDeployments(poolName)
	s.Require().Len(second, 1)
	s.NotEqual(first[0].UID, second[0].UID, "the pool is created afresh")
}

// TestRetry retries a workflow whose second pooled step failed: only that
// step runs again, as a fresh task, and succeeds.
func (s *HotWorkersSuite) TestRetry() {
	w := s.Given().
		Workflow(`
metadata:
  generateName: pool-retry-
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: pass
        value: "false"
  templates:
    - name: main
      steps:
        - - name: first
            template: check
            arguments:
              parameters:
                - name: pass
                  value: "true"
        - - name: second
            template: check
            arguments:
              parameters:
                - name: pass
                  value: "{{workflow.parameters.pass}}"
    - name: check
      pool:
        name: e2e-retry
        retentionSeconds: 300
      inputs:
        parameters:
          - name: pass
      container:
        image: argoproj/argosay:v2
        command: [test, "{{inputs.parameters.pass}}", "=", "true"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute)
	before := w.GetWorkflow().Status.DeepCopy()
	name := w.GetWorkflow().Name
	w.Exec("../../dist/argo", []string{"-n", fixtures.Namespace, "retry", name, "-p", "pass=true"}, func(t *testing.T, output string, err error) {
		require.NoError(t, err, output)
	}).
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			first, firstBefore := node(t, status, "first"), node(t, before, "first")
			assert.Equal(t, wfv1.NodeSucceeded, first.Phase)
			require.NotNil(t, first.PooledTask)
			assert.Equal(t, firstBefore.PooledTask.TaskSetUID, first.PooledTask.TaskSetUID, "the succeeded step isn't run again")
			second, secondBefore := node(t, status, "second"), node(t, before, "second")
			assert.Equal(t, wfv1.NodeFailed, secondBefore.Phase)
			assert.Equal(t, wfv1.NodeSucceeded, second.Phase)
			require.NotNil(t, second.PooledTask)
			assert.NotEqual(t, secondBefore.PooledTask.TaskSetUID, second.PooledTask.TaskSetUID, "the failed step runs as a fresh task")
		})
	s.expectNoPooledTaskSets(name)
}

// TestDegradedPool runs a task on a pool whose pods can't start, as its
// main container's env names a Secret that doesn't exist yet: the task
// waits, its node saying why, and runs once the Secret is created.
func (s *HotWorkersSuite) TestDegradedPool() {
	const secret = "e2e-hot-workers-late"
	ctx := logging.TestContext(s.T().Context())
	secrets := s.KubeClient.CoreV1().Secrets(fixtures.Namespace)
	s.T().Cleanup(func() { _ = secrets.Delete(context.WithoutCancel(ctx), secret, metav1.DeleteOptions{}) })
	s.Given().
		Workflow(`
metadata:
  generateName: pool-degraded-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: task
            template: late
    - name: late
      pool:
        name: e2e-degraded
        retentionSeconds: 0
      container:
        image: argoproj/argosay:v2
        command: [sh, -c, '[ "$LATE" = here ]']
        env:
          - name: LATE
            valueFrom:
              secretKeyRef:
                name: `+secret+`
                key: value
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(toHaveNodeMessage("task", `pool "e2e-degraded" is degraded: CreateContainerConfigError: container "main"`), 2*time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			n := node(t, status, "task")
			assert.Equal(t, wfv1.NodePending, n.Phase)
			assert.Contains(t, n.Message, secret)
		}).
		When().
		And(func() {
			_, err := secrets.Create(ctx, &apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secret},
				StringData: map[string]string{"value": "here"},
			}, metav1.CreateOptions{})
			s.CheckError(err)
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded, 3*time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Empty(t, node(t, status, "task").Message, "the pool's recovery clears the message")
		})
}
