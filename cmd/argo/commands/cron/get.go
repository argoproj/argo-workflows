package cron

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/argoproj/pkg/errors"
	"github.com/argoproj/pkg/humanize"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func NewGetCommand() *cobra.Command {
	var output string

	command := &cobra.Command{
		Use:   "get CRON_WORKFLOW...",
		Short: "display details about a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			errors.CheckError(err)
			namespace := client.Namespace()

			for _, arg := range args {
				cronWf, err := serviceClient.GetCronWorkflow(ctx, &cronworkflow.GetCronWorkflowRequest{
					Name:      arg,
					Namespace: namespace,
				})
				errors.CheckError(err)
				printCronWorkflow(cronWf, output)
			}
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	return command
}

func printCronWorkflow(wf *wfv1.CronWorkflow, outFmt string) {
	switch outFmt {
	case "name":
		fmt.Println(wf.ObjectMeta.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "wide", "":
		fmt.Print(getCronWorkflowGet(wf))
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func getCronWorkflowGet(cwf *wfv1.CronWorkflow) string {
	const fmtStr = "%-30s %v\n"

	out := ""
	out += fmt.Sprintf(fmtStr, "Name:", cwf.ObjectMeta.Name)
	out += fmt.Sprintf(fmtStr, "Namespace:", cwf.ObjectMeta.Namespace)
	out += fmt.Sprintf(fmtStr, "Created:", humanize.Timestamp(cwf.ObjectMeta.CreationTimestamp.Time))
	out += fmt.Sprintf(fmtStr, "Schedule:", cwf.Spec.GetScheduleString())
	out += fmt.Sprintf(fmtStr, "Suspended:", cwf.Spec.Suspend)
	if cwf.Spec.Timezone != "" {
		out += fmt.Sprintf(fmtStr, "Timezone:", cwf.Spec.Timezone)
	}
	if cwf.Spec.StartingDeadlineSeconds != nil {
		out += fmt.Sprintf(fmtStr, "StartingDeadlineSeconds:", *cwf.Spec.StartingDeadlineSeconds)
	}
	if cwf.Spec.ConcurrencyPolicy != "" {
		out += fmt.Sprintf(fmtStr, "ConcurrencyPolicy:", cwf.Spec.ConcurrencyPolicy)
	}
	if cwf.Status.LastScheduledTime != nil {
		out += fmt.Sprintf(fmtStr, "LastScheduledTime:", humanize.Timestamp(cwf.Status.LastScheduledTime.Time))
	}

	next, err := GetNextRuntime(cwf)
	if err == nil {
		out += fmt.Sprintf(fmtStr, "NextScheduledTime:", humanize.Timestamp(next)+" (assumes workflow-controller is in UTC)")
	}

	if len(cwf.Status.Active) > 0 {
		var activeWfNames []string
		for _, activeWf := range cwf.Status.Active {
			activeWfNames = append(activeWfNames, activeWf.Name)
		}
		out += fmt.Sprintf(fmtStr, "Active Workflows:", strings.Join(activeWfNames, ", "))
	}
	if len(cwf.Status.Conditions) > 0 {
		out += cwf.Status.Conditions.DisplayString(fmtStr, map[wfv1.ConditionType]string{wfv1.ConditionTypeSubmissionError: "✖"})
	}
	if len(cwf.Spec.WorkflowSpec.Arguments.Parameters) > 0 {
		out += fmt.Sprintf(fmtStr, "Workflow Parameters:", "")
		for _, param := range cwf.Spec.WorkflowSpec.Arguments.Parameters {
			if !param.HasValue() {
				continue
			}
			out += fmt.Sprintf(fmtStr, "  "+param.Name+":", param.GetValue())
		}
	}
	return out
}
