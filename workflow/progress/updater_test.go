package progress

import (
	"testing"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestUpdater(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	ns := "my-ns"
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"pod-1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, Progress: wfv1.Progress("25/50")},
				"pod-2": wfv1.NodeStatus{Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod, Progress: wfv1.Progress("50/150")},
				"http":  wfv1.NodeStatus{Phase: wfv1.NodeFailed, Type: wfv1.NodeTypeHTTP},
				"plug":  wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePlugin},
				"dag":   wfv1.NodeStatus{Children: []string{"pod-1", "pod-2", "http", "plug"}},
			},
		},
	}

	UpdateProgress(ctx, wf)

	nodes := wf.Status.Nodes
	assert.Equal(t, wfv1.Progress("50/50"), nodes["pod-1"].Progress, "succeeded pod is completed")
	assert.Equal(t, wfv1.Progress("50/150"), nodes["pod-2"].Progress, "running pod is unchanged")
	assert.Equal(t, wfv1.Progress("0/1"), nodes["http"].Progress, "failed http is unchanged")
	assert.Equal(t, wfv1.Progress("1/1"), nodes["plug"].Progress, "succeeded plug is completed")
	assert.Equal(t, wfv1.Progress("101/202"), nodes["dag"].Progress, "dag is summed up")
	assert.Equal(t, wfv1.Progress("101/202"), wf.Status.Progress, "wf is sum total")
}

func Test_executes(t *testing.T) {
	assert.False(t, executable(""))
	assert.True(t, executable(wfv1.NodeTypePod))
	assert.True(t, executable(wfv1.NodeTypeContainer))
	assert.False(t, executable(wfv1.NodeTypeSteps))
	assert.False(t, executable(wfv1.NodeTypeStepGroup))
	assert.False(t, executable(wfv1.NodeTypeDAG))
	assert.False(t, executable(wfv1.NodeTypeTaskGroup))
	assert.True(t, executable(wfv1.NodeTypeSuspend))
	assert.True(t, executable(wfv1.NodeTypeHTTP))
	assert.True(t, executable(wfv1.NodeTypePlugin))
}
