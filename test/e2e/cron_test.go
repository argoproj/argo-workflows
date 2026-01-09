//go:build cron

package e2e

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestBasic(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic
spec:
  schedules:
    -  "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Equal(t, cronWf.Spec.GetScheduleWithTimezoneString(), cronWf.GetLatestSchedule())
			assert.Greater(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
		})
}

func TestBasicTimezone(t *testing.T) {
	// This test works by scheduling a CronWorkflow for the next minute, but using the local time of another timezone
	// then seeing if the Workflow was ran within the next minute. Since this test would be trivial if the selected
	// timezone was the same as the local timezone, a little-used timezone is used.
	testTimezone := "Pacific/Niue"
	testLocation, err := time.LoadLocation(testTimezone)
	fixtures.CheckError(t, err)
	hour, minute, _ := time.Now().In(testLocation).Clock()
	minute++
	if minute == 60 {
		minute = 0
		hour = (hour + 1) % 24
	}
	scheduleInTestTimezone := strconv.Itoa(minute) + " " + strconv.Itoa(hour) + " * * *"
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
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
}

func TestSuspend(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-suspend
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
		When().
		CreateCronWorkflow().
		SuspendCronWorkflow().
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.True(t, cronWf.Spec.Suspend)
		})
}

func TestResume(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-resume
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
		When().
		CreateCronWorkflow().
		ResumeCronWorkflow("test-cron-wf-basic-resume").
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.False(t, cronWf.Spec.Suspend)
		})
}

func TestBasicForbid(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-forbid
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Forbid"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2
          args: ["sleep", "300s"]`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Wait(time.Minute). // wait for next scheduled time to ensure it's skipped by concurrencyPolicy
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Len(t, cronWf.Status.Active, 1)
			assert.Less(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
		})
}
func TestBasicAllow(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-allow
  labels:

spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2
          args: ["sleep", "300s"]`).
		When().
		CreateCronWorkflow().
		WaitForWorkflowListCount(3*time.Minute, 2).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Len(t, cronWf.Status.Active, 2)
		})
}

func TestBasicReplace(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-replace
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Replace"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2
          args: ["sleep", "300s"]`).
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
}

func TestSuccessfulJobHistoryLimit(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-succeed-1
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Forbid"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 1
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
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
}

func TestFailedJobHistoryLimit(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-fail-1
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Forbid"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 1
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2
          args: ["exit", "1"]`).
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
}

func TestStoppingConditionWithSucceeded(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-stop-condition-succeeded
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  stopStrategy:
    expression: "cronworkflow.succeeded >= 1"
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
            image: argoproj/argosay:v2
            command: [/argosay]`).
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
}
func TestStoppingConditionWithFailed(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-stop-condition-failed
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Allow"
  stopStrategy:
    expression: "cronworkflow.failed >= 1"
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2
          args: ["exit", "1"]`).
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
}

func TestMultipleWithTimezone(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().
		CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-multiple-with-timezone
spec:
  schedules:
    - "* * * * *"
    - "0 1 * * *"
  timezone: "America/Los_Angeles"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
		When().
		CreateCronWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
			assert.Equal(t, cronWf.Spec.GetScheduleWithTimezoneString(), cronWf.GetLatestSchedule())
			assert.Greater(t, cronWf.Status.LastScheduledTime.Time, time.Now().Add(-1*time.Minute))
		})
}

func TestMalformedCronWorkflow(t *testing.T) {
	runner := fixtures.NewRunner(t, fixtures.WithTestResourceCleanupEnabled(true))
	runner.Given().KubectlApply("testdata/malformed/malformed-cronworkflow.yaml", fixtures.ErrorOutput(".spec.workflowSpec.arguments.parameters: expected list"))
}
