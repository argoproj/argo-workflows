// +build e2e

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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/workflow/common"
)

type CronSuite struct {
	fixtures.E2ESuite
}

func (s *CronSuite) SetupSuite() {
	s.E2ESuite.SetupSuite()
	// Since tests run in parallel, delete all cron resources before the test suite is run
	s.E2ESuite.DeleteResources()
}

func (s *CronSuite) BeforeTest(string, string) {
	// noop
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]`).
			When().
			CreateCronWorkflow().
			Wait(1 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
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
  labels:
    argo-e2e: true
spec:
  schedule: "%s"
  timezone: "%s"
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]`, scheduleInTestTimezone, testTimezone)).
			When().
			CreateCronWorkflow().
			Wait(1 * time.Minute).
			Then().
			ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]`).
			When().
			CreateCronWorkflow().
			Then().
			RunCli([]string{"cron", "suspend", "test-cron-wf-basic-suspend"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "CronWorkflow 'test-cron-wf-basic-suspend' suspended")
			}).ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]`).
			When().
			CreateCronWorkflow().
			Then().
			RunCli([]string{"cron", "resume", "test-cron-wf-basic-resume"}, func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "CronWorkflow 'test-cron-wf-basic-resume' resumed")
			}).ExpectCron(func(t *testing.T, cronWf *wfv1.CronWorkflow) {
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Forbid"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["sleep 300"]`).
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
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["sleep 300"]`).
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Replace"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["sleep 300"]`).
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Forbid"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 1
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["python", -c]
          args: ["import random; import sys; exit_code = random.choice([0]); print('exiting with code {}'.format(exit_code)); sys.exit(exit_code)"] `).
			When().
			CreateCronWorkflow().
			Wait(2*time.Minute).
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
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Forbid"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 1
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["python", -c]
          args: ["import random; import sys; exit_code = random.choice([1]); print('exiting with code {}'.format(exit_code)); sys.exit(exit_code)"] `).
			When().
			CreateCronWorkflow().
			Wait(2*time.Minute).
			Then().
			ExpectWorkflowList(listOptions, func(t *testing.T, wfList *wfv1.WorkflowList) {
				assert.Equal(t, 1, len(wfList.Items))
				assert.True(t, wfList.Items[0].Status.FinishedAt.Time.After(time.Now().Add(-1*time.Minute)))
			})
	})
}

func wfInformerListOptionsFunc(options *v1.ListOptions, cronWfName string) {
	options.FieldSelector = fields.Everything().String()
	isCronWorkflowChildReq, err := labels.NewRequirement(common.LabelKeyCronWorkflow, selection.Equals, []string{cronWfName})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*isCronWorkflowChildReq)
	options.LabelSelector = labelSelector.String()
}

func TestCronSuite(t *testing.T) {
	if testing.Short() {
		log.Infof("Skipping CronSuite because --short flag is enabled")
		t.SkipNow()
	}
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
