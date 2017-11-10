package git

import (
	"os/exec"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
)

// GitArtifactDriver is the artifact driver for a git repo
type GitArtifactDriver struct{}

// Load download artifacts from an git URL
func (g *GitArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	// Download the file to a local file path
	cmd := exec.Command("git", "clone", inputArtifact.Git.Repo, path)
	err := cmd.Run()
	if err != nil {
		exErr := err.(*exec.ExitError)
		log.Errorf("`%s %s` failed: %s", cmd.Path, strings.Join(cmd.Args, " "), exErr.Stderr)
		return errors.InternalWrapError(err)
	}
	if inputArtifact.Git.Revision != "" {
		cmd = exec.Command("git", "-C", path, "checkout", inputArtifact.Git.Revision)
		err := cmd.Run()
		if err != nil {
			exErr := err.(*exec.ExitError)
			log.Errorf("`%s %s` failed: %s", cmd.Path, strings.Join(cmd.Args, " "), exErr.Stderr)
			return errors.InternalWrapError(err)
		}
	}
	return nil
}

func (g *GitArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Git output artifacts unsupported")
}
