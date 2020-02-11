package template

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
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
		Run: func(cmd *cobra.Command, args []string) {
			var ns string
			if listArgs.allNamespaces {
				ns = apiv1.NamespaceAll
			} else {
				ns, _, _ = client.Config.Namespace()
			}
			var wftmplList *wfv1.WorkflowTemplateList
			var err error
			if client.ArgoServer != "" {
				wftmplReq := workflowtemplatepkg.WorkflowTemplateListRequest{
					Namespace: ns,
				}
				conn := client.GetClientConn()
				wftmplApiClient, ctx := GetWFtmplApiServerGRPCClient(conn)
				wftmplList, err = wftmplApiClient.ListWorkflowTemplates(ctx, &wftmplReq)
				if err != nil {
					log.Fatal(err)
				}

			} else {

				var wftmplClient v1alpha1.WorkflowTemplateInterface
				if listArgs.allNamespaces {
					wftmplClient = InitWorkflowTemplateClient(apiv1.NamespaceAll)
				} else {
					wftmplClient = InitWorkflowTemplateClient()
				}
				listOpts := metav1.ListOptions{}
				labelSelector := labels.NewSelector()
				listOpts.LabelSelector = labelSelector.String()
				wftmplList, err = wftmplClient.List(listOpts)
				if err != nil {
					log.Fatal(err)
				}
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
	command.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "Show workflows from all namespaces")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	return command
}

func printTable(wfList []wfv1.WorkflowTemplate, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		fmt.Fprint(w, "NAMESPACE\t")
	}
	fmt.Fprint(w, "NAME")
	fmt.Fprint(w, "\n")
	for _, wf := range wfList {
		if listArgs.allNamespaces {
			fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Name)
		fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
