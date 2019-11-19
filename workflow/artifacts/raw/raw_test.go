package raw_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/artifacts/raw"
)

const (
	LoadFileName string = "argo_raw_artifact_test_load.txt"
)

func TestLoad(t *testing.T) {

	content := "time: " + string(time.Now().UnixNano())
	lf, err := ioutil.TempFile("", LoadFileName)
	assert.NoError(t, err)
	defer os.Remove(lf.Name())

	art := &wfv1.Artifact{}
	art.Raw = &wfv1.RawArtifact{
		Data: content,
	}
	driver := &raw.RawArtifactDriver{}
	err = driver.Load(art, lf.Name())
	assert.NoError(t, err)

	dat, err := ioutil.ReadFile(lf.Name())
	assert.NoError(t, err)
	assert.Equal(t, content, string(dat))

}
