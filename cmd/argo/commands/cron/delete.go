package cron

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		all bool
	)

	var command = &cobra.Command{
		Use:   "delete CRON_WORKFLOW",
		Short: "delete a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			cronWfClient := InitCronWorkflowClient()
			if all {
				deleteCronWorkflows(cronWfClient, metav1.ListOptions{})
			} else {
				if len(args) == 0 {
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				for _, wftmplName := range args {
					deleCronWorkflow(cronWfClient, wftmplName)
				}
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflow templates")
	return command
}

func deleCronWorkflow(cronWfClient v1alpha1.CronWorkflowInterface, cronWf string) {
	err := cronWfClient.Delete(cronWf, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("CronWorkflow '%s' deleted\n", cronWf)
}

func deleteCronWorkflows(cronWfClient v1alpha1.CronWorkflowInterface, options metav1.ListOptions) {
	cronWfList, err := cronWfClient.List(options)
	if err != nil {
		log.Fatal(err)
	}
	for _, cronWf := range cronWfList.Items {
		deleCronWorkflow(cronWfClient, cronWf.ObjectMeta.Name)
	}
}
