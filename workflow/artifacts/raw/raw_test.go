package raw_test

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/raw"
)

const (
	LoadFileName string = "argo_raw_artifact_test_load.txt"
)

func TestLoad(t *testing.T) {
	content := fmt.Sprintf("time: %v", time.Now().UnixNano())
	lf, err := os.CreateTemp("", LoadFileName)
	assert.NoError(t, err)
	defer os.Remove(lf.Name())

	art := &wfv1.Artifact{}
	art.Raw = &wfv1.RawArtifact{
		Data: content,
	}
	driver := &raw.ArtifactDriver{}
	err = driver.Load(art, lf.Name())
	assert.NoError(t, err)

	dat, err := os.ReadFile(lf.Name())
	assert.NoError(t, err)
	assert.Equal(t, content, string(dat))
}

func TestOpenStream(t *testing.T) {
	content := fmt.Sprintf("time: %v", time.Now().UnixNano())
	art := &wfv1.Artifact{}
	art.Raw = &wfv1.RawArtifact{
		Data: content,
	}
	driver := &raw.ArtifactDriver{}
	rc, err := driver.OpenStream(art)
	assert.NoError(t, err)
	defer rc.Close()

	dat, err := io.ReadAll(rc)
	assert.NoError(t, err)
	assert.Equal(t, content, string(dat))
}
