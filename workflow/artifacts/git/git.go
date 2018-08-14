package git

import (
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	ssh2 "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// GitArtifactDriver is the artifact driver for a git repo
type GitArtifactDriver struct {
	Username      string
	Password      string
	SSHPrivateKey string
}

// Load download artifacts from an git URL
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
		Auth:              auth,
	}
	_, err := git.PlainClone(path, false, &cloneOptions)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	if inputArtifact.Git.Revision != "" {
		// We still rely on forking git for checkout, since go-git does not have a reliable
		// way of resolving revisions (e.g. mybranch, HEAD^, v1.2.3)
		log.Infof("Checking out revision %s", inputArtifact.Git.Revision)
		cmd := exec.Command("git", "checkout", inputArtifact.Git.Revision)
		cmd.Dir = path
		output, err := cmd.Output()
		if err != nil {
			if exErr, ok := err.(*exec.ExitError); ok {
				log.Errorf("`%s` stderr:\n%s", cmd.Args, string(exErr.Stderr))
				return errors.InternalError(strings.Split(string(exErr.Stderr), "\n")[0])
			}
			return errors.InternalWrapError(err)
		}
		log.Errorf("`%s` stdout:\n%s", cmd.Args, string(output))
	}
	return nil
}
