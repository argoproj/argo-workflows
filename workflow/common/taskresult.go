package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/labels"
)

// WorkflowOwnerReference is the owner reference of an object the workflow
// owns, such as its task results.
func WorkflowOwnerReference(name string, uid types.UID) metav1.OwnerReference {
	return ownerReference(workflow.WorkflowKind, name, uid)
}

// TaskSetOwnerReference is the owner reference of an object a
// WorkflowTaskSet owns, such as a hot worker's claim on it.
func TaskSetOwnerReference(name string, uid types.UID) metav1.OwnerReference {
	return ownerReference(workflow.WorkflowTaskSetKind, name, uid)
}

func ownerReference(kind, name string, uid types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: workflow.APIVersion,
		Kind:       kind,
		Name:       name,
		UID:        uid,
	}
}

// NewTaskResult returns a node's WorkflowTaskResult as it is first created:
// labelled with its workflow and not yet complete, and owned by the workflow.
// The controller's result informer selects on the instance-ID label being
// absent when it has no instance ID, so the label is set only when instanceID
// is.
func NewTaskResult(namespace, name, workflowName string, workflowUID types.UID, instanceID string) *wfv1.WorkflowTaskResult {
	result := &wfv1.WorkflowTaskResult{
		TypeMeta: metav1.TypeMeta{
			APIVersion: workflow.APIVersion,
			Kind:       workflow.WorkflowTaskResultKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				LabelKeyWorkflow:               workflowName,
				LabelKeyReportOutputsCompleted: "false",
			},
			OwnerReferences: []metav1.OwnerReference{WorkflowOwnerReference(workflowName, workflowUID)},
		},
	}
	labels.Label(result, LabelKeyControllerInstanceID, instanceID)
	return result
}
