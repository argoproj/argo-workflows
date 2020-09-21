package template

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type listFlags struct {
	allNamespaces bool   // --all-namespaces
	output        string // --output
}

func NewListCommand() *cobra.Command {
	var (
		listArgs listFlags
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "list workflow templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewWorkflowTemplateServiceClient()
			namespace := client.Namespace()
			if listArgs.allNamespaces {
				namespace = apiv1.NamespaceAll
			}
			wftmplList, err := serviceClient.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{
				Namespace: namespace,
			})
			if err != nil {
				return err
			}
			switch listArgs.output {
			case "", "wide":
				printTable(wftmplList.Items, &listArgs)
			case "name":
				for _, wftmp := range wftmplList.Items {
					fmt.Println(wftmp.ObjectMeta.Name)
				}
			default:
				return fmt.Errorf("Unknown output mode: %s", listArgs.output)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "Show workflows from all namespaces")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
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
			_, _ = fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		_, _ = fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Name)
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
