package argo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestAuditLogger_LogWorkflowEvent(t *testing.T) {
	kube := fake.NewSimpleClientset()
	l := AuditLogger{kIf: kube, component: "my-component"}
	l.LogWorkflowEvent(
		&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf", UID: "my-uid", ResourceVersion: "1234"},
		}, EventInfo{Type: corev1.EventTypeNormal, Reason: EventReasonWorkflowSucceeded}, "my message")
	list, err := kube.CoreV1().Events("my-ns").List(metav1.ListOptions{})
	if assert.NoError(t, err) && assert.Len(t, list.Items, 1) {
		e := list.Items[0]
		assert.Contains(t, e.Name, "my-wf.")
		assert.Len(t, e.Annotations, 0)
		assert.Equal(t, "my-component", e.Source.Component)
		assert.Equal(t, corev1.ObjectReference{
			Kind:            "Workflow",
			Namespace:       "my-ns",
			Name:            "my-wf",
			UID:             "my-uid",
			APIVersion:      "argoproj.io/v1alpha1",
			ResourceVersion: "1234",
		}, e.InvolvedObject)
		assert.NotEmpty(t, e.FirstTimestamp)
		assert.NotEmpty(t, e.LastTimestamp)
		assert.Equal(t, int32(1), e.Count)
		assert.Equal(t, "my message", e.Message)
		assert.Equal(t, "Normal", e.Type)
		assert.Equal(t, "WorkflowSucceeded", e.Reason)
	}
}

func TestAuditLogger_LogWorkflowNodeEvent(t *testing.T) {
	kube := fake.NewSimpleClientset()
	l := AuditLogger{kIf: kube, component: "my-component"}
	t.Run("Enabled", func(t *testing.T) {
		l.LogWorkflowNodeEvent(
			&wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf", UID: "my-uid", ResourceVersion: "1234"},
			},
			&wfv1.NodeStatus{ID: "my-node-id", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, Name: "my-node", Message: "my message"},
		)
		list, err := kube.CoreV1().Events("my-ns").List(metav1.ListOptions{})
		if assert.NoError(t, err) && assert.Len(t, list.Items, 1) {
			e := list.Items[0]
			// because success is the nodes terminal state, we do not append time,
			// which means we can create many with the same name (during workflow retry) and
			// this prevents create many events we don't need
			assert.Equal(t, "my-wf.my-node-id", e.Name)
			assert.Len(t, e.Annotations, 2)
			assert.Equal(t, "my-node", e.Annotations["workflows.argoproj.io/node-name"])
			assert.Equal(t, "Pod", e.Annotations["workflows.argoproj.io/node-type"])
			assert.Equal(t, "my-component", e.Source.Component)
			assert.Equal(t, corev1.ObjectReference{
				Kind:            "Workflow",
				Namespace:       "my-ns",
				Name:            "my-wf",
				UID:             "my-uid",
				APIVersion:      "argoproj.io/v1alpha1",
				ResourceVersion: "1234",
			}, e.InvolvedObject)
			assert.NotEmpty(t, e.FirstTimestamp)
			assert.NotEmpty(t, e.LastTimestamp)
			assert.Equal(t, int32(1), e.Count)
			assert.Equal(t, "Succeeded node my-node: my message", e.Message)
			assert.Equal(t, "Normal", e.Type)
			assert.Equal(t, "WorkflowNodeSucceeded", e.Reason)
		}
	})
	t.Run("Duplicate", func(t *testing.T) {
		l.LogWorkflowNodeEvent(&wfv1.Workflow{}, &wfv1.NodeStatus{})
		l.LogWorkflowNodeEvent(&wfv1.Workflow{}, &wfv1.NodeStatus{})
		list, err := kube.CoreV1().Events("my-ns").List(metav1.ListOptions{})
		if assert.NoError(t, err) {
			// not 2
			assert.Len(t, list.Items, 1)
		}
	})
	t.Run("Disabled", func(t *testing.T) {
		l.nodeEvents.Enabled = pointer.BoolPtr(false)
		l.LogWorkflowNodeEvent(&wfv1.Workflow{}, &wfv1.NodeStatus{})
		list, err := kube.CoreV1().Events("my-ns").List(metav1.ListOptions{})
		if assert.NoError(t, err) {
			// not 2
			assert.Len(t, list.Items, 1)
		}
	})
}

func Test_eventType(t *testing.T) {
	assert.Equal(t, corev1.EventTypeWarning, eventType(wfv1.NodeError))
	assert.Equal(t, corev1.EventTypeWarning, eventType(wfv1.NodeFailed))
	assert.Equal(t, corev1.EventTypeNormal, eventType(wfv1.NodeSucceeded))
}

func Test_nodePhaseReason(t *testing.T) {
	assert.Equal(t, EventReasonWorkflowNodeError, nodePhaseReason(wfv1.NodeError))
	assert.Equal(t, EventReasonWorkflowNodeFailed, nodePhaseReason(wfv1.NodeFailed))
	assert.Equal(t, EventReasonWorkflowNodeSucceeded, nodePhaseReason(wfv1.NodeSucceeded))
}

func Test_nodeMessage(t *testing.T) {
	assert.Equal(t, "Succeeded node my-node", nodeMessage(&wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Name: "my-node"}))
	assert.Equal(t, "Succeeded node my-node: my-message", nodeMessage(&wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Name: "my-node", Message: "my-message"}))
}
