package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/workflow/util"
)

func NewTerminateCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "terminate WORKFLOW WORKFLOW2...",
		Short: "terminate a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			namespace, _, _ := client.Config.Namespace()
			if client.ArgoServer != "" {
				conn := client.GetClientConn()
				apiGRPCClient, ctx := GetApiServerGRPCClient(conn)
				for _, wfName := range args {
					wfUptReq := workflow.WorkflowUpdateRequest{
						WorkflowName: wfName,
						Namespace:    namespace,
						Memoized:     false,
					}
					wf, err := apiGRPCClient.TerminateWorkflow(ctx, &wfUptReq)
					errors.CheckError(err)
					fmt.Printf("workflow %s terminated\n", wf.Name)
				}
			} else {
				InitWorkflowClient()
				for _, name := range args {
					err := util.TerminateWorkflow(wfClient, name)
					errors.CheckError(err)
					fmt.Printf("Workflow '%s' terminated\n", name)
				}
			}
		},
	}
	return command
}
