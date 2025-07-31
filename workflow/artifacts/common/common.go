package common

import (
	"errors"
	"io"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *v1alpha1.Artifact, path string) error

	// OpenStream opens an artifact for reading. If the artifact is a file,
	// then the file should be opened. If the artifact is a directory, the
	// driver may return that as a tarball. OpenStream is intended to be efficient,
	// so implementations should minimise usage of disk, CPU and memory.
	// Implementations must not implement retry mechanisms. This will be handled by
	// the client, so would result in O(nm) cost.
	OpenStream(a *v1alpha1.Artifact) (io.ReadCloser, error)

	// Save uploads the path to artifact destination
	Save(path string, outputArtifact *v1alpha1.Artifact) error

	Delete(artifact *v1alpha1.Artifact) error

	ListObjects(artifact *v1alpha1.Artifact) ([]string, error)

	IsDirectory(artifact *v1alpha1.Artifact) (bool, error)
}

// ErrDeleteNotSupported Sentinel error definition for artifact deletion
var ErrDeleteNotSupported = errors.New("delete not supported for this artifact storage, please check" +
	" the following issue for details: https://github.com/argoproj/argo-workflows/issues/3102")
