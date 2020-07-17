package executor

import (
	"io"
	"io/ioutil"
	"os"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// StreamingDriver will be more memory and disk efficient when it can be used.
type StreamingDriver interface {
	ArtifactDriver
	// Get returns a stream form the artifact. It cannot support directories like `Save` does.
	Get(art *wfv1.Artifact) (io.Reader, error)
	// Put writes a stream to the artifact. It cannot support directories like `Load` does.
	// objectSize can be -1 if you don't know the size, but performance maybe impacted.
	Put(reader io.Reader, objectSize int64, art *wfv1.Artifact) error
}

type adapter struct {
	ArtifactDriver
}

func (a adapter) Get(art *wfv1.Artifact) (io.Reader, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	err = a.Load(art, file.Name())
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return os.Open(file.Name())
}

func (a adapter) Put(reader io.Reader, _ int64, art *wfv1.Artifact) error {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	open, err := os.Open(file.Name())
	if err != nil {
		return err
	}
	_, err = io.Copy(open, reader)
	if err != nil {
		return err
	}
	err = a.Save(file.Name(), art)
	if err != nil {
		return err
	}
	return nil
}

// Return either the driver (if already a StreamingDriver) or an adapter.
// This is a convenience function for when you want to use `Get` or `Put` and never write to a file.
// If you want to write to a file, do NOT use, as it will 2x disk and memory usage.
func NewStreamingDriver(driver ArtifactDriver) StreamingDriver {
	if streamingDriver, ok := driver.(StreamingDriver); ok {
		return streamingDriver
	}
	return &adapter{driver}
}
