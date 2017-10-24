package executor

import (
	"path"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	artifacts "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/common"
)

type WorkflowExecutor struct {
	Template       wfv1.Template
	ArtifactDriver artifacts.ArtifactDriver
	//kubeletCl
}

func (we *WorkflowExecutor) LoadArtifacts() error {
	for artName, art := range we.Template.Inputs.Artifacts {
		artPath := path.Join(common.ExecutorArtifactBaseDir, artName)
		we.ArtifactDriver.Load(string(art.From), artPath)
	}

	return nil
}
