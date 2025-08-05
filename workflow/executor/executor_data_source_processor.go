package executor

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type executorDataSourceProcessor struct {
	ctx context.Context
	we  *WorkflowExecutor
}

func newExecutorDataSourceProcessor(ctx context.Context, we *WorkflowExecutor) *executorDataSourceProcessor {
	return &executorDataSourceProcessor{
		ctx: ctx,
		we:  we,
	}
}

func (ep *executorDataSourceProcessor) ProcessArtifactPaths(artifacts *wfv1.ArtifactPaths) (interface{}, error) {
	driverArt, err := ep.we.newDriverArt(&artifacts.Artifact)
	if err != nil {
		return nil, err
	}
	artDriver, err := ep.we.InitDriver(ep.ctx, driverArt)
	if err != nil {
		return nil, err
	}

	var files []string
	files, err = artDriver.ListObjects(&artifacts.Artifact)
	if err != nil {
		return nil, err
	}

	return files, nil
}
