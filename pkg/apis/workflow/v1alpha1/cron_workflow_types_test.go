package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestCronWorkflowStatus_HasActiveUID(t *testing.T) {
	cwfStatus := CronWorkflowStatus{
		Active: []v1.ObjectReference{{UID: "123"}},
	}

	assert.True(t, cwfStatus.HasActiveUID("123"))
	assert.False(t, cwfStatus.HasActiveUID("foo"))
}

func TestCronWorkflowSpec_GetScheduleStrings(t *testing.T) {
	cwfSpec := CronWorkflowSpec{
		Timezone: "",
		Schedule: "* * * * *",
	}
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	assert.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedules(ctx))
	assert.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedulesWithTimezone(ctx))
	assert.Equal(t, "* * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	assert.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedules(ctx))
	assert.Equal(t, []string{"CRON_TZ=America/Los_Angeles * * * * *"}, cwfSpec.GetSchedulesWithTimezone(ctx))
	assert.Equal(t, "* * * * *", cwfSpec.GetScheduleString())
	assert.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *", cwfSpec.GetScheduleWithTimezoneString())

	cwfSpec = CronWorkflowSpec{
		Timezone:  "",
		Schedules: []string{"* * * * *", "0 * * * *"},
	}
	assert.Equal(t, "* * * * *,0 * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	assert.Equal(t, []string{"* * * * *", "0 * * * *"}, cwfSpec.GetSchedules(ctx))
	assert.Equal(t, []string{"CRON_TZ=America/Los_Angeles * * * * *", "CRON_TZ=America/Los_Angeles 0 * * * *"}, cwfSpec.GetSchedulesWithTimezone(ctx))
	assert.Equal(t, "* * * * *,0 * * * *", cwfSpec.GetScheduleString())
	assert.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *,CRON_TZ=America/Los_Angeles 0 * * * *", cwfSpec.GetScheduleWithTimezoneString())
}
