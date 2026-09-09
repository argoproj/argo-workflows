package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// WorkflowActionType is the type of action to perform on the target workflow
type WorkflowActionType string

const (
	ActionTypeSuspend   WorkflowActionType = "Suspend"
	ActionTypeResume    WorkflowActionType = "Resume"
	ActionTypeStop      WorkflowActionType = "Stop"
	ActionTypeTerminate WorkflowActionType = "Terminate"
)

// WorkflowActionPhase is the lifecycle phase of a WorkflowAction
type WorkflowActionPhase string

const (
	WorkflowActionPending   WorkflowActionPhase = "Pending"
	WorkflowActionSucceeded WorkflowActionPhase = "Succeeded"
	WorkflowActionFailed    WorkflowActionPhase = "Failed"
)

// Machine-readable reasons for a Failed WorkflowAction
const (
	WorkflowActionReasonWorkflowNotFound  = "WorkflowNotFound"
	WorkflowActionReasonWorkflowCompleted = "WorkflowCompleted"
	WorkflowActionReasonInvalidAction     = "InvalidAction"
)

// WorkflowAction requests the workflow controller to perform a lifecycle action on a Workflow.
// The controller applies the action during reconciliation and reports the outcome on status.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=wfa
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.workflowRef.name`
// +kubebuilder:printcolumn:name="Action",type=string,JSONPath=`.spec.action`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type WorkflowAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowActionSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            WorkflowActionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// WorkflowActionRef identifies the target Workflow
type WorkflowActionRef struct {
	// Name of the target Workflow in the same namespace
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// UID optionally pins the target across name reuse. Reserved for archived targets (retry).
	UID types.UID `json:"uid,omitempty" protobuf:"bytes,2,opt,name=uid,casttype=k8s.io/apimachinery/pkg/types.UID"`
}

// WorkflowActionSpec is the requested action. It is immutable after creation.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable"
// +kubebuilder:validation:XValidation:rule="!has(self.resume) || self.action == 'Resume'",message="resume parameters require action: Resume"
// +kubebuilder:validation:XValidation:rule="!has(self.stop) || self.action == 'Stop'",message="stop parameters require action: Stop"
type WorkflowActionSpec struct {
	// WorkflowRef is the target Workflow
	WorkflowRef WorkflowActionRef `json:"workflowRef" protobuf:"bytes,1,opt,name=workflowRef"`
	// Action to perform on the target
	// +kubebuilder:validation:Enum=Suspend;Resume;Stop;Terminate
	Action WorkflowActionType `json:"action" protobuf:"bytes,2,opt,name=action,casttype=WorkflowActionType"`
	// Resume parameters, only for action: Resume
	Resume *ResumeAction `json:"resume,omitempty" protobuf:"bytes,3,opt,name=resume"`
	// Stop parameters, only for action: Stop
	Stop *StopAction `json:"stop,omitempty" protobuf:"bytes,4,opt,name=stop"`
}

// ResumeAction parameterizes a Resume action
type ResumeAction struct {
	// NodeFieldSelector selects suspended nodes to resume. Empty resumes the whole workflow.
	NodeFieldSelector string `json:"nodeFieldSelector,omitempty" protobuf:"bytes,1,opt,name=nodeFieldSelector"`
	// OutputParameters sets raw output parameters on the resumed suspend nodes
	OutputParameters map[string]string `json:"outputParameters,omitempty" protobuf:"bytes,2,rep,name=outputParameters"`
}

// StopAction parameterizes a Stop action
type StopAction struct {
	// Message set on stopped nodes
	Message string `json:"message,omitempty" protobuf:"bytes,1,opt,name=message"`
	// NodeFieldSelector selects suspended nodes to fail instead of stopping the whole workflow
	NodeFieldSelector string `json:"nodeFieldSelector,omitempty" protobuf:"bytes,2,opt,name=nodeFieldSelector"`
}

// WorkflowActionStatus is the observed outcome of the action
type WorkflowActionStatus struct {
	// Phase of the action: Pending, Succeeded or Failed. Empty means Pending.
	Phase WorkflowActionPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=WorkflowActionPhase"`
	// Reason is a machine-readable reason for a Failed phase
	Reason string `json:"reason,omitempty" protobuf:"bytes,2,opt,name=reason"`
	// Message is a human-readable elaboration of Reason
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// CompletionTime is when the action reached a terminal phase
	CompletionTime *metav1.Time `json:"completionTime,omitempty" protobuf:"bytes,4,opt,name=completionTime"`
}

// Fulfilled returns whether the action has reached a terminal phase
func (s WorkflowActionStatus) Fulfilled() bool {
	return s.Phase == WorkflowActionSucceeded || s.Phase == WorkflowActionFailed
}

// WorkflowActionList is a list of WorkflowAction resources.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowAction `json:"items" protobuf:"bytes,2,rep,name=items"`
}
