package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowEvent is the definition of an event resource
// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=wfev
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowEventSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// WorkflowEventList is list of event resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowEvent `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// Event can trigger this workflow template.
type WorkflowEventSpec struct {
	// Expression (https://github.com/antonmedv/expr) that we must must match the event. E.g. `payload.message == "test"`
	// +kubebuilder:validation:MinLength=4
	Expression string `json:"expression" protobuf:"bytes,1,opt,name=expression"`

	// Parameters extracted from the event and then set as arguments to the workflow created.
	Parameters []Parameter `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`

	// WorkflowTemplateRef the workflow template to submit when we match the event
	WorkflowTemplateRef corev1.LocalObjectReference `json:"workflowTemplateRef" protobuf:"bytes,3,opt,name=workflowTemplateRef"`
}
