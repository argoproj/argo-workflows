package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/workflow/util"
)

func NewSuspendCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "suspend WORKFLOW1 WORKFLOW2...",
		Short: "suspend a workflow",
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
					wf, err := apiGRPCClient.SuspendWorkflow(ctx, &wfUptReq)
					if err != nil {
						log.Fatalf("Failed to suspended %s: %+v", wfName, err)
					}
					fmt.Printf("workflow %s suspended\n", wf.Name)
				}
			} else {
				InitWorkflowClient()
				for _, wfName := range args {
					err := util.SuspendWorkflow(wfClient, wfName)
					if err != nil {
						log.Fatalf("Failed to suspend %s: %v", wfName, err)
					}
					fmt.Printf("workflow %s suspended\n", wfName)
				}
			}
		},
	}
	return command
}
