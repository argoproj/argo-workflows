package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	argoJson "github.com/argoproj/pkg/json"
	"github.com/robfig/cron/v3"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"

	"github.com/argoproj/pkg/humanize"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// GetNextRuntime returns the next time the workflow should run in local time. It assumes the workflow-controller is in
// UTC, but nevertheless returns the time in the local timezone.
func GetNextRuntime(ctx context.Context, cwf *v1alpha1.CronWorkflow) (time.Time, error) {
	var nextRunTime time.Time
	now := time.Now().UTC()
	for _, schedule := range cwf.Spec.GetSchedulesWithTimezone(ctx) {
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

func generateCronWorkflows(filePaths []string, strict bool) []v1alpha1.CronWorkflow {
	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var cronWorkflows []v1alpha1.CronWorkflow
	for _, body := range fileContents {
		cronWfs := unmarshalCronWorkflows(body, strict)
		cronWorkflows = append(cronWorkflows, cronWfs...)
	}

	if len(cronWorkflows) == 0 {
		log.Fatalln("No CronWorkflows found in given files")
	}

	return cronWorkflows
}

// unmarshalCronWorkflows unmarshals the input bytes as either json or yaml
func unmarshalCronWorkflows(wfBytes []byte, strict bool) []wfv1.CronWorkflow {
	var cronWf wfv1.CronWorkflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &cronWf, jsonOpts...)
	if err == nil {
		return []wfv1.CronWorkflow{cronWf}
	}
	yamlWfs, err := common.SplitCronWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse cron workflow: %v", err)
	return nil
}

func printCronWorkflow(ctx context.Context, wf *wfv1.CronWorkflow, outFmt string) {
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
		fmt.Print(getCronWorkflowGet(ctx, wf))
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func getCronWorkflowGet(ctx context.Context, cwf *wfv1.CronWorkflow) string {
	const fmtStr = "%-30s %v\n"

	out := ""
	out += fmt.Sprintf(fmtStr, "Name:", cwf.ObjectMeta.Name)
	out += fmt.Sprintf(fmtStr, "Namespace:", cwf.ObjectMeta.Namespace)
	out += fmt.Sprintf(fmtStr, "Created:", humanize.Timestamp(cwf.ObjectMeta.CreationTimestamp.Time))
	out += fmt.Sprintf(fmtStr, "Schedules:", cwf.Spec.GetScheduleString())
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

	next, err := GetNextRuntime(ctx, cwf)
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
