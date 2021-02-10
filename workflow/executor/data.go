package executor

import (
	"context"
	"encoding/json"
	"fmt"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	data2 "github.com/argoproj/argo-workflows/v3/workflow/data"
	"k8s.io/utils/pointer"
)

func (we *WorkflowExecutor) Data(ctx context.Context) error {
	dataTemplate := we.Template.Data
	if dataTemplate == nil {
		return nil
	}

	var data interface{}
	var err error
	switch {
	case dataTemplate.Source != nil:
		switch {
		case dataTemplate.Source.WithArtifactPaths != nil:
			data, err = we.processWithArtifactPaths(ctx, dataTemplate.Source.WithArtifactPaths)
			if err != nil {
				return fmt.Errorf("unable to source artifact paths: %w", err)
			}
		}
	// We could logically add another case here to process inputs when we determine we need a Pod to run certain transformations
	default:
		return fmt.Errorf("internal error: should not launch data Pod if no source is used")
	}

	data, err = data2.ProcessTransformation(dataTemplate.Transformation, data)
	if err != nil {
		return fmt.Errorf("unable to process transformation: %w", err)
	}

	return we.processOutput(ctx, data)
}

func (we *WorkflowExecutor) processWithArtifactPaths(ctx context.Context, artifacts *wfv1.WithArtifactPaths) ([]string, error) {
	driverArt, err := we.newDriverArt(&artifacts.Artifact)
	if err != nil {
		return nil, err
	}
	artDriver, err := we.InitDriver(ctx, driverArt)
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

func (we *WorkflowExecutor) processOutput(ctx context.Context, data interface{}) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	we.Template.Outputs.Result = pointer.StringPtr(string(out))
	err = we.AnnotateOutputs(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
