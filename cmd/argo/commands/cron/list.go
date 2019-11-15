package cron

import (
	"fmt"
	"github.com/argoproj/pkg/humanize"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

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
		Short: "list cron workflows",
		Run: func(cmd *cobra.Command, args []string) {
			var cronWfClient v1alpha1.CronWorkflowInterface
			if listArgs.allNamespaces {
				cronWfClient = InitCronWorkflowClient(apiv1.NamespaceAll)
			} else {
				cronWfClient = InitCronWorkflowClient()
			}
			listOpts := metav1.ListOptions{}
			labelSelector := labels.NewSelector()
			listOpts.LabelSelector = labelSelector.String()
			cronWfList, err := cronWfClient.List(listOpts)
			if err != nil {
				log.Fatal(err)
			}

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
	command.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "Show workflows from all namespaces")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	return command
}

func printTable(wfList []wfv1.CronWorkflow, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		fmt.Fprint(w, "NAMESPACE\t")
	}
	fmt.Fprint(w, "NAME\tAGE\tLAST RUN\tSCHEDULE\tSUSPENDED")
	fmt.Fprint(w, "\n")
	for _, wf := range wfList {
		if listArgs.allNamespaces {
			fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t", wf.ObjectMeta.Name, humanize.RelativeDurationShort(wf.ObjectMeta.CreationTimestamp.Time, time.Now()), humanize.RelativeDurationShort(wf.Status.LastScheduledTime.Time, time.Now()), wf.Options.Schedule, wf.Options.Suspend)
		fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
