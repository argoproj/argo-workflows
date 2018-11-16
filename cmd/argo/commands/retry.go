package commands

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/workflow/util"
)

func NewRetryCommand() *cobra.Command {
	var (
		cliSubmitOpts cliSubmitOpts
	)
	var command = &cobra.Command{
		Use:   "retry WORKFLOW",
		Short: "retry a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			kubeClient := initKubeClient()
			wfClient := InitWorkflowClient()
			wf, err := wfClient.Get(args[0], metav1.GetOptions{})
			if err != nil {
				log.Fatal(err)
			}
			wf, err = util.RetryWorkflow(kubeClient, wfClient, wf)
			if err != nil {
				log.Fatal(err)
			}
			printWorkflow(wf, cliSubmitOpts.output)
			waitOrWatch([]string{wf.Name}, cliSubmitOpts)
		},
	}
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	return command
}
