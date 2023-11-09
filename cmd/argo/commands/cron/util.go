package cron

import (
	"time"

	"github.com/robfig/cron/v3"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// GetNextRuntime returns the next time the workflow should run in local time. It assumes the workflow-controller is in
// UTC, but nevertheless returns the time in the local timezone.
func GetNextRuntime(cwf *v1alpha1.CronWorkflow) (time.Time, error) {
	cronSchedule, err := cron.ParseStandard(cwf.Spec.GetScheduleString())
	if err != nil {
		return time.Time{}, err
	}
	return cronSchedule.Next(time.Now().UTC()).Local(), nil
}
