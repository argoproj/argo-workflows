package clustertemplate

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type listFlags struct {
	output string // --output
}

func NewListCommand() *cobra.Command {
	var listArgs listFlags
	command := &cobra.Command{
		Use:   "list",
		Short: "list cluster workflow templates",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
			if err != nil {
				log.Fatal(err)
			}

			cwftmplList, err := serviceClient.ListClusterWorkflowTemplates(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateListRequest{})
			if err != nil {
				log.Fatal(err)
			}
			switch listArgs.output {
			case "", "wide":
				printTable(cwftmplList.Items)
			case "name":
				for _, cwftmp := range cwftmplList.Items {
					fmt.Println(cwftmp.ObjectMeta.Name)
				}
			default:
				log.Fatalf("Unknown output mode: %s", listArgs.output)
			}
		},
	}
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	return command
}

func printTable(wfList []wfv1.ClusterWorkflowTemplate) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "NAME")
	_, _ = fmt.Fprint(w, "\n")
	for _, wf := range wfList {
		_, _ = fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Name)
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
