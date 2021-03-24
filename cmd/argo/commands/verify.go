package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/TwinProduction/go-color"
	"github.com/argoproj/pkg/errors"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/workflow/verify"
)

func verifyWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string) {
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "VERIFICATION")
	failAtEnd := false
	for _, name := range workflowNames {
		wf, err := serviceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{
			Namespace: namespace,
			Name:      name,
		})
		errors.CheckError(err)
		if err := verify.Workflow(wf); err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "%s %s: %v\n", color.Ize(color.Red, "✖"), name, err)
			failAtEnd = true
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", color.Ize(color.Green, "✔"), name)
		}
	}
	if failAtEnd {
		os.Exit(1)
	}
}
