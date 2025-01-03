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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type CronSuite struct {
	fixtures.E2ESuite
}

func (s *CronSuite) SetupSuite() {
	s.E2ESuite.SetupSuite()
	// Since tests run in parallel, delete all cron resources before the test suite is run
	s.E2ESuite.DeleteResources()
}

func (s *CronSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
}

func (s *CronSuite) TearDownSuite() {
	s.E2ESuite.DeleteResources()
	s.E2ESuite.TearDownSuite()
}

func (s *CronSuite) TestBasic() {
	s.Run("TestBasic", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic
spec:
  schedule: "* * * * *"
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
  schedule: "%s"
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
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-suspend
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestResume", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-resume
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestBasicForbid", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-forbid
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestBasicAllow", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-allow
  labels:

spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestBasicReplace", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic-replace
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestSuccessfulJobHistoryLimit", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-succeed-1
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestFailedJobHistoryLimit", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-fail-1
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestStoppingConditionWithSucceeded", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-stop-condition-succeeded
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestStoppingConditionWithFailed", func() {
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-stop-condition-failed
spec:
  schedule: "* * * * *"
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
	})
	s.Run("TestMultipleWithTimezone", func() {
		s.Given().
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
	})
}

func (s *CronSuite) TestMalformedCronWorkflow() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-cronworkflow.yaml"}, fixtures.NoError).
		CronWorkflow("@testdata/wellformed/wellformed-cronworkflow.yaml").
		When().
		CreateCronWorkflow().
		WaitForWorkflow(2*time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Labels[common.LabelKeyCronWorkflow])
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		}).
		ExpectAuditEvents(
			fixtures.HasInvolvedObjectWithName(workflow.CronWorkflowKind, "malformed"),
			1,
			func(t *testing.T, e []corev1.Event) {
				assert.Equal(t, corev1.EventTypeWarning, e[0].Type)
				assert.Equal(t, "Malformed", e[0].Reason)
				assert.Equal(t, "cannot restore slice from map", e[0].Message)
			},
		)
}

func TestCronSuite(t *testing.T) {
	suite.Run(t, new(CronSuite))
}
