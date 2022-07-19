package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:resource:shortName=wfts
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
type ArtifactGCWork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ArtifactGCWorkSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            ArtifactGCSWorktatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type ArtifactGCWorkSpec struct {
	ArtifactsByNode map[string]ArtifactNodeSpec `json:"artifactsByNode,omitempty" protobuf:"bytes,1,rep,name=artifactsByNode"`
}

type ArtifactNodeSpec struct {
	ArchiveLocation *ArtifactLocation    `json:"archiveLocation,omitempty" protobuf:"bytes,1,opt,name=archiveLocation"`
	Artifacts       map[string]Artifacts `json:"artifacts,omitempty" protobuf:"bytes,2,rep,name=artifacts"`
}

type ArtifactGCSWorktatus struct {
	ArtifactResultsByNode map[string]ArtifactResultNodeStatus `json:"artifactResultsByNode,omitempty" protobuf:"bytes,1,rep,name=artifactResultsByNode"`
}

type ArtifactResultNodeStatus struct {
	ArtifactResults []ArtifactResult `json:"artifactResults,omitempty" protobuf:"bytes,1,rep,name=artifactResults"`
}

type ArtifactResult struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	Deleted bool `json:"deleted,omitempty" protobuf:"varint,2,opt,name=deleted"`

	Error *string `json:"error,omitempty" protobuf:"bytes,3,opt,name=error"`
}
