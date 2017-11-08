package http

import (
	"os/exec"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
)

// HTTPArtifactDriver is the artifact driver for a HTTP URL
type HTTPArtifactDriver struct {
	URL string
}

// Load download artifacts from an HTTP URL
func (h *HTTPArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	// Download the file to a local file path
	cmd := exec.Command("curl", "-L", "-o", path, inputArtifact.HTTP.URL)
	err := cmd.Run()
	if err != nil {
		exErr := err.(*exec.ExitError)
		log.Errorf("`%s %s` failed: %s", cmd.Path, strings.Join(cmd.Args, " "), exErr.Stderr)
		return errors.InternalWrapError(err)
	}
	return nil
}

func (h *HTTPArtifactDriver) Save(path string, destURL string) (string, error) {

	return destURL, nil
}
