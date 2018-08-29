package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/argo/workflow/util"
	"github.com/spf13/cobra"
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
			InitWorkflowClient()
			for _, wfName := range args {
				err := util.SuspendWorkflow(wfClient, wfName)
				if err != nil {
					log.Fatalf("Failed to suspend %s: %v", wfName, err)
				}
				fmt.Printf("workflow %s suspended\n", wfName)
			}
		},
	}
	return command
}
