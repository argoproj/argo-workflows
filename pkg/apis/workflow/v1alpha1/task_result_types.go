package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTaskResult is a used to communicate a result back to the controller. Unlike WorkflowTaskSet, it has
// more capacity. This is an internal type. Users should never create this resource directly, much like you would
// never create a ReplicaSet directly.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTaskResult struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	NodeResult        `json:",inline" protobuf:"bytes,2,opt,name=nodeResult"`
}

// WorkflowTaskResultList is a list of WorkflowTaskResult resources.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTaskResultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowTaskResult `json:"items" protobuf:"bytes,2,rep,name=items"`
}
