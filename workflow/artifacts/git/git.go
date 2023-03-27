package git

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

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
		var gitAuth []string
		gitAuth = append(gitAuth, "-c", fmt.Sprintf("url.https://%s:%s@.insteadOf=https://", g.Username, g.Password))
		return func() {},
			&http.BasicAuth{Username: g.Username, Password: g.Password},
			gitAuth,
			nil
	}
	return func() {}, nil, nil, nil
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
	closer, auth, env, err := g.auth(sshUser)
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
		if err = r.Fetch(opts); isAlreadyUpToDateErr(err) {
			return fmt.Errorf("failed to fetch %v: %w", refSpecs, err)
		}
	}

	if a.Revision != "" {
		// We still rely on forking git for checkout, since go-git does not have a reliable
		// way of resolving revisions (e.g. mybranch, HEAD^, v1.2.3)
		rev := getRevisionForCheckout(inputArtifact.Git.Revision)
		log.Info("Checking out revision ", rev)
		cmd := exec.Command("git", "checkout", rev, "--")
		cmd.Dir = path
		cmd.Env = env
		output, err := cmd.Output()
		if err != nil {
			return g.error(err, cmd)
		}
		log.Infof("`%s` stdout:\n%s", cmd.Args, string(output))
	}
	if !a.DisableSubmodules {
		submodulesCmd := exec.Command("git", "submodule", "update", "--init", "--recursive", "--force")
		submodulesCmd.Dir = path
		submodulesCmd.Env = env
		submoduleOutput, err := submodulesCmd.Output()
		if err != nil {
			return g.error(err, submodulesCmd)
		}
		log.Infof("`%s` stdout:\n%s", submodulesCmd.Args, string(submoduleOutput))
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
