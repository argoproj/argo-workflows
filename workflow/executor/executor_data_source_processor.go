package executor

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type executorDataSourceProcessor struct {
	we *WorkflowExecutor
}

func newExecutorDataSourceProcessor(we *WorkflowExecutor) *executorDataSourceProcessor {
	return &executorDataSourceProcessor{
		we: we,
	}
}

func (ep *executorDataSourceProcessor) ProcessArtifactPaths(ctx context.Context, artifacts *wfv1.ArtifactPaths) (any, error) {
	driverArt, err := ep.we.newDriverArt(&artifacts.Artifact)
	if err != nil {
		return nil, err
	}
	artDriver, err := ep.we.InitDriver(ctx, driverArt)
	if err != nil {
		return nil, err
	}

	var files []string
	files, err = artDriver.ListObjects(ctx, &artifacts.Artifact)
	if err != nil {
		return nil, err
	}

	return files, nil
}
