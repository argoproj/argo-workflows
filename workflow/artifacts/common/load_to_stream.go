package common

import (
	"context"
	"io"
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/util/rand"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

const loadToStreamPrefix = `wfstream-`

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
func LoadToStream(ctx context.Context, a *wfv1.Artifact, g ArtifactDriver) (io.ReadCloser, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Infof(ctx, "Efficient artifact streaming is not supported for type %v: see https://github.com/argoproj/argo-workflows/issues/8489",
		reflect.TypeOf(g))
	filename := "/tmp/" + loadToStreamPrefix + rand.String(32)
	if err := g.Load(ctx, a, filename); err != nil {
		return nil, err
	}
	f, err := os.Open(filename)
	if err != nil {
		_ = os.Remove(filename)
		return nil, err
	}
	return &selfDestructingFile{*f}, nil
}
