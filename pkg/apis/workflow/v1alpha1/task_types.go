package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// pro (very minor): we can use print columns to display outcomes

// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Template",type="string",JSONPath=".spec.name",description="Template"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status"
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              *Template   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            *NodeResult `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowTask `json:"items" protobuf:"bytes,2,opt,name=items"`
}
