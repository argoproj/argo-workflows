package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(pauseCmd)
}

var pauseCmd = &cobra.Command{
	Use:   "pause WORKFLOW1 WORKFLOW2...",
	Short: "pause a workflow",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		}
		PauseWorkflows(args)
	},
}

// PauseWorkflows pauses a lit of running workflows
func PauseWorkflows(workflows []string) {
	InitWorkflowClient()
	for _, wfName := range workflows {
		err := common.PauseWorkflow(wfClient, wfName)
		if err != nil {
			log.Fatalf("Failed to pause %s: %+v", wfName, err)
		}
		fmt.Printf("workflow %s paused\n", wfName)
	}
}
