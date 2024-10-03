package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// CliSubmitOpts holds submission options specific to CLI submission (e.g. controlling output)
type CliSubmitOpts struct {
	Output        EnumFlagValue // --output
	Wait          bool          // --wait
	Watch         bool          // --watch
	Log           bool          // --log
	Strict        bool          // --strict
	Priority      *int32        // --priority
	GetArgs       GetFlags
	ScheduledTime string   // --scheduled-time
	Parameters    []string // --parameter
}

func NewCliSubmitOpts() CliSubmitOpts {
	return CliSubmitOpts{
		Output: NewPrintWorkflowOutputValue(""),
	}
}

func WaitWatchOrLog(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, cliSubmitOpts CliSubmitOpts) {
	if cliSubmitOpts.Log {
		for _, workflow := range workflowNames {
			LogWorkflow(ctx, serviceClient, namespace, workflow, "", "", "", &corev1.PodLogOptions{
				Container: common.MainContainerName,
				Follow:    true,
				Previous:  false,
			})
		}
	}
	if cliSubmitOpts.Wait {
		WaitWorkflows(ctx, serviceClient, namespace, workflowNames, false, !(cliSubmitOpts.Output.String() == "" || cliSubmitOpts.Output.String() == "wide"))
	} else if cliSubmitOpts.Watch {
		for _, workflow := range workflowNames {
			WatchWorkflow(ctx, serviceClient, namespace, workflow, cliSubmitOpts.GetArgs)
		}
	}
}
