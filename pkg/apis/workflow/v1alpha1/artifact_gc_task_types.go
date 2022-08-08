package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowArtifactGCTask specifies the Artifacts that need to be deleted as well as the status of deletion
// +genclient
// +kubebuilder:resource:shortName=wfat
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
type WorkflowArtifactGCTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ArtifactGCSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            ArtifactGCStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// ArtifactGCSpec specifies the Artifacts that need to be deleted
type ArtifactGCSpec struct {
	// ArtifactsByNode maps Node name to information pertaining to Artifacts on that Node
	ArtifactsByNode map[string]ArtifactNodeSpec `json:"artifactsByNode,omitempty" protobuf:"bytes,1,rep,name=artifactsByNode"`
}

// ArtifactNodeSpec specifies the Artifacts that need to be deleted for a given Node
type ArtifactNodeSpec struct {
	// ArchiveLocation is the template-level Artifact location specification
	ArchiveLocation *ArtifactLocation `json:"archiveLocation,omitempty" protobuf:"bytes,1,opt,name=archiveLocation"`
	// Artifacts maps artifact name to Artifact description
	Artifacts map[string]Artifact `json:"artifacts,omitempty" protobuf:"bytes,2,rep,name=artifacts"`
}

// ArtifactGCStatus describes the result of the deletion
type ArtifactGCStatus struct {
	// ArtifactResultsByNode maps Node name to result
	ArtifactResultsByNode map[string]ArtifactResultNodeStatus `json:"artifactResultsByNode,omitempty" protobuf:"bytes,1,rep,name=artifactResultsByNode"`
}

// ArtifactResultNodeStatus describes the result of the deletion on a given node
type ArtifactResultNodeStatus struct {
	// ArtifactResults maps Artifact name to result of the deletion
	ArtifactResults map[string]ArtifactResult `json:"artifactResults,omitempty" protobuf:"bytes,1,rep,name=artifactResults"`
}

// ArtifactResult describes the result of attempting to delete a given Artifact
type ArtifactResult struct {
	// Name is the name of the Artifact
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Success describes whether the deletion succeeded
	Success bool `json:"success,omitempty" protobuf:"varint,2,opt,name=success"`

	// Error is an optional error message which should be set if Success==false
	Error *string `json:"error,omitempty" protobuf:"bytes,3,opt,name=error"`
}

// WorkflowArtifactGCTaskList is list of WorkflowArtifactGCTask resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowArtifactGCTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []WorkflowArtifactGCTask `json:"items" protobuf:"bytes,2,opt,name=items"`
}
