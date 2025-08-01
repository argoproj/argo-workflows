//go:build cron

package e2e

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type CronSuite struct {
	fixtures.E2ESuite
}

func (s *CronSuite) TearDownSubTest() {
	// Delete all CronWorkflows after every subtest (as opposed to after each
	// test, which is the default) to avoid workflows from accumulating and
	// causing intermittent failures due to the controller reaching the
	// parallelism limit. When that happens, workflows can be postponed long
	// enough to reach the test timeout, and you'll see the following in the logs:
	//    time="2025-02-23T06:11:00.023Z" level=info msg="Workflow processing has been postponed due to max parallelism limit" key=argo/test-cron-wf-succeed-1-1740291060
	//    time="2025-02-23T06:11:00.023Z" level=info msg="Updated phase  -> Pending" namespace=argo workflow=test-cron-wf-succeed-1-1740291060
	//    time="2025-02-23T06:11:00.023Z" level=info msg="Updated message  -> Workflow processing has been postponed because too many workflows are already running" namespace=argo workflow=test-cron-wf-succeed-1-1740291060
	s.DeleteResources()
}

func (s *CronSuite) TestBasic() {
	s.Run("TestBasic", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-basic.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, cronWf.Spec.GetScheduleWithTimezoneString(), cronWf.GetLatestSchedule())
				assert.Greater(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
			})
	})
	s.Run("TestBasicTimezone", func() {
		// This test works by scheduling a CronWorkflow for the next minute, but using the local time of another timezone
		// then seeing if the Workflow was ran within the next minute. Since this test would be trivial if the selected
		// timezone was the same as the local timezone, a little-used timezone is used.
		testTimezone := "Pacific/Niue"
		testLocation, err := time.LoadLocation(testTimezone)
		s.CheckError(err)
		hour, min, _ := time.Now().In(testLocation).Clock()
		min++
		if min == 60 {
			min = 0
			hour = (hour + 1) % 24
		}
		scheduleInTestTimezone := strconv.Itoa(min) + " " + strconv.Itoa(hour) + " * * *"
		s.Given().
			CronWorkflow(fmt.Sprintf(`
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-timezone
spec:
  schedules:
    - "%s"
  timezone: "%s"
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`, scheduleInTestTimezone, testTimezone)).
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, cronWf.Spec.GetScheduleWithTimezoneString(), cronWf.GetLatestSchedule())
				assert.Greater(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
			})
	})
	s.Run("TestSuspend", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-basic-suspend.yaml").
			When().
			CreateCronWorkflow().
			SuspendCronWorkflow().
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.True(t, cronWf.Spec.Suspend)
			})
	})
	s.Run("TestResume", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-basic-resume.yaml").
			When().
			CreateCronWorkflow().
			ResumeCronWorkflow("test-cron-wf-basic-resume").
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.False(t, cronWf.Spec.Suspend)
			})
	})
	s.Run("TestBasicForbid", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-basic-forbid.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeRunning).
			Wait(time.Minute). // wait for next scheduled time to ensure it's skipped by concurrencyPolicy
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Len(t, cronWf.Status.Active, 1)
				assert.Less(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
			})
	})
	s.Run("TestBasicAllow", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-basic-allow.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflowListCount(3*time.Minute, 2).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Len(t, cronWf.Status.Active, 2)
			})
	})
	s.Run("TestBasicReplace", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-basic-replace.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeRunning).
			WaitForNewWorkflow(fixtures.ToBeRunning).
			WaitForCronWorkflow().
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Len(t, cronWf.Status.Active, 1)
				require.NotNil(t, cronWf.Status.LastScheduledTime)
				assert.Greater(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
			})
	})
	s.Run("TestSuccessfulJobHistoryLimit", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-succeed-1.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			WaitForNewWorkflow(fixtures.ToBeRunning).
			WaitForWorkflowListCount(2*time.Minute, 1).
			Then().
			ExpectWorkflowListFromCronWorkflow(func(t *testing.T, wfList *wfv1.WorkflowList) {
				assert.Len(t, wfList.Items, 1)
				assert.Greater(t, wfList.Items[0].Status.FinishedAt.Time, time.Now().Add(-1*time.Minute))
			})
	})
	s.Run("TestFailedJobHistoryLimit", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-fail-1.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeFailed).
			WaitForNewWorkflow(fixtures.ToBeRunning).
			WaitForWorkflowListCount(2*time.Minute, 1).
			Then().
			ExpectWorkflowListFromCronWorkflow(func(t *testing.T, wfList *wfv1.WorkflowList) {
				assert.Len(t, wfList.Items, 1)
				assert.Greater(t, wfList.Items[0].Status.FinishedAt.Time, time.Now().Add(-1*time.Minute))
			})
	})
	s.Run("TestStoppingConditionWithSucceeded", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-stop-condition-succeeded.yaml").
			When().
			CreateCronWorkflow().
			WaitForCronWorkflowCompleted(3 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, int64(0), cronWf.Status.Failed)
				assert.Equal(t, int64(1), cronWf.Status.Succeeded)
				assert.Equal(t, wfv1.StoppedPhase, cronWf.Status.Phase)
				assert.Equal(t, "true", cronWf.Labels[common.LabelKeyCronWorkflowCompleted])
			})
	})
	s.Run("TestStoppingConditionWithFailed", func() {
		s.Given().
			CronWorkflow("@cron/test-cron-wf-stop-condition-failed.yaml").
			When().
			CreateCronWorkflow().
			WaitForCronWorkflowCompleted(3 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, int64(0), cronWf.Status.Succeeded)
				assert.Equal(t, int64(1), cronWf.Status.Failed)
				assert.Equal(t, wfv1.StoppedPhase, cronWf.Status.Phase)
				assert.Equal(t, "true", cronWf.Labels[common.LabelKeyCronWorkflowCompleted])
			})
	})
	s.Run("TestMultipleWithTimezone", func() {
		s.Given().
			CronWorkflow("@cron/test-multiple-with-timezone.yaml").
			When().
			CreateCronWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, cronWf.Spec.GetScheduleWithTimezoneString(), cronWf.GetLatestSchedule())
				assert.Greater(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
			})
	})
}

func (s *CronSuite) TestMalformedCronWorkflow() {
	s.Given().KubectlApply("testdata/malformed/malformed-cronworkflow.yaml", fixtures.ErrorOutput(".spec.workflowSpec.arguments.parameters: expected list"))
}

func TestCronSuite(t *testing.T) {
	suite.Run(t, new(CronSuite))
}
