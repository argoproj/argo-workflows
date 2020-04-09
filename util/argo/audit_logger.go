package argo

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

type AuditLogger struct {
	kIf       kubernetes.Interface
	component string
	ns        string
}

type EventInfo struct {
	Type   string
	Reason string
}

type ObjectRef struct {
	Name            string
	Namespace       string
	ResourceVersion string
	UID             types.UID
}

const (
	EventReasonWorkflowRunning       = "WorkflowRunning"
	EventReasonWorkflowSucceeded     = "WorkflowSucceeded"
	EventReasonWorkflowFailed        = "WorkflowFailed"
	EventReasonWorkflowTimedOut      = "WorkflowTimedOut"
	EventReasonWorkflowNodeSucceeded = "WorkflowNodeSucceeded"
	EventReasonWorkflowNodeFailed    = "WorkflowNodeFailed"
	EventReasonWorkflowNodeError     = "WorkflowNodeError"
)

func (l *AuditLogger) logEvent(objMeta ObjectRef, gvk schema.GroupVersionKind, info EventInfo, message string, annotations map[string]string) {
	logCtx := log.WithFields(log.Fields{
		"type":   info.Type,
		"reason": info.Reason,
	})
	switch gvk.Kind {
	case "Workflow":
		logCtx = logCtx.WithField("workflow", objMeta.Name)
	default:
		logCtx = logCtx.WithField("name", objMeta.Name)
	}
	t := metav1.Time{Time: time.Now()}
	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%v.%x", objMeta.Name, t.UnixNano()),
			Annotations: annotations,
		},
		Source: corev1.EventSource{
			Component: l.component,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:            gvk.Kind,
			Name:            objMeta.Name,
			Namespace:       objMeta.Namespace,
			ResourceVersion: objMeta.ResourceVersion,
			APIVersion:      gvk.Version,
			UID:             objMeta.UID,
		},

		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Message:        message,
		Type:           info.Type,
		Reason:         info.Reason,
	}
	logCtx.WithField("event", event).Debug()
	_, err := l.kIf.CoreV1().Events(objMeta.Namespace).Create(&event)
	if err != nil {
		logCtx.Errorf("Unable to create audit event: %v", err)
		return
	}
}

func (l *AuditLogger) logWorkflowEvent(workflow *wfv1.Workflow, info EventInfo, message string, annotations map[string]string) {
	objectMeta := ObjectRef{
		Name:            workflow.ObjectMeta.Name,
		Namespace:       workflow.ObjectMeta.Namespace,
		ResourceVersion: workflow.ObjectMeta.ResourceVersion,
		UID:             workflow.ObjectMeta.UID,
	}
	l.logEvent(objectMeta, wfv1.WorkflowSchemaGroupVersionKind, info, message, annotations)
}

func (l *AuditLogger) LogWorkflowEvent(workflow *wfv1.Workflow, info EventInfo, message string) {
	l.logWorkflowEvent(workflow, info, message, nil)
}

func (l *AuditLogger) LogWorkflowNodeEvent(workflow *wfv1.Workflow, node *wfv1.NodeStatus) {
	l.logWorkflowEvent(
		workflow,
		EventInfo{Type: eventType(node.Phase), Reason: nodePhaseReason(node.Phase)},
		nodeMessage(node),
		map[string]string{
			common.AnnotationKeyNodeType: string(node.Type),
			common.AnnotationKeyNodeName: node.Name,
		})
}

func eventType(phase wfv1.NodePhase) string {
	return map[wfv1.NodePhase]string{
		wfv1.NodeError:     corev1.EventTypeWarning,
		wfv1.NodeFailed:    corev1.EventTypeWarning,
		wfv1.NodeSucceeded: corev1.EventTypeNormal,
	}[phase]
}
func nodePhaseReason(phase wfv1.NodePhase) string {
	return map[wfv1.NodePhase]string{
		wfv1.NodeError:     EventReasonWorkflowNodeError,
		wfv1.NodeFailed:    EventReasonWorkflowNodeFailed,
		wfv1.NodeSucceeded: EventReasonWorkflowNodeSucceeded,
	}[phase]
}

func nodeMessage(node *wfv1.NodeStatus) string {
	message := fmt.Sprintf("%v node %s", node.Phase, node.Name)
	if node.Message != "" {
		message = message + ": " + node.Message
	}
	return message
}

func NewAuditLogger(ns string, kIf kubernetes.Interface, component string) *AuditLogger {
	return &AuditLogger{
		ns:        ns,
		kIf:       kIf,
		component: component,
	}
}
