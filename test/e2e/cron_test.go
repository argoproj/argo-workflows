//go:build cron
// +build cron

package e2e

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/argoproj/pkg/humanize"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
		s.T().Parallel()
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
			When().
			CreateCronWorkflow().
			Wait(1 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, cronWf.Spec.GetScheduleString(), cronWf.GetLatestSchedule())
				assert.True(t, cronWf.Status.LastScheduledTime.Time.After(time.Now().Add(-1*time.Minute)))
			})
	})
	s.Run("TestBasicTimezone", func() {
		s.T().Parallel()
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
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`, scheduleInTestTimezone, testTimezone)).
			When().
			CreateCronWorkflow().
			Wait(1 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, cronWf.Spec.GetScheduleString(), cronWf.GetLatestSchedule())
				assert.True(t, cronWf.Status.LastScheduledTime.Time.After(time.Now().Add(-1*time.Minute)))
			})
	})
	s.Run("TestSuspend", func() {
		s.T().Parallel()
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
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
		s.T().Parallel()
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
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
		s.T().Parallel()
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
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
			Wait(2 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, 1, len(cronWf.Status.Active))
				assert.True(t, cronWf.Status.LastScheduledTime.Time.Before(time.Now().Add(-1*time.Minute)))
			})
	})
	s.Run("TestBasicAllow", func() {
		s.T().Parallel()
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
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
			Wait(2 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, 2, len(cronWf.Status.Active))
			})
	})
	s.Run("TestBasicReplace", func() {
		s.T().Parallel()
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
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
			Wait(2*time.Minute + 20*time.Second).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
				assert.Equal(t, 1, len(cronWf.Status.Active))
				if assert.NotNil(t, cronWf.Status.LastScheduledTime) {
					assert.True(t, cronWf.Status.LastScheduledTime.Time.After(time.Now().Add(-1*time.Minute)))
				}
			})
	})
	s.Run("TestSuccessfulJobHistoryLimit", func() {
		s.T().Parallel()
		var listOptions v1.ListOptions
		wfInformerListOptionsFunc(&listOptions, "test-cron-wf-succeed-1")
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2`).
			When().
			CreateCronWorkflow().
			Wait(2*time.Minute+25*time.Second).
			Then().
			ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
				assert.Equal(t, 1, len(wfList.Items))
				assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1*time.Minute)))
			})
	})
	s.Run("TestFailedJobHistoryLimit", func() {
		s.T().Parallel()
		var listOptions v1.ListOptions
		wfInformerListOptionsFunc(&listOptions, "test-cron-wf-fail-1")
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
  workflowSpec:
    metadata:
      labels:
        workflows.argoproj.io/test: "true"
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
			Wait(2*time.Minute+25*time.Second).
			Then().
			ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
				assert.Equal(t, 1, len(wfList.Items))
				assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1*time.Minute)))
			})
	})
}

func wfInformerListOptionsFunc(options *v1.ListOptions, cronWfName string) {
	options.LabelSelector = common.LabelKeyCronWorkflow + "=" + cronWfName
}

func (s *CronSuite) TestMalformedCronWorkflow() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-cronworkflow.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-cronworkflow.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(1*time.Minute+15*time.Second).
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
	// To ensure consistency, always start at the next 30 second mark
	_, _, sec := time.Now().Clock()
	var toWait time.Duration
	if sec <= 30 {
		toWait = time.Duration(30-sec) * time.Second
	} else {
		toWait = time.Duration(90-sec) * time.Second
	}
	log.Infof("Waiting %s to start", humanize.Duration(toWait))
	time.Sleep(toWait)
	suite.Run(t, new(CronSuite))
}
