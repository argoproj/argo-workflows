package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowEventBinding is the definition of an event resource
// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=wfeb
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowEventBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowEventBindingSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// WorkflowEventBindingList is list of event resources
// +kubebuilder:resource:shortName=wfebs
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowEventBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowEventBinding `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type WorkflowEventBindingSpec struct {
	// Event is the event to bind to
	Event Event `json:"event" protobuf:"bytes,1,opt,name=event"`
	// Submit is the workflow template to submit
	Submit *Submit `json:"submit,omitempty" protobuf:"bytes,2,opt,name=submit"`
}

type Event struct {
	// Selector (https://github.com/antonmedv/expr) that we must must match the event. E.g. `payload.message == "test"`
	Selector string `json:"selector" protobuf:"bytes,1,opt,name=selector"`
}

type Submit struct {
	// WorkflowTemplateRef the workflow template to submit
	WorkflowTemplateRef WorkflowTemplateRef `json:"workflowTemplateRef" protobuf:"bytes,1,opt,name=workflowTemplateRef"`

	// Metadata optional means to customize select fields of the workflow metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,3,opt,name=metadata"`

	// Arguments extracted from the event and then set as arguments to the workflow created.
	Arguments *Arguments `json:"arguments,omitempty" protobuf:"bytes,2,opt,name=arguments"`
}
