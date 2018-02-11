package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(resumeCmd)
}

var resumeCmd = &cobra.Command{
	Use:   "resume WORKFLOW1 WORKFLOW2...",
	Short: "resume a workflow",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		}
		ResumeWorkflows(args)
	},
}

// ResumeWorkflows resumes a list of suspended workflows
func ResumeWorkflows(workflows []string) {
	InitWorkflowClient()
	for _, wfName := range workflows {
		err := common.ResumeWorkflow(wfClient, wfName)
		if err != nil {
			log.Fatalf("Failed to resume %s: %+v", wfName, err)
		}
		fmt.Printf("workflow %s resumed\n", wfName)
	}
}
