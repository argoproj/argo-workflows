package commands

import (
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewResubmitCommand() *cobra.Command {
	var (
		memoized   bool
		submitArgs submitFlags
	)
	var command = &cobra.Command{
		Use:   "resubmit WORKFLOW",
		Short: "resubmit a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			wfClient := InitWorkflowClient()
			wf, err := wfClient.Get(args[0], metav1.GetOptions{})
			if err != nil {
				log.Fatal(err)
			}
			newWF, err := common.FormulateResubmitWorkflow(wf, memoized)
			if err != nil {
				log.Fatal(err)
			}
			_, err = submitWorkflow(newWF, &submitArgs)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	command.Flags().StringVarP(&submitArgs.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&submitArgs.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&submitArgs.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&memoized, "memoized", false, "re-use successful steps & outputs from the previous run (experimental)")
	return command
}
