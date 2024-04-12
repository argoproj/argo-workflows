package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestCronWorkflowStatus_HasActiveUID(t *testing.T) {
	cwfStatus := CronWorkflowStatus{
		Active: []v1.ObjectReference{{UID: "123"}},
	}

	assert.True(t, cwfStatus.HasActiveUID("123"))
	assert.False(t, cwfStatus.HasActiveUID("foo"))
}

func TestCronWorkflowSpec_GetScheduleString(t *testing.T) {
	cwfSpec := CronWorkflowSpec{
		Timezone: "",
		Schedule: "* * * * *",
	}

	assert.Equal(t, "* * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	assert.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *", cwfSpec.GetScheduleString())
	cwfSpec = CronWorkflowSpec{
		Timezone:  "",
		Schedules: []string{"* * * * *", "0 * * * *"},
	}
	assert.Equal(t, "* * * * *,0 * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	assert.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *,CRON_TZ=America/Los_Angeles 0 * * * *", cwfSpec.GetScheduleString())
}
