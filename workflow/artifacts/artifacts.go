package executor

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *wfv1.Artifact, path string) error

	// Save uploads the path to artifact destination
	Save(path string, outputArtifact *wfv1.Artifact) error
}
