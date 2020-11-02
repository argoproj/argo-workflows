package progress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestUpdator(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"pod-1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod},
				"pod-2": wfv1.NodeStatus{Type: wfv1.NodeTypePod},
				"wf":    wfv1.NodeStatus{Children: []string{"pod-1", "pod-2"}},
			},
		},
	}
	UpdateProgress(wf)
	assert.Equal(t, wfv1.Progress("1/1"), wf.Status.Nodes["pod-1"].Progress)
	assert.Equal(t, wfv1.Progress("0/1"), wf.Status.Nodes["pod-2"].Progress)
	assert.Equal(t, wfv1.Progress("1/2"), wf.Status.Nodes["wf"].Progress)
	assert.Equal(t, wfv1.Progress("1/2"), wf.Status.Progress)
}
