package git

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
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
		sshKeyFile, err := ioutil.TempFile("", "ssh-key-")
		if err != nil {
			return errors.InternalWrapError(err)
		}
		defer func() {
			_ = os.Remove(sshKeyFile.Name())
		}()
		content := []byte(g.SSHPrivateKey)
		if _, err := sshKeyFile.Write(content); err != nil {
			return errors.InternalWrapError(err)
		}
		if err := sshKeyFile.Close(); err != nil {
			return errors.InternalWrapError(err)
		}
		err = common.RunCommand("ssh-add", sshKeyFile.Name())
		if err != nil {
			return err
		}
		re := regexp.MustCompile("@(.*):")
		repoHost := re.FindStringSubmatch(inputArtifact.Git.Repo)
		err = common.RunCommand("mkdir", "~/.ssh")
		if err != nil {
			return err
		}
		err = common.RunCommand("ssh-keyscan", fmt.Sprintf("%s", repoHost), ">", "~/.ssh/know_hosts")
		if err != nil {
			return err
		}

		return gitClone(path, inputArtifact)
	}
	if g.Username != "" || g.Password != "" {
		// Formulate an insecure repo URL which incorporates the credentials which
		// we temporarily store it to a git-credentials file during the clone.
		insecureURL, err := url.Parse(inputArtifact.Git.Repo)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		insecureURL.User = url.UserPassword(g.Username, g.Password)
		tmpfile, err := ioutil.TempFile("", "git-cred-")
		if err != nil {
			return errors.InternalWrapError(err)
		}
		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()
		content := []byte(insecureURL.String() + "\n")
		if _, err := tmpfile.Write(content); err != nil {
			return errors.InternalWrapError(err)
		}
		if err := tmpfile.Close(); err != nil {
			return errors.InternalWrapError(err)
		}
		err = common.RunCommand("git", "config", "--global", "credential.helper", fmt.Sprintf("store --file=%s", tmpfile.Name()))
		if err != nil {
			return err
		}
		defer func() {
			_ = common.RunCommand("git", "config", "--global", "--remove-section", "credential")
		}()
	}
	return gitClone(path, inputArtifact)
}

// Save is unsupported for git output artifacts
func (g *GitArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Git output artifacts unsupported")
}

func gitClone(path string, inputArtifact *wfv1.Artifact) error {
	err := common.RunCommand("git", "clone", inputArtifact.Git.Repo, path)
	if err != nil {
		lines := strings.Split(err.Error(), "\n")
		if len(lines) > 1 {
			// give only the last, most-useful error line from git
			return errors.New(errors.CodeBadRequest, lines[len(lines)-1])
		}
		return err
	}
	if inputArtifact.Git.Revision != "" {
		err := common.RunCommand("git", "-C", path, "checkout", inputArtifact.Git.Revision)
		if err != nil {
			return err
		}
	}
	return nil
}
