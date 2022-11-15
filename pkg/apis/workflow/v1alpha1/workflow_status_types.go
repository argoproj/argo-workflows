package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// WorkflowStatusResult is a used to communicate workflow status back to the controller.
// This is an internal type. Users should never create this resource directly, much like you would
// never create a ReplicaSet directly.
// +kubebuilder:resource:shortName=wfsr
// +genclient
// +genclient:noStatus
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the workflow"
// +kubebuilder:printcolumn:name="Age",type="date",format="date-time",JSONPath=".status.startedAt",description="When the workflow was started"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message",description="Human readable message indicating details about why the workflow is in this condition."

type WorkflowStatusResult struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	WorkflowStatus    WorkflowStatus `json:"workflowStatus" protobuf:"bytes,2,opt,name=workflowStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowStatusResultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowStatusResult `json:"items" protobuf:"bytes,2,rep,name=items"`
}
