package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowOp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowOpSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowOpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowOp `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type WorkflowOpSpec struct {
	Suspend  *SuspendOp  `json:"suspend,omitempty" protobuf:"varint,1,opt,name=suspend"`
	Resume   *ResumeOp   `json:"resume,omitempty" protobuf:"varint,2,opt,name=resume"`
	Shutdown *ShutdownOp `json:"shutdown,omitempty" protobuf:"bytes,3,opt,name=shutdown"`
}

type SuspendOp struct{}

type ResumeOp struct {
	NodeSelector string `protobuf:"bytes,1,opt,name=nodeSelector"`
}

type ShutdownOp struct {
	ShutdownStrategy ShutdownStrategy `json:"shutdownStrategy,omitempty" protobuf:"bytes,1,opt,name=shutdownStrategy,casttype=ShutdownStrategy"`
	NodeSelector     string           `protobuf:"bytes,4,opt,name=nodeSelector"`
	Message          string           `protobuf:"bytes,3,opt,name=message"`
}
