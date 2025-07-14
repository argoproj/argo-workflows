package data

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func ProcessData(ctx context.Context, data *wfv1.Data, processor wfv1.DataSourceProcessor) (interface{}, error) {
	sourcedData, err := processSource(ctx, data.Source, processor)
	if err != nil {
		return nil, fmt.Errorf("unable to process data source: %w", err)
	}
	transformedData, err := processTransformation(sourcedData, &data.Transformation)
	if err != nil {
		return nil, fmt.Errorf("unable to process data transformation: %w", err)
	}
	return transformedData, nil
}

func processSource(ctx context.Context, source wfv1.DataSource, processor wfv1.DataSourceProcessor) (interface{}, error) {
	var data interface{}
	var err error
	switch {
	case source.ArtifactPaths != nil:
		data, err = processor.ProcessArtifactPaths(ctx, source.ArtifactPaths)
		if err != nil {
			return nil, fmt.Errorf("unable to source artifact paths: %w", err)
		}
	default:
		return nil, fmt.Errorf("no valid source is used for data template")
	}

	return data, nil
}

func processTransformation(data interface{}, transformation *wfv1.Transformation) (interface{}, error) {
	if transformation == nil {
		return data, nil
	}

	var err error
	for i, step := range *transformation {
		switch {
		case step.Expression != "":
			data, err = processExpression(step.Expression, data)
		}
		if err != nil {
			return nil, fmt.Errorf("error processing data step %d: %w", i, err)
		}
	}

	return data, nil
}

func processExpression(expression string, data interface{}) (interface{}, error) {
	env := map[string]interface{}{"data": data}
	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		return nil, err
	}
	return expr.Run(program, env)
}
