package volume_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/artifacts/volume"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	tmpdir := os.TempDir()
	defer func() {
		_ = os.RemoveAll(tmpdir)
	}()

	err := os.Mkdir(filepath.Join(tmpdir, "foo"), 0777)
	assert.NoError(t, err)
	srcf, err := os.Create(filepath.Join(tmpdir, "foo", "bar"))
	assert.NoError(t, err)
	content := []byte("time: " + string(time.Now().UnixNano()))
	_, err = srcf.Write(content)
	assert.NoError(t, err)

	dstf, err := ioutil.TempFile("", "dst")
	assert.NoError(t, err)
	defer util.Close(dstf)

	art := &wfv1.Artifact{}
	art.Volume = &wfv1.VolumeArtifact{
		Name:    "foo",
		SubPath: "bar",
	}
	driver := &volume.ArtifactDriver{
		MountPath: tmpdir,
	}
	err = driver.Load(art, dstf.Name())
	assert.NoError(t, err)

	dat, err := ioutil.ReadFile(dstf.Name())
	assert.NoError(t, err)
	assert.Equal(t, content, dat)
}
func TestSave(t *testing.T) {
	tmpdir := os.TempDir()
	defer func() {
		_ = os.RemoveAll(tmpdir)
	}()

	err := os.Mkdir(filepath.Join(tmpdir, "foo"), 0777)
	assert.NoError(t, err)
	dstf, err := os.Create(filepath.Join(tmpdir, "foo", "bar"))
	assert.NoError(t, err)

	srcf, err := ioutil.TempFile("", "dst")
	assert.NoError(t, err)
	defer util.Close(srcf)
	content := []byte("time: " + string(time.Now().UnixNano()))
	_, err = srcf.Write(content)
	assert.NoError(t, err)

	art := &wfv1.Artifact{}
	art.Volume = &wfv1.VolumeArtifact{
		Name:    "foo",
		SubPath: "bar",
	}
	driver := &volume.ArtifactDriver{
		MountPath: tmpdir,
	}
	err = driver.Save(srcf.Name(), art)
	assert.NoError(t, err)

	dat, err := ioutil.ReadFile(dstf.Name())
	assert.NoError(t, err)
	assert.Equal(t, content, dat)
}
