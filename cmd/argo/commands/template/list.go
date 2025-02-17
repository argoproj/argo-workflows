package template

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type listFlags struct {
	allNamespaces bool   // --all-namespaces
	output        string // --output
}

func NewListCommand() *cobra.Command {
	var listArgs listFlags
	command := &cobra.Command{
		Use:   "list",
		Short: "list workflow templates",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
			if err != nil {
				log.Fatal(err)
			}
			namespace := client.Namespace()
			if listArgs.allNamespaces {
				namespace = apiv1.NamespaceAll
			}
			wftmplList, err := serviceClient.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{
				Namespace: namespace,
			})
			if err != nil {
				log.Fatal(err)
			}
			switch listArgs.output {
			case "", "wide":
				printTable(wftmplList.Items, &listArgs)
			case "name":
				for _, wftmp := range wftmplList.Items {
					fmt.Println(wftmp.ObjectMeta.Name)
				}
			default:
				log.Fatalf("Unknown output mode: %s", listArgs.output)
			}
		},
	}
	command.Flags().BoolVarP(&listArgs.allNamespaces, "all-namespaces", "A", false, "Show workflows from all namespaces")
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
