package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/workflow/util"
)

func NewResumeCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "resume WORKFLOW1 WORKFLOW2...",
		Short: "resume a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			namespace, _, _ := client.Config.Namespace()
			if client.ArgoServer != "" {
				conn := client.GetClientConn()
				apiGRPCClient, ctx := GetWFApiServerGRPCClient(conn)
				for _, wfName := range args {
					wfUptReq := workflowpkg.WorkflowResumeRequest{
						Name:      wfName,
						Namespace: namespace,
					}
					wf, err := apiGRPCClient.ResumeWorkflow(ctx, &wfUptReq)
					if err != nil {
						log.Fatalf("Failed to resume %s: %+v", wfName, err)
					}
					fmt.Printf("workflow %s resumed\n", wf.Name)
				}
			} else {
				InitWorkflowClient()
				for _, wfName := range args {
					err := util.ResumeWorkflow(wfClient, wfName)
					if err != nil {
						log.Fatalf("Failed to resume %s: %+v", wfName, err)
					}
					fmt.Printf("workflow %s resumed\n", wfName)
				}
			}
		},
	}
	return command
}
