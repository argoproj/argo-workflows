package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(suspendCmd)
}

var suspendCmd = &cobra.Command{
	Use:   "suspend WORKFLOW1 WORKFLOW2...",
	Short: "suspend a workflow",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		}
		SuspendWorkflows(args)
	},
}

// SuspendWorkflows suspends a lit of running workflows
func SuspendWorkflows(workflows []string) {
	InitWorkflowClient()
	for _, wfName := range workflows {
		err := common.SuspendWorkflow(wfClient, wfName)
		if err != nil {
			log.Fatalf("Failed to suspend %s: %v", wfName, err)
		}
		fmt.Printf("workflow %s suspended\n", wfName)
	}
}
