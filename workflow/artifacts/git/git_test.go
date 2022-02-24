package git

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var d = uint64(1)

func TestGitArtifactDriver_Save(t *testing.T) {
	driver := &ArtifactDriver{}
	err := driver.Save("", nil)
	assert.Error(t, err)
}

func TestGitArtifactDriverLoad_HTTPS(t *testing.T) {
	for _, tt := range []struct {
		url string
	}{
		{"https://github.com/argoproj/empty.git"},
	} {
		if os.Getenv("GITHUB_TOKEN") == "" {
			t.Skip("not running an GITHUB_TOKEN not set")
		}
		_ = os.Remove("git-ask-pass.sh")
		tmp := t.TempDir()
		driver := &ArtifactDriver{Username: os.Getenv("GITHUB_TOKEN")}
		assert.NotEmpty(t, driver.Username)
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Git: &wfv1.GitArtifact{
					Repo:     tt.url,
					Fetch:    []string{"+refs/heads/*:refs/remotes/origin/*"},
					Revision: "HEAD",
					Depth:    &d,
				},
			},
		}, tmp)
		assert.NoError(t, err)
		println(tmp)
	}
}

func TestGitArtifactDriverLoad_SSL(t *testing.T) {
	t.SkipNow()
	for _, tt := range []struct {
		name     string
		insecure bool
		url      string
	}{
		{"Insecure", true, "https://github.com/argoproj/argo-workflows.git"},
		{"Secure", false, "https://github.com/argoproj/argo-workflows.git"},
		{"Insecure", true, "https://github.com/argoproj/empty.git"},
		{"Secure", false, "https://github.com/argoproj/empty.git"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove("git-ask-pass.sh")
			key := os.Getenv("HOME") + "/.ssh/id_rsa"
			data, err := ioutil.ReadFile(key)
			if err != nil && os.IsNotExist(err) {
				t.Skip(key + " does not exist")
			}
			assert.NoError(t, err)
			tmp := t.TempDir()
			println(tmp)
			driver := &ArtifactDriver{SSHPrivateKey: string(data)}
			err = driver.Load(&wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					Git: &wfv1.GitArtifact{
						Repo:                  tt.url,
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

func TestGetCheckoutRevision(t *testing.T) {
	for _, tt := range []struct {
		in       string
		expected string
	}{
		{"my-branch", "my-branch"},
		{"refs/heads/my-branch", "my-branch"},
		{"refs/tags/1.0.0", "refs/tags/1.0.0"},
		{"ae7b5432cfa15577d4740fb047762254be3652db", "ae7b5432cfa15577d4740fb047762254be3652db"},
	} {
		t.Run(tt.in, func(t *testing.T) {
			result := getRevisionForCheckout(tt.in)
			assert.Equal(t, result, tt.expected)
		})
	}
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
