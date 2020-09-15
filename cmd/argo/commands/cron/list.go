package cron

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/argoproj/pkg/humanize"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
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
		Short: "list cron workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClient()
			serviceClient := apiClient.NewCronWorkflowServiceClient()
			namespace := client.Namespace()
			if listArgs.allNamespaces {
				namespace = ""
			}
			listOpts := metav1.ListOptions{}
			labelSelector := labels.NewSelector()
			listOpts.LabelSelector = labelSelector.String()
			cronWfList, err := serviceClient.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{
				Namespace:   namespace,
				ListOptions: &listOpts,
			})
			if err != nil {
				return err
			}
			switch listArgs.output {
			case "", "wide":
				printTable(cronWfList.Items, &listArgs)
			case "name":
				for _, cronWf := range cronWfList.Items {
					fmt.Println(cronWf.ObjectMeta.Name)
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

func printTable(wfList []wfv1.CronWorkflow, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		_, _ = fmt.Fprint(w, "NAMESPACE\t")
	}
	_, _ = fmt.Fprint(w, "NAME\tAGE\tLAST RUN\tNEXT RUN\tSCHEDULE\tSUSPENDED")
	_, _ = fmt.Fprint(w, "\n")
	for _, wf := range wfList {
		if listArgs.allNamespaces {
			_, _ = fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		var cleanLastScheduledTime string
		if wf.Status.LastScheduledTime != nil {
			cleanLastScheduledTime = humanize.RelativeDurationShort(wf.Status.LastScheduledTime.Time, time.Now())
		} else {
			cleanLastScheduledTime = "N/A"
		}
		var cleanNextScheduledTime string
		if next, err := wf.GetNextRuntime(); err == nil {
			cleanNextScheduledTime = humanize.RelativeDurationShort(next, time.Now())
		} else {
			cleanNextScheduledTime = "N/A"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%t", wf.ObjectMeta.Name, humanize.RelativeDurationShort(wf.ObjectMeta.CreationTimestamp.Time, time.Now()), cleanLastScheduledTime, cleanNextScheduledTime, wf.Spec.Schedule, wf.Spec.Suspend)
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
