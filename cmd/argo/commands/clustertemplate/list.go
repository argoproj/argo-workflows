package clustertemplate

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func NewListCommand() *cobra.Command {
	var output = common.EnumFlagValue{
		AllowedValues: []string{"wide", "name"},
	}
	command := &cobra.Command{
		Use:   "list",
		Short: "list cluster workflow templates",
		Example: `# List Cluster Workflow Templates:
  argo cluster-template list
	
# List Cluster Workflow Templates with additional details such as labels, annotations, and status:
  argo cluster-template list --output wide
  
# List Cluster Workflow Templates by name only:
  argo cluster-template list -o name
`,
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
			switch output.String() {
			case "", "wide":
				printTable(cwftmplList.Items)
			case "name":
				for _, cwftmp := range cwftmplList.Items {
					fmt.Println(cwftmp.ObjectMeta.Name)
				}
			default:
				log.Fatalf("Unknown output mode: %s", output.String())
			}
		},
	}
	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
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
