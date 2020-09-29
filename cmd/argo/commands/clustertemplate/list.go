package clustertemplate

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type listFlags struct {
	output string // --output
}

func NewListCommand() *cobra.Command {
	var (
		listArgs listFlags
	)
	var command = &cobra.Command{
		Use:          "list",
		Short:        "list cluster workflow templates",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()

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
			return nil
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
