package cron

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/argoproj/pkg/humanize"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type listFlags struct {
	allNamespaces bool   // --all-namespaces
	output        string // --output
	labelSelector string // --selector
}

var (
	listCronWFExample = `# List all cron workflows

  argo cron list

# List all cron workflows in all namespaces

  argo cron list --all-namespaces

# List all cron workflows in all namespaces with a label selector

  argo cron list --all-namespaces --selector key1=value1,key2=value2

# List all cron workflows in all namespaces and output only the names

  argo cron list --all-namespaces --output name`
)

func NewListCommand() *cobra.Command {
	var listArgs listFlags
	command := &cobra.Command{
		Use:     "list",
		Short:   "list cron workflows",
		Example: listCronWFExample,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			errors.CheckError(err)
			namespace := client.Namespace()
			if listArgs.allNamespaces {
				namespace = ""
			}
			listOpts := metav1.ListOptions{}
			listOpts.LabelSelector = listArgs.labelSelector
			cronWfList, err := serviceClient.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{
				Namespace:   namespace,
				ListOptions: &listOpts,
			})
			errors.CheckError(err)
			switch listArgs.output {
			case "", "wide":
				printTable(cronWfList.Items, &listArgs)
			case "name":
				for _, cronWf := range cronWfList.Items {
					fmt.Println(cronWf.ObjectMeta.Name)
				}
			default:
				log.Fatalf("Unknown output mode: %s", listArgs.output)
			}
		},
	}
	command.Flags().BoolVarP(&listArgs.allNamespaces, "all-namespaces", "A", false, "Show workflows from all namespaces")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	command.Flags().StringVarP(&listArgs.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.")
	return command
}

func printTable(wfList []wfv1.CronWorkflow, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		_, _ = fmt.Fprint(w, "NAMESPACE\t")
	}
	_, _ = fmt.Fprint(w, "NAME\tAGE\tLAST RUN\tNEXT RUN\tSCHEDULES\tTIMEZONE\tSUSPENDED")
	_, _ = fmt.Fprint(w, "\n")
	for _, cwf := range wfList {
		if listArgs.allNamespaces {
			_, _ = fmt.Fprintf(w, "%s\t", cwf.ObjectMeta.Namespace)
		}
		var cleanLastScheduledTime string
		if cwf.Status.LastScheduledTime != nil {
			cleanLastScheduledTime = humanize.RelativeDurationShort(cwf.Status.LastScheduledTime.Time, time.Now())
		} else {
			cleanLastScheduledTime = "N/A"
		}
		var cleanNextScheduledTime string
		if next, err := GetNextRuntime(&cwf); err == nil {
			cleanNextScheduledTime = humanize.RelativeDurationShort(next, time.Now())
		} else {
			cleanNextScheduledTime = "N/A"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t", cwf.ObjectMeta.Name, humanize.RelativeDurationShort(cwf.ObjectMeta.CreationTimestamp.Time, time.Now()), cleanLastScheduledTime, cleanNextScheduledTime, cwf.Spec.GetScheduleString(), cwf.Spec.Timezone, cwf.Spec.Suspend)
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
