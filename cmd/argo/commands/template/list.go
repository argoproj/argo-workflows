package template

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type listFlags struct {
	allNamespaces bool                 // --all-namespaces
	output        common.EnumFlagValue // --output
	labels        string               // --selector
}

func NewListCommand() *cobra.Command {
	var listArgs = listFlags{output: common.EnumFlagValue{AllowedValues: []string{"wide", "name"}}}
	command := &cobra.Command{
		Use:   "list",
		Short: "list workflow templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
			if err != nil {
				return err
			}
			namespace := client.Namespace(ctx)
			if listArgs.allNamespaces {
				namespace = apiv1.NamespaceAll
			}
			labelSelector, err := labels.Parse(listArgs.labels)
			if err != nil {
				return err
			}

			wftmplList, err := serviceClient.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{
				Namespace: namespace,
				ListOptions: &metav1.ListOptions{
					LabelSelector: labelSelector.String(),
				},
			})
			if err != nil {
				return err
			}
			switch listArgs.output.String() {
			case "", "wide":
				printTable(wftmplList.Items, &listArgs)
			case "name":
				for _, wftmp := range wftmplList.Items {
					fmt.Println(wftmp.Name)
				}
			default:
				return fmt.Errorf("unknown output mode: %s", listArgs.output)
			}
			return nil
		},
	}
	command.Flags().BoolVarP(&listArgs.allNamespaces, "all-namespaces", "A", false, "Show workflows from all namespaces")
	command.Flags().VarP(&listArgs.output, "output", "o", "Output format. "+listArgs.output.Usage())
	command.Flags().StringVarP(&listArgs.labels, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	return command
}

func printTable(wfList []wfv1.WorkflowTemplate, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		_, _ = fmt.Fprint(w, "NAMESPACE\t")
	}
	_, _ = fmt.Fprint(w, "NAME")
	_, _ = fmt.Fprint(w, "\n")
	for _, wf := range wfList {
		if listArgs.allNamespaces {
			_, _ = fmt.Fprintf(w, "%s\t", wf.Namespace)
		}
		_, _ = fmt.Fprintf(w, "%s\t", wf.Name)
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
