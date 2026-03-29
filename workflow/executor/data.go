package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/workflow/data"
)

func (we *WorkflowExecutor) processData(ctx context.Context) (any, error) {
	dataCtx, span := we.Tracing.StartProcessDataTemplate(ctx)
	defer span.End()
	dataTemplate := we.Template.Data
	if dataTemplate == nil {
		return nil, fmt.Errorf("no data template found")
	}

	return data.ProcessData(dataCtx, dataTemplate, newExecutorDataSourceProcessor(we))
}

func (we *WorkflowExecutor) Data(ctx context.Context) error {
	transformedData, err := we.processData(ctx)
	if err != nil {
		return fmt.Errorf("unable to process data template: %w", err)
	}

	out, err := json.Marshal(transformedData)
	if err != nil {
		return err
	}
	we.Template.Outputs.Result = new(string(out))
	err = we.ReportOutputs(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
