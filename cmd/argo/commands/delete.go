package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVar(&deleteArgs.all, "all", false, "Delete all workflows")
	deleteCmd.Flags().BoolVar(&deleteArgs.completed, "completed", false, "Delete completed workflows")
}

type deleteFlags struct {
	all       bool // --all
	completed bool // --completed
}

var deleteArgs deleteFlags

var deleteCmd = &cobra.Command{
	Use:   "delete WORKFLOW",
	Short: "delete a workflow and its associated pods",
	Run:   deleteWorkflowCmd,
}

func deleteWorkflowCmd(cmd *cobra.Command, args []string) {
	wfClient = InitWorkflowClient()
	if deleteArgs.all {
		deleteWorkflows(metav1.ListOptions{})
		return
	} else if deleteArgs.completed {
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
