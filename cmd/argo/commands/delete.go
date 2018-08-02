package commands

import (
	"fmt"
	"log"
	"os"
	"time"

	argotime "github.com/argoproj/pkg/time"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/workflow/common"
)

var (
	completedWorkflowListOption = metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", common.LabelKeyCompleted),
	}
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		all       bool
		completed bool
		older     string
	)

	var command = &cobra.Command{
		Use:   "delete WORKFLOW",
		Short: "delete a workflow and its associated pods",
		Run: func(cmd *cobra.Command, args []string) {
			wfClient = InitWorkflowClient()
			if all {
				deleteWorkflows(metav1.ListOptions{}, nil)
			} else if older != "" {
				olderTime, err := argotime.ParseSince(older)
				if err != nil {
					log.Fatal(err)
				}
				deleteWorkflows(completedWorkflowListOption, olderTime)
			} else if completed {
				deleteWorkflows(completedWorkflowListOption, nil)
			} else {
				if len(args) == 0 {
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				for _, wfName := range args {
					deleteWorkflow(wfName)
				}
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflows")
	command.Flags().BoolVar(&completed, "completed", false, "Delete completed workflows")
	command.Flags().StringVar(&older, "older", "", "Delete completed workflows older than the specified duration (e.g. 10m, 3h, 1d)")
	return command
}

func deleteWorkflow(wfName string) {
	err := wfClient.Delete(wfName, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Workflow '%s' deleted\n", wfName)
}

func deleteWorkflows(options metav1.ListOptions, older *time.Time) {
	wfList, err := wfClient.List(options)
	if err != nil {
		log.Fatal(err)
	}
	for _, wf := range wfList.Items {
		if older != nil {
			if wf.Status.FinishedAt.IsZero() || wf.Status.FinishedAt.After(*older) {
				continue
			}
		}
		deleteWorkflow(wf.ObjectMeta.Name)
	}
}
