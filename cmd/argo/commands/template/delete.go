package template

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/argoproj/pkg/errors"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

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
			if client.ArgoServer != "" {
				apiServerDeleteWorkflowTemplates(all, args)
			} else {
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
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflow templates")
	return command
}

func apiServerDeleteWorkflowTemplates(allWFs bool, wfTmplNames []string) {
	conn := client.GetClientConn()
	defer conn.Close()
	ns, _, _ := client.Config.Namespace()
	wftmplApiClient, ctx := GetWFtmplApiServerGRPCClient(conn)

	var delWFTmplNames []string
	var err error
	if allWFs {
		wftmplReq := workflowtemplate.WorkflowTemplateListRequest{
			Namespace: ns,
		}
		var wftmplList *wfv1.WorkflowTemplateList
		wftmplList, err = wftmplApiClient.ListWorkflowTemplates(ctx, &wftmplReq)
		if err != nil {
			log.Fatal(err)
		}
		for _, wfTmpl := range wftmplList.Items {
			delWFTmplNames = append(delWFTmplNames, wfTmpl.Name)
		}

	} else {
		delWFTmplNames = wfTmplNames
	}
	for _, wfTmplNames := range delWFTmplNames {
		apiServerDeleteWorkflowTemplate(wftmplApiClient, ctx, ns, wfTmplNames)
	}

}

func apiServerDeleteWorkflowTemplate(client workflowtemplate.WorkflowTemplateServiceClient, ctx context.Context, ns, wftmplName string) {
	wfReq := workflowtemplate.WorkflowTemplateDeleteRequest{
		Name: wftmplName,
		Namespace:    ns,
	}
	_, err := client.DeleteWorkflowTemplate(ctx, &wfReq)
	if err != nil {
		errors.CheckError(err)
	}
	fmt.Printf("WorkflowTemplate '%s' deleted\n", wftmplName)
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
