package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:resource:shortName=agw
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
type ArtifactGCTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ArtifactGCSpec    `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            ArtifactGCSStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type ArtifactGCSpec struct {
	ArtifactsByNode map[string]ArtifactNodeSpec `json:"artifactsByNode,omitempty" protobuf:"bytes,1,rep,name=artifactsByNode"`
}

type ArtifactNodeSpec struct {
	ArchiveLocation *ArtifactLocation   `json:"archiveLocation,omitempty" protobuf:"bytes,1,opt,name=archiveLocation"`
	Artifacts       map[string]Artifact `json:"artifacts,omitempty" protobuf:"bytes,2,rep,name=artifacts"`
}

type ArtifactGCSStatus struct {
	ArtifactResultsByNode map[string]ArtifactResultNodeStatus `json:"artifactResultsByNode,omitempty" protobuf:"bytes,1,rep,name=artifactResultsByNode"`
}

type ArtifactResultNodeStatus struct {
	ArtifactResults map[string]ArtifactResult `json:"artifactResults,omitempty" protobuf:"bytes,1,rep,name=artifactResults"`
}

type ArtifactResult struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	Success bool `json:"success,omitempty" protobuf:"varint,2,opt,name=success"`

	Error *string `json:"error,omitempty" protobuf:"bytes,3,opt,name=error"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ArtifactGCTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ArtifactGCTask `json:"items" protobuf:"bytes,2,opt,name=items"`
}
