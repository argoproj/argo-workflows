package cron

import (
	"testing"
	"time"

	"github.com/argoproj/pkg/humanize"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/util"
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
      arguments: {}
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
	err := yaml.Unmarshal([]byte(scheduledWf), &cronWf)
	if err != nil {
		panic(err)
	}

	// Second value at runtime should be 30-31

	cronWf.Status.LastScheduledTime = &v1.Time{Time: time.Now().Add(-1 * time.Minute)}
	// StartingDeadlineSeconds is after the current second, so cron should be run
	startingDeadlineSeconds := int64(35)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc := &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	proceed, err := woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, proceed)

	// StartingDeadlineSeconds is not after the current second, so cron should not be run
	startingDeadlineSeconds = int64(25)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	proceed, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.False(t, proceed)

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
	proceed, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, proceed)

	// StartingDeadlineSeconds is not after the current second, so cron should not be run
	startingDeadlineSeconds = int64(25)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
		log:    logrus.WithFields(logrus.Fields{}),
	}
	proceed, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.False(t, proceed)
}

type fakeLister struct {
}

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
      arguments: {}
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
	err := yaml.Unmarshal([]byte(invalidWf), &cronWf)
	assert.NoError(t, err)

	cs := fake.NewSimpleClientset()
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset: cs,
		wfClient:    cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:    cs.ArgoprojV1alpha1().CronWorkflows(""),
		wfLister:    &fakeLister{},
		cronWf:      &cronWf,
		log:         logrus.WithFields(logrus.Fields{}),
		metrics:     &testMetrics,
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
    arguments: {}
    entrypoint: whalesay
    templates:
    - arguments: {}
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
	err := yaml.Unmarshal([]byte(specError), &cronWf)
	assert.NoError(t, err)

	cs := fake.NewSimpleClientset()
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset: cs,
		wfClient:    cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:    cs.ArgoprojV1alpha1().CronWorkflows(""),
		wfLister:    &fakeLister{},
		cronWf:      &cronWf,
		log:         logrus.WithFields(logrus.Fields{}),
		metrics:     &testMetrics,
	}

	err = woc.validateCronWorkflow()
	assert.Error(t, err)
	assert.Len(t, woc.cronWf.Status.Conditions, 1)
	submissionErrorCond := woc.cronWf.Status.Conditions[0]
	assert.Equal(t, v1.ConditionTrue, submissionErrorCond.Status)
	assert.Equal(t, v1alpha1.ConditionTypeSpecError, submissionErrorCond.Type)
	assert.Contains(t, submissionErrorCond.Message, "cron schedule is malformed: end of range (12737123) above maximum (12): 12737123")
}

func TestReapplyUpdate(t *testing.T) {
	cronWf := v1alpha1.CronWorkflow{
		ObjectMeta: v1.ObjectMeta{Name: "my-wf"},
		Spec:       v1alpha1.CronWorkflowSpec{Schedule: "* * * * *"},
	}

	cs := fake.NewSimpleClientset(&cronWf)
	testMetrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	woc := &cronWfOperationCtx{
		wfClientset: cs,
		wfClient:    cs.ArgoprojV1alpha1().Workflows(""),
		cronWfIf:    cs.ArgoprojV1alpha1().CronWorkflows(""),
		wfLister:    &fakeLister{},
		cronWf:      &cronWf,
		origCronWf:  cronWf.DeepCopy(),
		name:        cronWf.Name,
		log:         logrus.WithFields(logrus.Fields{}),
		metrics:     &testMetrics,
	}

	cronWf.Spec.Schedule = "1 * * * *"
	err := woc.reapplyUpdate()
	if assert.NoError(t, err) {
		updatedCronWf, err := woc.cronWfIf.Get("my-wf", v1.GetOptions{})
		if assert.NoError(t, err) {
			assert.Equal(t, "1 * * * *", updatedCronWf.Spec.Schedule)
		}
	}
}
