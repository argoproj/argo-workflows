package cron

import (
	"testing"
	"time"

	"github.com/argoproj/pkg/humanize"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
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
	}
	proceed, err := woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, proceed)

	// StartingDeadlineSeconds is not after the current second, so cron should not be run
	startingDeadlineSeconds = int64(25)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
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
	}
	proceed, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.True(t, proceed)

	// StartingDeadlineSeconds is not after the current second, so cron should not be run
	startingDeadlineSeconds = int64(25)
	cronWf.Spec.StartingDeadlineSeconds = &startingDeadlineSeconds
	woc = &cronWfOperationCtx{
		cronWf: &cronWf,
	}
	proceed, err = woc.shouldOutstandingWorkflowsBeRun()
	assert.NoError(t, err)
	assert.False(t, proceed)
}
