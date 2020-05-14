package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestGitArtifactDriver_Load(t *testing.T) {
	driver := &GitArtifactDriver{}
	t.Run("Found", func(t *testing.T) {
		path := "/tmp/git-found"
		assert.NoError(t, os.RemoveAll(path))
		assert.NoError(t, os.MkdirAll(path, 0777))
		d := uint64(1)
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Git: &wfv1.GitArtifact{Repo: "https://github.com/argoproj/argoproj.git", Revision: "master", Depth: &d},
			},
		}, path)
		if assert.NoError(t, err) {
			_, err := os.Stat(path)
			assert.NoError(t, err)
		}
	})
}

func TestGitArtifactDriver_Save(t *testing.T) {
	driver := &GitArtifactDriver{}
	err := driver.Save("", nil)
	assert.Error(t, err)
}
