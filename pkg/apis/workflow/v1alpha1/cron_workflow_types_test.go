package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCronWorkflowStatus_HasActiveUID(t *testing.T) {
	cwfStatus := CronWorkflowStatus{
		Active: []v1.ObjectReference{{UID: "123"}},
	}

	require.True(t, cwfStatus.HasActiveUID("123"))
	require.False(t, cwfStatus.HasActiveUID("foo"))
}

func TestCronWorkflowSpec_GetScheduleStrings(t *testing.T) {
	cwfSpec := CronWorkflowSpec{
		Timezone: "",
		Schedule: "* * * * *",
	}

	require.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedules())
	require.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedulesWithTimezone())
	require.Equal(t, "* * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	require.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedules())
	require.Equal(t, []string{"CRON_TZ=America/Los_Angeles * * * * *"}, cwfSpec.GetSchedulesWithTimezone())
	require.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *", cwfSpec.GetScheduleString())

	cwfSpec = CronWorkflowSpec{
		Timezone:  "",
		Schedules: []string{"* * * * *", "0 * * * *"},
	}
	require.Equal(t, "* * * * *,0 * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	require.Equal(t, []string{"* * * * *", "0 * * * *"}, cwfSpec.GetSchedules())
	require.Equal(t, []string{"CRON_TZ=America/Los_Angeles * * * * *", "CRON_TZ=America/Los_Angeles 0 * * * *"}, cwfSpec.GetSchedulesWithTimezone())
	require.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *,CRON_TZ=America/Los_Angeles 0 * * * *", cwfSpec.GetScheduleString())
}
