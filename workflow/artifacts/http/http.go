package http

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is the artifact driver for a HTTP URL
type ArtifactDriver struct{}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Load download artifacts from an HTTP URL
func (h *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	// Download the file to a local file path
	args := []string{"-fsS", "-L", "-o", path, inputArtifact.HTTP.URL}
	headers := inputArtifact.HTTP.Headers
	for _, v := range headers {
		// Build curl -H string for each key-value header parameter
		args = append(args, "-H", fmt.Sprintf("%s: %s", v.Name, v.Value))
	}
	log.Info(strings.Join(append([]string{"curl"}, args...), " "))
	cmd := exec.Command("curl", args...)
	output, err := cmd.CombinedOutput()
	log.Info(string(output))
	if err != nil {
		log.WithError(err).Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			// https://ec.haxx.se/usingcurl/usingcurl-returns
			// 22 - HTTP page not retrieved.
			if exitErr.ExitCode() == 22 {
				return errors.New(errors.CodeNotFound, exitErr.Error())
			}
		}
		return err
	}
	return nil
}

func (h *ArtifactDriver) Save(string, *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "HTTP output artifacts unsupported")
}

func (h *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
