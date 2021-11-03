package progress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestUpdater(t *testing.T) {
	ns := "my-ns"
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"pod-1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, Progress: wfv1.Progress("50/50")},
				"pod-2": wfv1.NodeStatus{Type: wfv1.NodeTypePod, Progress: wfv1.Progress("50/150")},
				"pod-3": wfv1.NodeStatus{Type: wfv1.NodeTypePod, Progress: wfv1.Progress("50/100")},
				"wf":    wfv1.NodeStatus{Children: []string{"pod-1", "pod-2", "pod-3"}},
			},
		},
	}

	UpdateProgress(wf)

	assert.Equal(t, wfv1.Progress("50/50"), wf.Status.Nodes["pod-1"].Progress)
	assert.Equal(t, wfv1.Progress("50/150"), wf.Status.Nodes["pod-2"].Progress)
	assert.Equal(t, wfv1.Progress("50/100"), wf.Status.Nodes["pod-3"].Progress)
	assert.Equal(t, wfv1.Progress("150/300"), wf.Status.Nodes["wf"].Progress)
	assert.Equal(t, wfv1.Progress("150/300"), wf.Status.Progress)
}
