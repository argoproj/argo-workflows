package git

import (
	"os"
	"testing"

	"k8s.io/client-go/util/homedir"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestGitArtifactDriver_Save(t *testing.T) {
	driver := &ArtifactDriver{}
	err := driver.Save("", nil)
	assert.Error(t, err)
}

func TestGitArtifactDriver_Load(t *testing.T) {
	t.Run("EmptyRepo", func(t *testing.T) {
		driver := &ArtifactDriver{}
		assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/empty-test-repo.git"}))
		assert.DirExists(t, path)
	})
	t.Run("PrivateRepo", func(t *testing.T) {
		t.Run("SSH", func(t *testing.T) {
			if os.Getenv("CI") == "true" {
				t.SkipNow()
			}
			privateKey, err := os.ReadFile(homedir.HomeDir() + "/.ssh/id_rsa")
			assert.NoError(t, err)
			driver := &ArtifactDriver{SSHPrivateKey: string(privateKey)}
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "git@github.com:argoproj-labs/private-test-repo.git"}))
			assert.FileExists(t, path+"/README.md")
		})
		t.Run("HTTPS", func(t *testing.T) {
			token := os.Getenv("PERSONAL_ACCESS_TOKEN")
			if token == "" {
				t.SkipNow()
			}
			driver := &ArtifactDriver{Username: "alexec", Password: token}
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/private-test-repo.git"}))
			assert.FileExists(t, path+"/README.md")
		})
	})
	t.Run("PublicRepo", func(t *testing.T) {
		driver := &ArtifactDriver{}
		assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git"}))
		assert.FileExists(t, path+"/README.md")
	})
	t.Run("Depth", func(t *testing.T) {
		driver := &ArtifactDriver{}
		var depth uint64 = 1
		assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Depth: &depth}))
		assert.FileExists(t, path+"/README.md")
	})
	t.Run("FetchRefs", func(t *testing.T) {
		driver := &ArtifactDriver{}
		t.Run("Garbage", func(t *testing.T) {
			assert.Error(t, load(driver, &wfv1.GitArtifact{
				Repo:  "https://github.com/argoproj-labs/test-repo.git",
				Fetch: []string{"garbage"},
			}))
		})
		t.Run("Valid", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{
				Repo:  "https://github.com/argoproj-labs/test-repo.git",
				Fetch: []string{"+refs/heads/*:refs/remotes/origin/*"},
			}))
			assert.FileExists(t, path+"/README.md")
		})
	})
	t.Run("Revision", func(t *testing.T) {
		driver := &ArtifactDriver{}
		t.Run("Garbage", func(t *testing.T) {
			assert.Error(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "garbage"}))
		})
		t.Run("Hash", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "6093d6a"}))
			assert.FileExists(t, path+"/README.md")
		})
		t.Run("HEAD", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "HEAD"}))
			assert.FileExists(t, path+"/README.md")
		})
		t.Run("HEAD~1", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "HEAD~1"}))
			assert.FileExists(t, path+"/README.md")
		})
		t.Run("Main", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "main"}))
			assert.FileExists(t, path+"/README.md")
		})
		t.Run("RemoteBranch", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "origin/my-branch"}))
			assert.FileExists(t, path+"/my-branch")
		})
		t.Run("LocalBranch", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "my-branch"}))
			assert.FileExists(t, path+"/my-branch")
		})
		t.Run("Tag", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo.git", Revision: "v0.0.0"}))
			assert.FileExists(t, path+"/README.md")
		})
	})
	t.Run("Submodules", func(t *testing.T) {
		driver := &ArtifactDriver{}
		t.Run("Disabled", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo-w-submodule.git", DisableSubmodules: true}))
			assert.FileExists(t, path+"/README.md")
		})
		t.Run("Enabled", func(t *testing.T) {
			assert.NoError(t, load(driver, &wfv1.GitArtifact{Repo: "https://github.com/argoproj-labs/test-repo-w-submodule.git"}))
			assert.FileExists(t, path+"/test-repo/README.md")
		})
	})
}

const path = "/tmp/repo"

func load(driver *ArtifactDriver, git *wfv1.GitArtifact) error {
	_ = os.RemoveAll(path)
	return driver.Load(&wfv1.Artifact{ArtifactLocation: wfv1.ArtifactLocation{Git: git}}, path)
}

func TestGetUser(t *testing.T) {
	for _, tt := range []struct {
		name string
		url  string
		user string
	}{
		{"Username in SSH url", "gitaly@github.com:argoproj/argo-workflows.git", "gitaly"},
		{"Default username", "https://github.com/argoproj/argo-workflows.git", "git"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sshUser := GetUser(tt.url)
			assert.Equal(t, sshUser, tt.user)
		})
	}
}
