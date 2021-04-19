package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=wfTaskSet
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTaskSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowTaskSetSpec    `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            *WorkflowTaskSetStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type WorkflowTaskSets []WorkflowTaskSet

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTaskSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           WorkflowTaskSets `json:"items" protobuf:"bytes,2,opt,name=items"`
}

type WorkflowTaskSetSpec struct {
	// Key NodeID, value: Template
	Templates []Task `json:"templates,omitempty" protobuf:"bytes,1,rep,name=templates"`
}

type Task struct {
	NodeID   string   `json:"nodeId" protobuf:"bytes,1,opt,name=nodeId"`
	Template Template `json:"template" protobuf:"bytes,2,opt,name=template"`
}

type WorkflowTaskSetStatus struct {
	Nodes map[string]TaskResult `json:"nodes,omitempty" protobuf:"bytes,1,rep,name=nodes"`
}

type TaskResult struct {
	Phase   NodePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=NodePhase"`
	Message string    `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	Outputs *Outputs  `json:"outputs,omitempty" protobuf:"bytes,3,opt,name=outputs"`
}
