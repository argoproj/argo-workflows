package cron

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/humanize"
)

type listFlags struct {
	allNamespaces bool                 // --all-namespaces
	output        common.EnumFlagValue // --output
	labelSelector string               // --selector
}

func NewListCommand() *cobra.Command {
	var listArgs = listFlags{
		output: common.EnumFlagValue{AllowedValues: []string{"wide", "name"}},
	}
	command := &cobra.Command{
		Use:   "list",
		Short: "list cron workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			if err != nil {
				return err
			}
			namespace := client.Namespace(ctx)
			if listArgs.allNamespaces {
				namespace = ""
			}
			listOpts := metav1.ListOptions{}
			listOpts.LabelSelector = listArgs.labelSelector
			cronWfList, err := serviceClient.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{
				Namespace:   namespace,
				ListOptions: &listOpts,
			})
			if err != nil {
				return err
			}
			switch listArgs.output.String() {
			case "", "wide":
				printTable(ctx, cronWfList.Items, &listArgs)
			case "name":
				for _, cronWf := range cronWfList.Items {
					fmt.Println(cronWf.Name)
				}
			default:
				return fmt.Errorf("unknown output mode: %s", listArgs.output.String())
			}
			return nil
		},
	}
	command.Flags().BoolVarP(&listArgs.allNamespaces, "all-namespaces", "A", false, "Show workflows from all namespaces")
	command.Flags().VarP(&listArgs.output, "output", "o", "Output format. "+listArgs.output.Usage())
	command.Flags().StringVarP(&listArgs.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.")
	return command
}

func printTable(ctx context.Context, wfList []wfv1.CronWorkflow, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		_, _ = fmt.Fprint(w, "NAMESPACE\t")
	}
	_, _ = fmt.Fprint(w, "NAME\tAGE\tLAST RUN\tNEXT RUN\tSCHEDULES\tTIMEZONE\tSUSPENDED")
	_, _ = fmt.Fprint(w, "\n")
	for _, cwf := range wfList {
		if listArgs.allNamespaces {
			_, _ = fmt.Fprintf(w, "%s\t", cwf.Namespace)
		}
		var cleanLastScheduledTime string
		if cwf.Status.LastScheduledTime != nil {
			cleanLastScheduledTime = humanize.RelativeDurationShort(cwf.Status.LastScheduledTime.Time, time.Now())
		} else {
			cleanLastScheduledTime = "N/A"
		}
		var cleanNextScheduledTime string
		if next, err := GetNextRuntime(ctx, &cwf); err == nil {
			cleanNextScheduledTime = humanize.RelativeDurationShort(next, time.Now())
		} else {
			cleanNextScheduledTime = "N/A"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t", cwf.Name, humanize.RelativeDurationShort(cwf.CreationTimestamp.Time, time.Now()), cleanLastScheduledTime, cleanNextScheduledTime, cwf.Spec.GetScheduleString(), cwf.Spec.Timezone, cwf.Spec.Suspend)
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
