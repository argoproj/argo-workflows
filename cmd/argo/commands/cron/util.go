package cron

import (
	"log"
	"os"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/workflow/util"

	"github.com/robfig/cron/v3"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// GetNextRuntime returns the next time the workflow should run in local time. It assumes the workflow-controller is in
// UTC, but nevertheless returns the time in the local timezone.
func GetNextRuntime(cwf *v1alpha1.CronWorkflow) (time.Time, error) {
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

func checkArgs(cmd *cobra.Command, args []string, parametersFile string, submitOpts *v1alpha1.SubmitOpts) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}

	if parametersFile != "" {
		err := util.ReadParametersFile(parametersFile, submitOpts)
		errors.CheckError(err)
	}
}
