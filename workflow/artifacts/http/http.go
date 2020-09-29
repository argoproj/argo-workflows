package http

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// HTTPArtifactDriver is the artifact driver for a HTTP URL
type HTTPArtifactDriver struct{}

// Load download artifacts from an HTTP URL
func (h *HTTPArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
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

func (h *HTTPArtifactDriver) Save(string, *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "HTTP output artifacts unsupported")
}
