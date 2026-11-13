package common

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	workflowmocks "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// pooledTask is a ledger for taskName, enqueued and taken through steps:
// "claim" by worker, "release" and "interrupt".
func pooledTask(taskName, worker string, steps ...string) *wfv1.PooledTaskStatus {
	task := &wfv1.PooledTaskStatus{TaskName: taskName}
	now := metav1.Now()
	task.Enqueue(now)
	for _, step := range steps {
		switch step {
		case "claim":
			task.Claim("claim-uid", now, worker, "pod-uid", 0)
		case "release":
			task.Release(now)
		case "interrupt":
			task.Interrupt(now)
		}
	}
	return task
}

func pooledLogsWorkflow() *wfv1.Workflow {
	synced := true
	started := func(sec int64) metav1.Time { return metav1.Unix(sec, 0) }
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Namespace: "argo"},
		Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{
			"my-wf-1": {
				ID: "my-wf-1", DisplayName: "queued", Type: wfv1.NodeTypePod, Phase: wfv1.NodePending, StartedAt: started(1),
				PooledTask: pooledTask("my-wf-work-1", ""),
			},
			"my-wf-2": {
				ID: "my-wf-2", DisplayName: "running", Type: wfv1.NodeTypePod, Phase: wfv1.NodeRunning, StartedAt: started(2),
				PooledTask: pooledTask("my-wf-work-2", "work-0123456789-abcde", "claim"),
			},
			"my-wf-3": {
				ID: "my-wf-3", DisplayName: "archived", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, StartedAt: started(3), TaskResultSynced: &synced,
				Outputs:    &wfv1.Outputs{Artifacts: wfv1.Artifacts{{Name: "main-logs"}}},
				PooledTask: pooledTask("my-wf-work-3", "work-0123456789-abcde", "claim"),
			},
			"my-wf-4": {
				ID: "my-wf-4", DisplayName: "interrupted", Type: wfv1.NodeTypePod, Phase: wfv1.NodeFailed, StartedAt: started(4), TaskResultSynced: &synced,
				PooledTask: pooledTask("my-wf-work-4", "work-0123456789-fghij", "claim", "interrupt"),
			},
			"my-wf-5": {ID: "my-wf-5", DisplayName: "pod", Type: wfv1.NodeTypePod, Phase: wfv1.NodeRunning, StartedAt: started(5)},
		}},
	}
}

func Test_pooledLogNotes(t *testing.T) {
	queued := "my-wf-work-1: queued is a pooled task with no live logs; its logs are archived when it finishes, if log archiving is enabled"
	running := "work-0123456789-abcde: running is a pooled task with no live logs; its logs are archived when it finishes, if log archiving is enabled"
	archived := "work-0123456789-abcde: archived is a pooled task with no live logs; fetch its archived logs with `argo cp -n argo my-wf . --node-id my-wf-3 --artifact-name main-logs`"
	interrupted := "work-0123456789-fghij: interrupted is a pooled task with no live logs; it has no archived logs"
	tests := []struct {
		name      string
		podName   string
		container string
		want      []string
	}{
		{name: "whole workflow", container: "main", want: []string{queued, running, archived, interrupted}},
		{name: "by task name", podName: "my-wf-work-3", container: "main", want: []string{archived}},
		{name: "by worker pod", podName: "work-0123456789-abcde", container: "main", want: []string{running, archived}},
		{name: "a pod", podName: "my-wf-pod-5", container: "main", want: []string{}},
		{name: "a container without archived logs", podName: "my-wf-work-3", container: "wait", want: []string{
			"work-0123456789-abcde: archived is a pooled task with no live logs; it has no archived logs",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pooledLogNotes(pooledLogsWorkflow(), tt.podName, tt.container))
		})
	}
}

// captureStderr returns what f writes to os.Stderr.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	stderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = stderr }()
	f()
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

type emptyLogStream struct{ grpc.ClientStream }

func (emptyLogStream) Recv() (*workflowpkg.LogEntry, error) { return nil, io.EOF }

func TestLogWorkflowLooksUpPooledNodes(t *testing.T) {
	t.Run("without a selector", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		c.On("WorkflowLogs", mock.Anything, mock.Anything).Return(emptyLogStream{}, nil)
		c.On("GetWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowpkg.WorkflowGetRequest) bool {
			return req.Name == "my-wf" && req.Namespace == "argo"
		})).Return(pooledLogsWorkflow(), nil)
		err := LogWorkflow(logging.TestContext(t.Context()), c, "argo", "my-wf", "", "", "", &corev1.PodLogOptions{Container: "main"})
		require.NoError(t, err)
		c.AssertNumberOfCalls(t, "GetWorkflow", 1)
	})
	t.Run("naming a pooled task", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		c.On("WorkflowLogs", mock.Anything, mock.Anything).Return(emptyLogStream{}, nil)
		c.On("GetWorkflow", mock.Anything, mock.Anything).Return(pooledLogsWorkflow(), nil)
		err := LogWorkflow(logging.TestContext(t.Context()), c, "argo", "my-wf", "my-wf-work-3", "", "", &corev1.PodLogOptions{Container: "main"})
		require.EqualError(t, err, "my-wf-work-3 names only pooled tasks, which have no live logs")
	})
	t.Run("naming a pod", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		c.On("WorkflowLogs", mock.Anything, mock.Anything).Return(emptyLogStream{}, nil)
		c.On("GetWorkflow", mock.Anything, mock.Anything).Return(pooledLogsWorkflow(), nil)
		err := LogWorkflow(logging.TestContext(t.Context()), c, "argo", "my-wf", "my-wf-pod-5", "", "", &corev1.PodLogOptions{Container: "main"})
		require.NoError(t, err)
	})
	t.Run("when the lookup fails", func(t *testing.T) {
		for _, podName := range []string{"", "my-wf-work-3"} {
			c := &workflowmocks.WorkflowServiceClient{}
			c.On("WorkflowLogs", mock.Anything, mock.Anything).Return(emptyLogStream{}, nil)
			c.On("GetWorkflow", mock.Anything, mock.Anything).Return(nil, status.Error(codes.NotFound, "not found"))
			err := LogWorkflow(logging.TestContext(t.Context()), c, "argo", "my-wf", podName, "", "", &corev1.PodLogOptions{Container: "main"})
			require.NoError(t, err, "POD %q", podName)
		}
	})
	t.Run("with a selector", func(t *testing.T) {
		c := &workflowmocks.WorkflowServiceClient{}
		c.On("WorkflowLogs", mock.Anything, mock.Anything).Return(emptyLogStream{}, nil)
		c.On("GetWorkflow", mock.Anything, mock.Anything).Return(pooledLogsWorkflow(), nil)
		var err error
		stderr := captureStderr(t, func() {
			err = LogWorkflow(logging.TestContext(t.Context()), c, "argo", "my-wf", "", "", "app=x", &corev1.PodLogOptions{Container: "main"})
		})
		require.NoError(t, err)
		assert.Equal(t, strings.Join(pooledLogNotes(pooledLogsWorkflow(), "", "main"), "\n")+"\n", stderr)
	})
}

func TestWaitWatchOrLogSkipsPooledLookup(t *testing.T) {
	c := &workflowmocks.WorkflowServiceClient{}
	c.On("WorkflowLogs", mock.Anything, mock.Anything).Return(emptyLogStream{}, nil)
	err := WaitWatchOrLog(logging.TestContext(t.Context()), c, "argo", []string{"my-wf"}, CliSubmitOpts{Log: true})
	require.NoError(t, err)
	c.AssertNotCalled(t, "GetWorkflow", mock.Anything, mock.Anything)
}
