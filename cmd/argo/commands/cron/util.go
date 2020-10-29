package cron

import (
	"time"

	"github.com/robfig/cron/v3"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// GetNextRuntime returns the next time the workflow should run in local time. It assumes the workflow-controller is in
// UTC, but nevertheless returns the time in the local timezone.
func GetNextRuntime(cwf *v1alpha1.CronWorkflow) (time.Time, error) {
	cronScheduleString := cwf.Spec.Schedule
	if cwf.Spec.Timezone != "" {
		cronScheduleString = "CRON_TZ=" + cwf.Spec.Timezone + " " + cronScheduleString
	}
	cronSchedule, err := cron.ParseStandard(cronScheduleString)
	if err != nil {
		return time.Time{}, err
	}
	return cronSchedule.Next(time.Now().UTC()).Local(), nil
}
