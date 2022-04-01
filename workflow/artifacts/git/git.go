package git

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	ssh2 "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

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

func (g *ArtifactDriver) auth(sshUser string) (func(), transport.AuthMethod, error) {
	if g.SSHPrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(g.SSHPrivateKey))
		if err != nil {
			return nil, nil, err
		}
		privateKeyFile, err := ioutil.TempFile("", "id_rsa.")
		if err != nil {
			return nil, nil, err
		}
		err = ioutil.WriteFile(privateKeyFile.Name(), []byte(g.SSHPrivateKey), 0o600)
		if err != nil {
			return nil, nil, err
		}
		auth := &ssh2.PublicKeys{User: sshUser, Signer: signer}
		if g.InsecureIgnoreHostKey {
			auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		}
		if g.InsecureIgnoreHostKey {
			auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		}
		return func() { _ = os.Remove(privateKeyFile.Name()) },
			auth,
			nil
	}
	if g.Username != "" || g.Password != "" {
		filename := filepath.Join(os.TempDir(), "git-ask-pass.sh")
		_, err := os.Stat(filename)
		if os.IsNotExist(err) {
			//nolint:gosec
			err := ioutil.WriteFile(filename, []byte(`#!/bin/sh
case "$1" in
Username*) echo "${GIT_USERNAME}" ;;
Password*) echo "${GIT_PASSWORD}" ;;
esac
`), 0o755)
			if err != nil {
				return nil, nil, err
			}
		}
		return func() {},
			&http.BasicAuth{Username: g.Username, Password: g.Password},
			nil
	}
	return func() {}, nil, nil
}

// Save is unsupported for git output artifacts
func (g *ArtifactDriver) Save(string, *wfv1.Artifact) error {
	return errors.New("git output artifacts unsupported")
}

func (g *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	sshUser := GetUser(inputArtifact.Git.Repo)
	closer, auth, err := g.auth(sshUser)
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
	switch err {
	case transport.ErrEmptyRemoteRepository:
		log.Info("Cloned an empty repository ")
		r, err := git.PlainInit(path, false)
		if err != nil {
			return err
		}
		if _, err := r.CreateRemote(&config.RemoteConfig{Name: git.DefaultRemoteName, URLs: []string{inputArtifact.Git.Repo}}); err != nil {
			return err
		}
		branchName := inputArtifact.Git.Revision
		if branchName == "" {
			branchName = "master"
		}
		if err = r.CreateBranch(&config.Branch{Name: branchName, Remote: git.DefaultRemoteName, Merge: plumbing.Master}); err != nil {
			return err
		}
		return nil
	default:
		return err
	case nil:
		// fallthrough ...
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
		h, err := repo.ResolveRevision(plumbing.Revision(inputArtifact.Git.Revision))
		if err != nil {
			return err
		}
		w, err := repo.Worktree()
		if err != nil {
			return err
		}
		if err := w.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(h.String())}); err != nil {
			return err
		}
		if !inputArtifact.Git.DisableSubmodules {
			s, err := w.Submodules()
			if err != nil {
				return err
			}
			if err := s.Update(&git.SubmoduleUpdateOptions{
				Init:              true,
				RecurseSubmodules: recurseSubmodules,
				Auth:              auth,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func isAlreadyUpToDateErr(err error) bool {
	return err != nil && err.Error() != "already up-to-date"
}

func (g *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
