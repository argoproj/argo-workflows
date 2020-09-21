package progress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	testutil "github.com/argoproj/argo/test/util"
	"github.com/argoproj/argo/workflow/common"
)

func TestUpdator(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"my-wf-pod-1": wfv1.NodeStatus{ID: "my-wf-pod-1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod},
				"my-wf-pod-2": wfv1.NodeStatus{ID: "my-wf-pod-2", Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod},
				"my-wf-pod-3": wfv1.NodeStatus{ID: "my-wf-pod-3", Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod},
				"my-wf":       wfv1.NodeStatus{Type: wfv1.NodeTypeDAG, Children: []string{"my-wf-pod-1", "my-wf-pod-2", "my-wf-pod-3"}},
			},
		},
	}
	podInformer := testutil.NewSharedIndexInformer()
	u := NewUpdator(podInformer, wf)
	podInformer.Indexer.SetByKey("my-ns/my-wf-pod-3", &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				common.AnnotationKeyProgress: "50/100",
			},
		},
	})
	u.Init()
	u.Visit("my-wf-pod-1")
	u.Visit("my-wf-pod-2")
	u.Visit("my-wf-pod-3")
	u.Visit("my-wf")
	assert.Equal(t, wfv1.Progress("1/1"), wf.Status.Nodes["my-wf-pod-1"].Progress)
	assert.Equal(t, wfv1.Progress("0/1"), wf.Status.Nodes["my-wf-pod-2"].Progress)
	assert.Equal(t, wfv1.Progress("50/100"), wf.Status.Nodes["my-wf-pod-3"].Progress)
	assert.Equal(t, wfv1.Progress("51/102"), wf.Status.Nodes["my-wf"].Progress)
	assert.Equal(t, wfv1.Progress("51/102"), wf.Status.Progress)
	assert.True(t, u.Updated)
}
