package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/humanize"
	argoJson "github.com/argoproj/argo-workflows/v4/util/json"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// GetNextRuntime returns the next time the workflow should run in local time. It assumes the workflow-controller is in
// UTC, but nevertheless returns the time in the local timezone.
func GetNextRuntime(ctx context.Context, cwf *v1alpha1.CronWorkflow) (time.Time, error) {
	var nextRunTime time.Time
	now := time.Now().UTC()
	for _, schedule := range cwf.Spec.GetSchedulesWithTimezone() {
		cronSchedule, err := cron.ParseStandard(schedule)
		if err != nil {
			return time.Time{}, err
		}
		next := cronSchedule.Next(now).Local()
		if nextRunTime.IsZero() || next.Before(nextRunTime) {
			nextRunTime = next
		}
	}

	return nextRunTime, nil
}

func generateCronWorkflows(ctx context.Context, filePaths []string, strict bool) []v1alpha1.CronWorkflow {
	fileContents, err := util.ReadManifest(ctx, filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var cronWorkflows []v1alpha1.CronWorkflow
	for _, body := range fileContents {
		cronWfs := unmarshalCronWorkflows(ctx, body, strict)
		cronWorkflows = append(cronWorkflows, cronWfs...)
	}

	if len(cronWorkflows) == 0 {
		log.Fatalln("No CronWorkflows found in given files")
	}

	return cronWorkflows
}

// unmarshalCronWorkflows unmarshals the input bytes as either json or yaml
func unmarshalCronWorkflows(ctx context.Context, wfBytes []byte, strict bool) []v1alpha1.CronWorkflow {
	var cronWf v1alpha1.CronWorkflow
	var jsonOpts []argoJson.Opt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &cronWf, jsonOpts...)
	if err == nil {
		return []v1alpha1.CronWorkflow{cronWf}
	}
	yamlWfs, err := common.SplitCronWorkflowYAMLFile(ctx, wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse cron workflow: %v", err)
	return nil
}

func printCronWorkflow(ctx context.Context, wf *v1alpha1.CronWorkflow, outFmt string) {
	switch outFmt {
	case "name":
		fmt.Println(wf.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "wide", "":
		fmt.Print(getCronWorkflowGet(ctx, wf))
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func getCronWorkflowGet(ctx context.Context, cwf *v1alpha1.CronWorkflow) string {
	const fmtStr = "%-30s %v\n"

	var out strings.Builder
	fmt.Fprintf(&out, fmtStr, "Name:", cwf.Name)
	fmt.Fprintf(&out, fmtStr, "Namespace:", cwf.Namespace)
	fmt.Fprintf(&out, fmtStr, "Created:", humanize.Timestamp(cwf.CreationTimestamp.Time))
	fmt.Fprintf(&out, fmtStr, "Schedules:", cwf.Spec.GetScheduleString())
	fmt.Fprintf(&out, fmtStr, "Suspended:", cwf.Spec.Suspend)
	if cwf.Spec.Timezone != "" {
		fmt.Fprintf(&out, fmtStr, "Timezone:", cwf.Spec.Timezone)
	}
	if cwf.Spec.StartingDeadlineSeconds != nil {
		fmt.Fprintf(&out, fmtStr, "StartingDeadlineSeconds:", *cwf.Spec.StartingDeadlineSeconds)
	}
	if cwf.Spec.ConcurrencyPolicy != "" {
		fmt.Fprintf(&out, fmtStr, "ConcurrencyPolicy:", cwf.Spec.ConcurrencyPolicy)
	}
	if cwf.Status.LastScheduledTime != nil {
		fmt.Fprintf(&out, fmtStr, "LastScheduledTime:", humanize.Timestamp(cwf.Status.LastScheduledTime.Time))
	}

	next, err := GetNextRuntime(ctx, cwf)
	if err == nil {
		fmt.Fprintf(&out, fmtStr, "NextScheduledTime:", humanize.Timestamp(next)+" (assumes workflow-controller is in UTC)")
	}

	if len(cwf.Status.Active) > 0 {
		var activeWfNames []string
		for _, activeWf := range cwf.Status.Active {
			activeWfNames = append(activeWfNames, activeWf.Name)
		}
		fmt.Fprintf(&out, fmtStr, "Active Workflows:", strings.Join(activeWfNames, ", "))
	}
	if len(cwf.Status.Conditions) > 0 {
		out.WriteString(cwf.Status.Conditions.DisplayString(fmtStr, map[v1alpha1.ConditionType]string{v1alpha1.ConditionTypeSubmissionError: "✖"}))
	}
	if len(cwf.Spec.WorkflowSpec.Arguments.Parameters) > 0 {
		fmt.Fprintf(&out, fmtStr, "Workflow Parameters:", "")
		for _, param := range cwf.Spec.WorkflowSpec.Arguments.Parameters {
			if !param.HasValue() {
				continue
			}
			fmt.Fprintf(&out, fmtStr, "  "+param.Name+":", param.GetValue())
		}
	}
	return out.String()
}
