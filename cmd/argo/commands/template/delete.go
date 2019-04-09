package template

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
		Use:   "delete WORKFLOW_TEMPLATE",
		Short: "delete a workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			wftmplClient := InitWorkflowTemplateClient()
			if all {
				deleteWorkflowTemplates(wftmplClient, metav1.ListOptions{})
			} else {
				if len(args) == 0 {
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				for _, wftmplName := range args {
					deleteWorkflowTemplate(wftmplClient, wftmplName)
				}
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflow templates")
	return command
}

func deleteWorkflowTemplate(wftmplClient v1alpha1.WorkflowTemplateInterface, wftmplName string) {
	err := wftmplClient.Delete(wftmplName, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("WorkflowTemplate '%s' deleted\n", wftmplName)
}

func deleteWorkflowTemplates(wftmplClient v1alpha1.WorkflowTemplateInterface, options metav1.ListOptions) {
	wftmplList, err := wftmplClient.List(options)
	if err != nil {
		log.Fatal(err)
	}
	for _, wftmpl := range wftmplList.Items {
		deleteWorkflowTemplate(wftmplClient, wftmpl.ObjectMeta.Name)
	}
}
