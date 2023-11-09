package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	ssh2 "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"
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
		privateKeyFile, err := os.CreateTemp("", "id_rsa.")
		if err != nil {
			return nil, nil, err
		}
		err = os.WriteFile(privateKeyFile.Name(), []byte(g.SSHPrivateKey), 0o600)
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
		return func() { _ = os.Remove(privateKeyFile.Name()) }, auth, nil
	}
	if g.Username != "" || g.Password != "" {
		return func() {}, &http.BasicAuth{Username: g.Username, Password: g.Password}, nil
	}
	return func() {}, nil, nil
}

// Save is unsupported for git output artifacts
func (g *ArtifactDriver) Save(string, *wfv1.Artifact) error {
	return errors.New("git output artifacts unsupported")
}

// Delete is unsupported for git artifacts
func (g *ArtifactDriver) Delete(s *wfv1.Artifact) error {
	return common.ErrDeleteNotSupported
}

func (g *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	a := inputArtifact.Git
	sshUser := GetUser(a.Repo)
	closer, auth, err := g.auth(sshUser)
	if err != nil {
		return err
	}
	defer closer()
	depth := a.GetDepth()
	cloneOptions := &git.CloneOptions{
		URL:          a.Repo,
		Auth:         auth,
		Depth:        depth,
		SingleBranch: a.SingleBranch,
	}
	if a.SingleBranch && a.Branch == "" {
		return errors.New("single branch mode without a branch specified")
	}
	if a.SingleBranch {
		cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(a.Branch)
	}

	r, err := git.PlainClone(path, false, cloneOptions)
	switch err {
	case transport.ErrEmptyRemoteRepository:
		log.Info("Cloned an empty repository")
		r, err := git.PlainInit(path, false)
		if err != nil {
			return fmt.Errorf("failed to plain init: %w", err)
		}
		if _, err := r.CreateRemote(&config.RemoteConfig{Name: git.DefaultRemoteName, URLs: []string{a.Repo}}); err != nil {
			return fmt.Errorf("failed to create remote %q: %w", a.Repo, err)
		}
		branchName := a.Revision
		if branchName == "" {
			branchName = "master"
		}
		if err = r.CreateBranch(&config.Branch{Name: branchName, Remote: git.DefaultRemoteName, Merge: plumbing.Master}); err != nil {
			return fmt.Errorf("failed to create branch %q: %w", branchName, err)
		}
		return nil
	case nil:
		// fallthrough ...
	default:
		return fmt.Errorf("failed to clone %q: %w", a.Repo, err)
	}
	if len(a.Fetch) > 0 {
		refSpecs := make([]config.RefSpec, len(a.Fetch))
		for i, spec := range a.Fetch {
			refSpecs[i] = config.RefSpec(spec)
		}
		opts := &git.FetchOptions{Auth: auth, RefSpecs: refSpecs, Depth: depth}
		if err := opts.Validate(); err != nil {
			return fmt.Errorf("failed to validate fetch %v: %w", refSpecs, err)
		}
		if err = r.Fetch(opts); isFetchErr(err) {
			return fmt.Errorf("failed to fetch %v: %w", refSpecs, err)
		}
	}
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get work tree: %w", err)
	}

	if a.Revision != "" {
		refSpecs := []config.RefSpec{"refs/heads/*:refs/heads/*"}
		if a.SingleBranch {
			refSpecs = []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", a.Branch, a.Branch))}
		}
		opts := &git.FetchOptions{Auth: auth, RefSpecs: refSpecs}
		if err := opts.Validate(); err != nil {
			return fmt.Errorf("failed to validate fetch %v: %w", refSpecs, err)
		}
		if err := r.Fetch(opts); isFetchErr(err) {
			return fmt.Errorf("failed to fetch %v: %w", refSpecs, err)
		}
		h, err := r.ResolveRevision(plumbing.Revision(a.Revision))
		if err != nil {
			return fmt.Errorf("failed to get resolve revision: %w", err)
		}
		if err := w.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(h.String())}); err != nil {
			return fmt.Errorf("failed to checkout %q: %w", h, err)
		}
	}
	if !a.DisableSubmodules {
		s, err := w.Submodules()
		if err != nil {
			return fmt.Errorf("failed to get submodules: %w", err)
		}
		if err := s.Update(&git.SubmoduleUpdateOptions{
			Init:              true,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			Auth:              auth,
		}); err != nil {
			return fmt.Errorf("failed to update submodules: %w", err)
		}
	}
	return nil
}

func isFetchErr(err error) bool {
	return err != nil && err.Error() != "already up-to-date"
}

func (g *ArtifactDriver) OpenStream(a *wfv1.Artifact) (io.ReadCloser, error) {
	// todo: this is a temporary implementation which loads file to disk first
	return common.LoadToStream(a, g)
}

func (g *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}

func (g *ArtifactDriver) IsDirectory(artifact *wfv1.Artifact) (bool, error) {
	return false, argoerrors.New(argoerrors.CodeNotImplemented, "IsDirectory currently unimplemented for Git")
}
