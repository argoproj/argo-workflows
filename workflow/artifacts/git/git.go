package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"golang.org/x/crypto/ssh"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	ssh2 "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// GitArtifactDriver is the artifact driver for a git repo
type GitArtifactDriver struct {
	Username      string
	Password      string
	SSHPrivateKey string
}

// Load download artifacts from an git URL
// Credentials are temporarily stored in a git-credentials file during the clone
// and deleted before returning. This is to prevent credentials from inadvertently
// leaking such as in the repo_dir/.git/config or logging an insecure url.
func (g *GitArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	if g.SSHPrivateKey != "" {
		tmpfile, err := ioutil.TempFile("", "ssh-know-hosts")
		if err != nil {
			fmt.Println(err)
		}
		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()
		re := regexp.MustCompile("@(.*):")
		repoHost := re.FindStringSubmatch(inputArtifact.Git.Repo)[1]
		os.Setenv("SSH_KNOWN_HOSTS", tmpfile.Name())
		err = common.RunCommand("sh", "-c", fmt.Sprintf("ssh-keyscan %s > %s", repoHost, tmpfile.Name()))
		signer, _ := ssh.ParsePrivateKey([]byte(g.SSHPrivateKey))
		auth := &ssh2.PublicKeys{User: "git", Signer: signer}

		return gitClone(path, inputArtifact, auth)

	}
	if g.Username != "" || g.Password != "" {
		auth := &http.BasicAuth{Username: g.Username, Password: g.Password}
		return gitClone(path, inputArtifact, auth)
	}
	return gitClone(path, inputArtifact, nil)
}

// Save is unsupported for git output artifacts
func (g *GitArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Git output artifacts unsupported")
}

func gitClone(path string, inputArtifact *wfv1.Artifact, auth transport.AuthMethod) error {
	cloneOptions := git.CloneOptions{
		URL:               inputArtifact.Git.Repo,
		Depth:             1,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	if auth != nil {
		cloneOptions = git.CloneOptions{
			URL:               inputArtifact.Git.Repo,
			Depth:             1,
			Auth:              auth,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		}
	}
	repo, err := git.PlainClone(path, false, &cloneOptions)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	if inputArtifact.Git.Revision != "" {
		w, err := repo.Worktree()
		if err != nil {
			return errors.InternalWrapError(err)
		}
		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(inputArtifact.Git.Revision),
		})
		if err != nil {
			return errors.InternalWrapError(err)
		}
	}
	return nil
}
