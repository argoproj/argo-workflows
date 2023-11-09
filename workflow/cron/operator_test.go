package cron

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/pkg/humanize"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

var scheduledWf = `
  apiVersion: argoproj.io/v1alpha1
  kind: CronWorkflow
  metadata:
    creationTimestamp: "2020-02-28T18:31:32Z"
    generation: 69
    name: hello-world
    namespace: argo
    resourceVersion: "53389"
    selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/cronworkflows/hello-world
    uid: f230ee83-2ddc-435e-b27c-f0ca63293100
  spec:
    schedule: '* * * * *'
    startingDeadlineSeconds: 30
    workflowSpec:
      entrypoint: whalesay
      templates:
      - container:
          args:
          - "\U0001F553 hello world"
          command:
          - cowsay
          image: docker/whalesay:latest
          name: ""
          resources: {}
        inputs: {}
        metadata: {}
        name: whalesay
        outputs: {}
  status:
    lastScheduledTime: "2020-02-28T19:05:00Z"
`

func TestRunOutstandingWorkflows(t *testing.T) {
	// To ensure consistency, always start at the next 30 second mark
	_, _, sec := time.Now().Clock()
	var toWait time.Duration
	if sec <= 30 {
		toWait = time.Duration(30-sec) * time.Second
	} else {
		toWait = time.Duration(90-sec) * time.Second
	}
	logrus.Infof("Waiting %s to start", humanize.Duration(toWait))
	time.Sleep(toWait)

	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(scheduledWf), &cronWf)

	// Second value at runtime should be 30-31

	cronWf.Status.LastScheduledTime = &v1.Time{Time: time.Now().Add(-1 * time.Minute)}
	// StartingDeadlineSeconds is after the current second, so cron should be run
	startingDeadlineSeconds := int64(35)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc := &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	woc.cronWf.SetSchedule(woc.cronWf.Spec.GetScheduleString())
	missedExecutionTime, err := woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	// The missedExecutionTime should be the last complete minute mark, which we can get with inferScheduledTime
	assert.Equal(t, inferScheduledTime().Unix(), missedExecutionTime.Unix())

	// StartingDeadlineSeconds is not after the current second, so cron should not be run
	startingDeadlineSeconds = int64(25)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	missedExecutionTime, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, missedExecutionTime.IsZero())

	// Same test, but simulate a change to the schedule immediately prior by setting a different last-used-schedule annotation
	// In this case, since a schedule change is detected, not workflow should be run
	woc.cronWf.SetSchedule("0 * * * *")
	missedExecutionTime, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, missedExecutionTime.IsZero())

	// Run the same test in a different timezone
	testTimezone := "Pacific/Niue"
	testLocation, err := time.LoadLocation(testTimezone)
	if err != nil {
		panic(err)
	}
	cronWf.Spec.Timezone = testTimezone
	cronWf.Status.LastScheduledTime = &v1.Time{Time: cronWf.Status.LastScheduledTime.In(testLocation)}

	// StartingDeadlineSeconds is after the current second, so cron should be run
	startingDeadlineSeconds = int64(35)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	// Reset last-used-schedule as if the current schedule has been used before
	woc.cronWf.SetSchedule(woc.cronWf.Spec.GetScheduleString())
	missedExecutionTime, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	// The missedExecutionTime should be the last complete minute mark, which we can get with inferScheduledTime
	assert.Equal(t, inferScheduledTime().Unix(), missedExecutionTime.Unix())

	// StartingDeadlineSeconds is not after the current second, so cron should not be run
	startingDeadlineSeconds = int64(25)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	missedExecutionTime, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, missedExecutionTime.IsZero())

	// Same test, but simulate a change to the schedule immediately prior by setting a different last-used-schedule annotation
	// In this case, since a schedule change is detected, not workflow should be run
	woc.cronWf.SetSchedule("0 * * * *")
	missedExecutionTime, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, missedExecutionTime.IsZero())
}

type fakeLister struct{}

func (f fakeLister) List() ([]*v1alpha1.Workflow, error) {
	// Do nothing
	return nil, nil
}

var _ util.WorkflowLister = &fakeLister{}

var invalidWf = `
  apiVersion: argoproj.io/v1alpha1
  kind: CronWorkflow
  metadata:
    name: hello-world
  spec:
    schedule: '* * * * *'
    startingDeadlineSeconds: 30
    workflowSpec:
      entrypoint: whalesay
      templates:
      - container:
          args:
          - "\U0001F553 hello world"
          command:
          - cowsay
          image: docker/whalesay:latest
          name: ""
          resources: {}
        inputs: {}
        metadata: {}
        name: "bad template name"
        outputs: {}
`

func TestCronWorkflowConditionSubmissionError(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(invalidWf), &cronWf)

	cs := fake.NewSimpleClientset()
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset:       cs,
		wfClient:          cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:          cs.ArgoprojV1alpha1().CronWorkflows(""),
		cronWf:            &cronWf,
		log:               logrus.WithFields(logrus.Fields{}),
		metrics:           testMetrics,
		scheduledTimeFunc: inferScheduledTime,
	}
	woc.Run()

	assert.Len(t, woc.cronWf.Status.Conditions, 1)
	submissionErrorCond := woc.cronWf.Status.Conditions[0]
	assert.Equal(t, v1.ConditionTrue, submissionErrorCond.Status)
	assert.Equal(t, v1alpha1.ConditionTypeSpecError, submissionErrorCond.Type)
	assert.Contains(t, submissionErrorCond.Message, "'bad template name' is invalid")
}

var specError = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
spec:
  concurrencyPolicy: Replace
  failedJobsHistoryLimit: 4
  schedule: 10 * * 12737123 *
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  timezone: America/Los_Angeles
  workflowSpec:
    entrypoint: whalesay
    templates:
    - 
      container:
        args:
        - "\U0001F553 hello world"
        command:
        - cowsay
        image: docker/whalesay:latest
        name: ""
        resources: {}
      inputs: {}
      metadata: {}
      name: whalesay
      outputs: {}
`

func TestSpecError(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(specError), &cronWf)

	cs := fake.NewSimpleClientset()
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset: cs,
		wfClient:    cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:    cs.ArgoprojV1alpha1().CronWorkflows(""),
		cronWf:      &cronWf,
		log:         logrus.WithFields(logrus.Fields{}),
		metrics:     testMetrics,
	}

	err := woc.validateCronWorkflow()
	assert.Error(t, err)
	assert.Len(t, woc.cronWf.Status.Conditions, 1)
	submissionErrorCond := woc.cronWf.Status.Conditions[0]
	assert.Equal(t, v1.ConditionTrue, submissionErrorCond.Status)
	assert.Equal(t, v1alpha1.ConditionTypeSpecError, submissionErrorCond.Type)
	assert.Contains(t, submissionErrorCond.Message, "cron schedule is malformed: end of range (12737123) above maximum (12): 12737123")
}

func TestScheduleTimeParam(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(scheduledWf), &cronWf)

	cs := fake.NewSimpleClientset()
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset:       cs,
		wfClient:          cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:          cs.ArgoprojV1alpha1().CronWorkflows(""),
		cronWf:            &cronWf,
		log:               logrus.WithFields(logrus.Fields{}),
		metrics:           testMetrics,
		scheduledTimeFunc: inferScheduledTime,
	}
	woc.Run()
	wsl, err := cs.ArgoprojV1alpha1().Workflows("").List(context.Background(), v1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wsl.Items.Len(), 1)
	wf := wsl.Items[0]
	assert.NotNil(t, wf)
	assert.Len(t, wf.GetAnnotations(), 1)
	assert.NotEmpty(t, wf.GetAnnotations()[common.AnnotationKeyCronWfScheduledTime])
}

const lastUsedSchedule = `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test
spec:
  concurrencyPolicy: Forbid
  failedJobsHistoryLimit: 1
  schedule: 41 12 * * *
  successfulJobsHistoryLimit: 1
  timezone: America/New_York
  workflowSpec:
    arguments: {}
    entrypoint: job
    templates:
    - container:
        args:
        - /bin/echo "hello argo"
        command:
        - /bin/sh
        - -c
        image: alpine
        imagePullPolicy: Always
      name: job
`

func TestLastUsedSchedule(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(lastUsedSchedule), &cronWf)

	cs := fake.NewSimpleClientset()
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset:       cs,
		wfClient:          cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:          cs.ArgoprojV1alpha1().CronWorkflows(""),
		cronWf:            &cronWf,
		log:               logrus.WithFields(logrus.Fields{}),
		metrics:           testMetrics,
		scheduledTimeFunc: inferScheduledTime,
	}

	missedExecutionTime, err := woc.shouldOutstandingWorkflowsBeRun()
	if assert.NoError(t, err) {
		assert.Equal(t, time.Time{}, missedExecutionTime)
	}

	woc.cronWf.SetSchedule(woc.cronWf.Spec.GetScheduleString())

	if assert.NotNil(t, woc.cronWf.Annotations) {
		assert.Equal(t, woc.cronWf.Spec.GetScheduleString(), woc.cronWf.GetLatestSchedule())
	}
}

var forbidMissedSchedule = `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  annotations:
    cronworkflows.argoproj.io/last-used-schedule: CRON_TZ=America/Los_Angeles 0-36/1
      21-22 * * *
  creationTimestamp: "2022-02-04T05:33:24Z"
  generation: 2
  name: hello-world
  namespace: argo
  resourceVersion: "341102"
  uid: 9ac888d8-95e3-4f93-8983-0d46c6c7d62a
spec:
  concurrencyPolicy: Forbid
  failedJobsHistoryLimit: 4
  schedule: 0-36/1 21-22 * * *
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  timezone: America/Los_Angeles
  workflowSpec:
    arguments: {}
    entrypoint: whalesay
    templates:
    - container:
        args:
        - sleep 600
        command:
        - sh
        - -c
        image: alpine:3.6
        name: ""
        resources: {}
      inputs: {}
      metadata: {}
      name: whalesay
      outputs: {}
status:
  active:
  - apiVersion: argoproj.io/v1alpha1
    kind: Workflow
    name: hello-world-1643952840
    namespace: argo
    resourceVersion: "341101"
    uid: c56a8f98-ff46-4815-9d6f-d9db5cfcd941
  lastScheduledTime: "2022-02-04T05:34:00Z"
`

func TestMissedScheduleAfterCronScheduleWithForbid(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(forbidMissedSchedule), &cronWf)
	// StartingDeadlineSeconds is after the current second, so cron should be run
	//startingDeadlineSeconds := int64(35)
	//cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	t.Run("ForbiddenWithMissedScheduleAfterCron", func(t *testing.T) {
		cronWf.Spec.StartingDeadlineSeconds = nil
		woc := &cronWfOperationCtx{
			cronWf: &cronWf,
			log:    logrus.WithFields(logrus.Fields{}),
		}
		woc.cronWf.SetSchedule(woc.cronWf.Spec.GetScheduleString())
		missedExecutionTime, err := woc.shouldOutstandingWorkflowsBeRun()
		assert.NoError(t, err)
		assert.True(t, missedExecutionTime.IsZero())
	})
}
