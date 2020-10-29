package cron

import (
	"time"

	"github.com/robfig/cron/v3"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// GetNextRuntime assumes the workflow-controller is in UTC, unless an overriding timezone is available
func GetNextRuntime(cwf *v1alpha1.CronWorkflow) (time.Time, error) {
	cronScheduleString := cwf.Spec.Schedule
	if cwf.Spec.Timezone != "" {
		cronScheduleString = "CRON_TZ=" + cwf.Spec.Timezone + " " + cronScheduleString
	}
	cronSchedule, err := cron.ParseStandard(cronScheduleString)
	if err != nil {
		return time.Time{}, err
	}
	return cronSchedule.Next(time.Now().UTC()), nil
}
