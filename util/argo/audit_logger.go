package argo

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

type AuditLogger struct {
	kIf        kubernetes.Interface
	component  string
	ns         string
	nodeEvents config.NodeEvents
}

type EventInfo struct {
	Type   string
	Reason string
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

func (l *AuditLogger) logEvent(name string, t time.Time, workflow *wfv1.Workflow, info EventInfo, message string, annotations map[string]string) {
	logCtx := log.WithFields(log.Fields{
		"type":     info.Type,
		"reason":   info.Reason,
		"workflow": workflow.Name,
	})
	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: annotations},
		Source:     corev1.EventSource{Component: l.component},
		InvolvedObject: corev1.ObjectReference{
			Kind:            wfv1.WorkflowSchemaGroupVersionKind.Kind,
			APIVersion:      wfv1.SchemeGroupVersion.String(),
			Name:            workflow.Name,
			Namespace:       workflow.Namespace,
			ResourceVersion: workflow.ResourceVersion,
			UID:             workflow.UID,
		},
		FirstTimestamp: metav1.Time{Time: t},
		LastTimestamp:  metav1.Time{Time: t},
		Count:          1,
		Message:        message,
		Type:           info.Type,
		Reason:         info.Reason,
	}
	logCtx.WithField("event", event).Debug()
	_, err := l.kIf.CoreV1().Events(workflow.Namespace).Create(&event)
	if err != nil && !apierr.IsAlreadyExists(err) {
		logCtx.Errorf("Unable to create audit event: %v", err)
	}
}

func (l *AuditLogger) logWorkflowEvent(workflow *wfv1.Workflow, info EventInfo, message string, annotations map[string]string) {
	// workflows only have one terminal state - successful
	t := time.Now()
	name := fmt.Sprintf("%v.%x", workflow.Name, t.UnixNano())
	l.logEvent(name, t, workflow, info, message, annotations)
}

func (l *AuditLogger) LogWorkflowEvent(workflow *wfv1.Workflow, info EventInfo, message string) {
	l.logWorkflowEvent(workflow, info, message, nil)
}

func (l *AuditLogger) LogWorkflowNodeEvent(workflow *wfv1.Workflow, node *wfv1.NodeStatus) {
	if !l.nodeEvents.IsEnabled() {
		return
	}
	// nodes only have one terminal state - successful
	t := time.Now()
	name := fmt.Sprintf("%s.%s", workflow.Name, node.ID)
	if node.Phase != wfv1.NodeSucceeded {
		name = fmt.Sprintf("%s.%x", name, t.UnixNano())
	}
	l.logEvent(
		name,
		time.Now(),
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

func NewAuditLogger(ns string, kIf kubernetes.Interface, component string, nodeEvents config.NodeEvents) *AuditLogger {
	return &AuditLogger{
		ns:         ns,
		kIf:        kIf,
		component:  component,
		nodeEvents: nodeEvents,
	}
}
