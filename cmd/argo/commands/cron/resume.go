package cron

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

// NewSuspendCommand returns a new instance of an `argo suspend` command
func NewResumeCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "resume CRON_WORKFLOW",
		Short: "resume a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			cronWfClient := InitCronWorkflowClient()
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			for _, wftmplName := range args {
				resumeCronWorkflow(cronWfClient, wftmplName)
			}
		},
	}

	return command
}

func resumeCronWorkflow(cronWfClient v1alpha1.CronWorkflowInterface, cronWfName string) {
	cronWf, err := cronWfClient.Get(cronWfName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	cronWf.Spec.Suspend = false
	newCronWf, err := cronWfClient.Update(cronWf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("CronWorkflow '%s' resumed\n", newCronWf.Name)
}
