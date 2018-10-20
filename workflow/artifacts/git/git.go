package git

import (
	"os/exec"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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
		signer, err := ssh.ParsePrivateKey([]byte(g.SSHPrivateKey))
		if err != nil {
			return errors.InternalWrapError(err)
		}
		auth := &ssh2.PublicKeys{User: "git", Signer: signer}
		auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
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
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	if auth != nil {
		cloneOptions.Auth = auth
	}
	repo, err := git.PlainClone(path, false, &cloneOptions)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	if inputArtifact.Git.Revision != "" {
		revParseCmd := exec.Command("git", "rev-parse", inputArtifact.Git.Revision)
		revParseCmd.Path = path
		gitRevision, err := revParseCmd.Output()
		if err != nil {
			return errors.InternalWrapError(err)
		}
		var parsedHash plumbing.Hash
		hash, err := repo.ResolveRevision(plumbing.Revision(gitRevision))
		parsedHash = *hash
		if err != nil {
			return errors.InternalWrapError(err)
		}
		w, err := repo.Worktree()
		if err != nil {
			return errors.InternalWrapError(err)
		}
		err = w.Checkout(&git.CheckoutOptions{
			Hash: parsedHash,
		})
		if err != nil {
			return errors.InternalWrapError(err)
		}
		subs, err := w.Submodules()
		if err != nil {
			return errors.InternalWrapError(err)
		}
		err = subs.Update(&git.SubmoduleUpdateOptions{
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			Auth:              auth,
		})
		if err != nil {
			return errors.InternalWrapError(err)
		}
	}
	return nil
}
