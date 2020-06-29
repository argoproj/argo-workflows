package git

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var d = uint64(1)

func TestGitArtifactDriver_Load(t *testing.T) {
	_ = os.Remove("git-ask-pass.sh")
	driver := &GitArtifactDriver{}
	path := "/tmp/git-found"
	assert.NoError(t, os.RemoveAll(path))
	assert.NoError(t, os.MkdirAll(path, 0777))
	err := driver.Load(&wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{
			Git: &wfv1.GitArtifact{
				Repo:     "https://github.com/argoproj/argoproj.git",
				Fetch:    []string{"+refs/heads/*:refs/remotes/origin/*"},
				Revision: "HEAD",
				Depth:    &d,
			},
		},
	}, path)
	if assert.NoError(t, err) {
		_, err := os.Stat(path)
		assert.NoError(t, err)
	}
}

func TestGitArtifactDriver_Save(t *testing.T) {
	driver := &GitArtifactDriver{}
	err := driver.Save("", nil)
	assert.Error(t, err)
}

func TestGitArtifactDriverLoad_HTTPS(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("not running an GITHUB_TOKEN not set")
	}
	_ = os.Remove("git-ask-pass.sh")
	tmp, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	driver := &GitArtifactDriver{Username: os.Getenv("GITHUB_TOKEN")}
	assert.NotEmpty(t, driver.Username)
	err = driver.Load(&wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{
			Git: &wfv1.GitArtifact{
				Repo:     "https://github.com/argoproj/argo.git",
				Fetch:    []string{"+refs/heads/*:refs/remotes/origin/*"},
				Revision: "HEAD",
				Depth:    &d,
			},
		},
	}, tmp)
	assert.NoError(t, err)
	println(tmp)
}

func TestGitArtifactDriverLoad_SSL(t *testing.T) {
	for _, tt := range []struct {
		name     string
		insecure bool
	}{
		{"Insecure", true},
		{"Secure", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove("git-ask-pass.sh")
			key := os.Getenv("HOME") + "/.ssh/id_rsa"
			data, err := ioutil.ReadFile(key)
			if err != nil && os.IsNotExist(err) {
				t.Skip(key + " does not exist")
			}
			assert.NoError(t, err)
			tmp, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			println(tmp)
			driver := &GitArtifactDriver{SSHPrivateKey: string(data)}
			err = driver.Load(&wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					Git: &wfv1.GitArtifact{
						Repo:                  "git@github.com:argoproj/argo.git",
						Fetch:                 []string{"+refs/heads/*:refs/remotes/origin/*"},
						Revision:              "HEAD",
						InsecureIgnoreHostKey: tt.insecure,
						Depth:                 &d,
					},
				},
			}, tmp)
			assert.NoError(t, err)
		})
	}
}
