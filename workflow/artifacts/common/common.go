package common

import "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *v1alpha1.Artifact, path string) error

	// Save uploads the path to artifact destination
	Save(path string, outputArtifact *v1alpha1.Artifact) error

	ListObjects(artifact *v1alpha1.Artifact) ([]string, error)
}
