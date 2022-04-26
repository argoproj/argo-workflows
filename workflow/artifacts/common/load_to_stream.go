package common

import (
	"io"
	"os"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// wrapper around os.File enables us to remove the file when it gets closed
type selfDestructingFile struct {
	os.File
}

func (w selfDestructingFile) Close() error {
	err := w.File.Close()
	_ = os.Remove(w.Name())
	return err
}

// Use ArtifactDriver.Load() to get a stream, which we can use for all implementations of ArtifactDriver.OpenStream()
// that aren't yet implemented the "right way" and/or for those that don't have a natural way of streaming
func LoadToStream(a *wfv1.Artifact, g ArtifactDriver) (io.ReadCloser, error) {
	filename := "/tmp/" + rand.String(32)
	if err := g.Load(a, filename); err != nil {
		return nil, err
	}
	f, err := os.Open(filename)
	if err != nil {
		_ = os.Remove(filename)
		return nil, err
	}
	return &selfDestructingFile{*f}, nil
}
