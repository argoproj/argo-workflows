package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Timezone:  "",
		Schedules: []string{"* * * * *"},
	}
	assert.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedules())
	assert.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedulesWithTimezone())
	assert.Equal(t, "* * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	assert.Equal(t, []string{"* * * * *"}, cwfSpec.GetSchedules())
	assert.Equal(t, []string{"CRON_TZ=America/Los_Angeles * * * * *"}, cwfSpec.GetSchedulesWithTimezone())
	assert.Equal(t, "* * * * *", cwfSpec.GetScheduleString())
	assert.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *", cwfSpec.GetScheduleWithTimezoneString())

	cwfSpec = CronWorkflowSpec{
		Timezone:  "",
		Schedules: []string{"* * * * *", "0 * * * *"},
	}
	assert.Equal(t, "* * * * *,0 * * * *", cwfSpec.GetScheduleString())

	cwfSpec.Timezone = "America/Los_Angeles"
	assert.Equal(t, []string{"* * * * *", "0 * * * *"}, cwfSpec.GetSchedules())
	assert.Equal(t, []string{"CRON_TZ=America/Los_Angeles * * * * *", "CRON_TZ=America/Los_Angeles 0 * * * *"}, cwfSpec.GetSchedulesWithTimezone())
	assert.Equal(t, "* * * * *,0 * * * *", cwfSpec.GetScheduleString())
	assert.Equal(t, "CRON_TZ=America/Los_Angeles * * * * *,CRON_TZ=America/Los_Angeles 0 * * * *", cwfSpec.GetScheduleWithTimezoneString())
}

func TestCronWorkflow_GetSchedulesResolvesHash(t *testing.T) {
	cwf := CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cron-wf", Namespace: "argo"},
		Spec: CronWorkflowSpec{
			Schedules: []string{"H H * * *", "0 * * * *"},
		},
	}
	// the spec is never rewritten, only the schedules handed to the cron engine are
	assert.Equal(t, []string{"H H * * *", "0 * * * *"}, cwf.Spec.GetSchedules())
	assert.Equal(t, []string{"9 6 * * *", "0 * * * *"}, cwf.GetSchedules())

	cwf.Spec.Timezone = "America/Los_Angeles"
	assert.Equal(t, []string{"CRON_TZ=America/Los_Angeles 9 6 * * *", "CRON_TZ=America/Los_Angeles 0 * * * *"}, cwf.GetSchedulesWithTimezone())

	// the hash only depends on name and namespace, so it never changes for a CronWorkflow
	other := cwf
	other.Name = "other-cron-wf"
	assert.NotEqual(t, cwf.GetSchedules(), other.GetSchedules())

	// malformed schedules are left to the cron parser to report
	malformed := CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cron-wf", Namespace: "argo"},
		Spec:       CronWorkflowSpec{Schedules: []string{"H(30-10) * * * *"}},
	}
	assert.Equal(t, []string{"H(30-10) * * * *"}, malformed.GetSchedules())
}

func TestCronWorkflow_SetResolvedSchedules(t *testing.T) {
	cwf := CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cron-wf", Namespace: "argo"},
		Spec:       CronWorkflowSpec{Schedules: []string{"H H * * *"}},
	}
	assert.Empty(t, cwf.Status.ResolvedSchedules)

	assert.True(t, cwf.SetResolvedSchedules())
	assert.Equal(t, map[string]string{"H H * * *": "9 6 * * *"}, cwf.Status.ResolvedSchedules)

	// unchanged schedules do not need to be persisted again
	assert.False(t, cwf.SetResolvedSchedules())

	// the schedules are keyed by what is configured, and only those using a `H` are listed
	cwf.Spec.Schedules = []string{"H,30 * * * *", "0 9 * * *"}
	assert.True(t, cwf.SetResolvedSchedules())
	assert.Equal(t, map[string]string{"H,30 * * * *": "9,30 * * * *"}, cwf.Status.ResolvedSchedules)

	// a CronWorkflow which does not use a `H` at all keeps an empty status
	plain := CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cron-wf", Namespace: "argo"},
		Spec:       CronWorkflowSpec{Schedules: []string{"0 9 * * *"}},
	}
	assert.False(t, plain.SetResolvedSchedules())
	assert.Empty(t, plain.Status.ResolvedSchedules)
}
