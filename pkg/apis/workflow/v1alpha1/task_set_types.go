package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTaskSet holds a set of tasks to be executed as part of a workflow.
// +genclient
// +kubebuilder:resource:shortName=wfts
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
type WorkflowTaskSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowTaskSetSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            WorkflowTaskSetStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type WorkflowTaskSetSpec struct {
	Tasks map[string]Template `json:"tasks,omitempty" protobuf:"bytes,1,rep,name=tasks"`
}

type WorkflowTaskSetStatus struct {
	Nodes map[string]NodeResult `json:"nodes,omitempty" protobuf:"bytes,1,rep,name=nodes"`
}

// WorkflowTaskSetList is a list of WorkflowTaskSet resources.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTaskSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowTaskSet `json:"items" protobuf:"bytes,2,opt,name=items"`
}

type NodeResult struct {
	Phase    NodePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=NodePhase"`
	Message  string    `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	Outputs  *Outputs  `json:"outputs,omitempty" protobuf:"bytes,3,opt,name=outputs"`
	Progress Progress  `json:"progress,omitempty" protobuf:"bytes,4,opt,name=progress,casttype=Progress"`
}
