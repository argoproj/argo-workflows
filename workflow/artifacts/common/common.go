package common

import (
	"errors"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var ErrDeleteNotSupported = errors.New("delete is not supported for this executor, please vote for support here https://github.com/argoproj/argo-workflows/issues/3102")

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *v1alpha1.Artifact, path string) error

	// Save uploads the path to artifact destination
	Save(path string, outputArtifact *v1alpha1.Artifact) error

	ListObjects(artifact *v1alpha1.Artifact) ([]string, error)

	// Delete deletes the object. It should be idempotent, if the object is already deleted, that must not return an error.
	// ErrDeleteNotSupported should be return when not supported.
	Delete(a v1alpha1.Artifact) error
}
