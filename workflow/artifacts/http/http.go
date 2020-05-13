package http

import (
	"os/exec"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

// HTTPArtifactDriver is the artifact driver for a HTTP URL
type HTTPArtifactDriver struct{}

// Load download artifacts from an HTTP URL
func (h *HTTPArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	// Download the file to a local file path
	err := common.RunCommand("curl", "-fsS", "-L", "-o", path, inputArtifact.HTTP.URL)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 6 || exitErr.ExitCode() == 22 {
				return errors.New(errors.CodeNotFound, exitErr.Error())
			}
		}
		return err
	}
	return nil
}

func (h *HTTPArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "HTTP output artifacts unsupported")
}
