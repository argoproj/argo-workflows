package git

import (
	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
)

// GitArtifactDriver is the artifact driver for a git repo
type GitArtifactDriver struct{}

// Load download artifacts from an git URL
func (g *GitArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	// Download the file to a local file path
	err := common.RunCommand("git", "clone", inputArtifact.Git.Repo, path)
	if err != nil {
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

func (g *GitArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Git output artifacts unsupported")
}
