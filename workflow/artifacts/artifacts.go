package executor

import (
	wfv1 "github.com/argoproj/argo/api/workflow/v1"
)

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *wfv1.Artifact, path string) error

	// Save uploads the path to a destination URL
	Save(path string, destURL string) (string, error)
}
