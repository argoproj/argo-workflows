package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDeleteCommand returns a new instance of an `argocd repo` command
func NewDeleteCommand() *cobra.Command {
	var (
		all       bool
		completed bool
	)

	var command = &cobra.Command{
		Use:   "delete WORKFLOW",
		Short: "delete a workflow and its associated pods",
		Run: func(cmd *cobra.Command, args []string) {
			wfClient = InitWorkflowClient()
			if all {
				deleteWorkflows(metav1.ListOptions{})
				return
			} else if completed {
				options := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=true", common.LabelKeyCompleted),
				}
				deleteWorkflows(options)
				return
			}
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			for _, wfName := range args {
				deleteWorkflow(wfName)
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflows")
	command.Flags().BoolVar(&completed, "completed", false, "Delete completed workflows")
	return command
}

func deleteWorkflow(wfName string) {
	err := wfClient.Delete(wfName, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Workflow '%s' deleted\n", wfName)
}

func deleteWorkflows(options metav1.ListOptions) {
	wfList, err := wfClient.List(options)
	if err != nil {
		log.Fatal(err)
	}
	for _, wf := range wfList.Items {
		deleteWorkflow(wf.ObjectMeta.Name)
	}
}
