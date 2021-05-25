package git

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	ssh2 "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is the artifact driver for a git repo
type ArtifactDriver struct {
	Username              string
	Password              string
	SSHPrivateKey         string
	InsecureIgnoreHostKey bool
	DisableSubmodules     bool
}

var _ common.ArtifactDriver = &ArtifactDriver{}

var sshURLRegex = regexp.MustCompile("^(ssh://)?([^/:]*?)@[^@]+$")

func GetUser(url string) string {
	matches := sshURLRegex.FindStringSubmatch(url)
	if len(matches) > 2 {
		return matches[2]
	}
	// default to `git` user unless username is specified in SSH url
	return "git"
}

func (g *ArtifactDriver) auth(sshUser string) (func(), transport.AuthMethod, []string, error) {
	if g.SSHPrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(g.SSHPrivateKey))
		if err != nil {
			return nil, nil, nil, err
		}
		privateKeyFile, err := ioutil.TempFile("", "id_rsa.")
		if err != nil {
			return nil, nil, nil, err
		}
		err = ioutil.WriteFile(privateKeyFile.Name(), []byte(g.SSHPrivateKey), 0o600)
		if err != nil {
			return nil, nil, nil, err
		}
		auth := &ssh2.PublicKeys{User: sshUser, Signer: signer}
		if g.InsecureIgnoreHostKey {
			auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		}
		args := []string{"ssh", "-i", privateKeyFile.Name()}
		if g.InsecureIgnoreHostKey {
			args = append(args, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null")
		} else {
			args = append(args, "-o", "StrictHostKeyChecking=yes", "-o")
		}
		env := []string{"GIT_SSH_COMMAND=" + strings.Join(args, " ")}
		if g.InsecureIgnoreHostKey {
			auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
			env = append(env, "GIT_SSL_NO_VERIFY=true")
		}
		return func() { _ = os.Remove(privateKeyFile.Name()) },
			auth,
			env,
			nil
	}
	if g.Username != "" || g.Password != "" {
		filename := filepath.Join(os.TempDir(), "git-ask-pass.sh")
		_, err := os.Stat(filename)
		if os.IsNotExist(err) {
			err := ioutil.WriteFile(filename, []byte(`#!/bin/sh
case "$1" in
Username*) echo "${GIT_USERNAME}" ;;
Password*) echo "${GIT_PASSWORD}" ;;
esac
`), 0o755)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		return func() {},
			&http.BasicAuth{Username: g.Username, Password: g.Password},
			[]string{
				"GIT_ASKPASS=" + filename,
				"GIT_USERNAME=" + g.Username,
				"GIT_PASSWORD=" + g.Password,
			},
			nil
	}
	return func() {}, nil, nil, nil
}

// Save is unsupported for git output artifacts
func (g *ArtifactDriver) Save(string, *wfv1.Artifact) error {
	return errors.New("git output artifacts unsupported")
}

func (g *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	sshUser := GetUser(path)
	closer, auth, env, err := g.auth(sshUser)
	if err != nil {
		return err
	}
	defer closer()

	var recurseSubmodules = git.DefaultSubmoduleRecursionDepth
	if inputArtifact.Git.DisableSubmodules {
		log.Info("Recursive cloning of submodules is disabled")
		recurseSubmodules = git.NoRecurseSubmodules
	}
	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:               inputArtifact.Git.Repo,
		RecurseSubmodules: recurseSubmodules,
		Auth:              auth,
		Depth:             inputArtifact.Git.GetDepth(),
	})
	if err != nil {
		return err
	}
	if inputArtifact.Git.Fetch != nil {
		refSpecs := make([]config.RefSpec, len(inputArtifact.Git.Fetch))
		for i, spec := range inputArtifact.Git.Fetch {
			refSpecs[i] = config.RefSpec(spec)
		}
		fetchOptions := git.FetchOptions{
			Auth:     auth,
			RefSpecs: refSpecs,
			Depth:    inputArtifact.Git.GetDepth(),
		}
		err = fetchOptions.Validate()
		if err != nil {
			return err
		}
		err = repo.Fetch(&fetchOptions)
		if isAlreadyUpToDateErr(err) {
			return err
		}
	}
	if inputArtifact.Git.Revision != "" {
		// We still rely on forking git for checkout, since go-git does not have a reliable
		// way of resolving revisions (e.g. mybranch, HEAD^, v1.2.3)
		rev := getRevisionForCheckout(inputArtifact.Git.Revision)
		log.Info("Checking out revision ", rev)
		cmd := exec.Command("git", "checkout", rev)
		cmd.Dir = path
		cmd.Env = env
		output, err := cmd.Output()
		if err != nil {
			return g.error(err, cmd)
		}
		log.Infof("`%s` stdout:\n%s", cmd.Args, string(output))
		if !inputArtifact.Git.DisableSubmodules {
			submodulesCmd := exec.Command("git", "submodule", "update", "--init", "--recursive", "--force")
			submodulesCmd.Dir = path
			submodulesCmd.Env = env
			submoduleOutput, err := submodulesCmd.Output()
			if err != nil {
				return g.error(err, cmd)
			}
			log.Infof("`%s` stdout:\n%s", cmd.Args, string(submoduleOutput))
		}
	}
	return nil
}

// getRevisionForCheckout trims "refs/heads/" from the revision name (if present)
// so that `git checkout` will succeed.
func getRevisionForCheckout(revision string) string {
	return strings.TrimPrefix(revision, "refs/heads/")
}

func isAlreadyUpToDateErr(err error) bool {
	return err != nil && err.Error() != "already up-to-date"
}

func (g *ArtifactDriver) error(err error, cmd *exec.Cmd) error {
	if exErr, ok := err.(*exec.ExitError); ok {
		log.Errorf("`%s` stderr:\n%s", cmd.Args, string(exErr.Stderr))
		return errors.New(strings.Split(string(exErr.Stderr), "\n")[0])
	}
	return err
}

func (g *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
