package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVar(&deleteArgs.all, "all", false, "Delete all workflows")
}

type deleteFlags struct {
	all bool // --all
}

var deleteArgs deleteFlags

var deleteCmd = &cobra.Command{
	Use:   "delete WORKFLOW",
	Short: "delete commands",
	Run:   deleteWorkflowCmd,
}

func deleteWorkflowCmd(cmd *cobra.Command, args []string) {
	wfClient = initWorkflowClient()
	if deleteArgs.all {
		deleteAllWorkflows()
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
	err := wfClient.DeleteWorkflow(wfName, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Workflow '%s' deleted\n", wfName)
}

func deleteAllWorkflows() {
	wfList, err := wfClient.ListWorkflows(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, wf := range wfList.Items {
		deleteWorkflow(wf.ObjectMeta.Name)
	}
}
