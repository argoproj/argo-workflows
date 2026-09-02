package cron

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var invalidCwf = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  creationTimestamp: "2020-05-19T16:47:25Z"
  generation: 98
  name: wonderful-tiger
  namespace: argo
  resourceVersion: "465179"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/cronworkflows/wonderful-tiger
  uid: c4ea2e84-ec58-4638-bf1d-5d543e7cc86a
spec:
  schedules:
    - '* * * * *'
  workflowSpec:
    entrypoint: argosay
    templates:
    -
      container:
        args:
        - echo
        - hello argo!
        command:
        - /argosay
        image: argoproj/argosay:v2
        name: main
        resources: {}
      inputs: {}
      metadata: {}
      name: argosay!3
      outputs: {}
status:
  conditions:
  - message: 'Failed to submit Workflow: spec.templates[0].name: ''argosay!3'' is
      invalid: name must consist of alpha-numeric characters or ''-'', and must start
      with an alpha-numeric character (e.g. My-name1-2, 123-NAME)'
    status: "True"
    type: SubmissionError
  lastScheduledTime: "2020-05-19T17:56:00Z"
`

var expectedOut = `
Conditions:                    
✖ SubmissionError              Failed to submit Workflow: spec.templates[0].name: 'argosay!3' is invalid: name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character (e.g. My-name1-2, 123-NAME)`

func TestPrintCronWorkflow(t *testing.T) {
	var cronWf = v1alpha1.MustUnmarshalCronWorkflow(invalidCwf)
	ctx := logging.TestContext(t.Context())
	out := getCronWorkflowGet(ctx, cronWf)
	assert.Contains(t, out, expectedOut)
}

func TestNextRuntime(t *testing.T) {
	var cronWf = v1alpha1.MustUnmarshalCronWorkflow(invalidCwf)
	ctx := logging.TestContext(t.Context())
	next, err := GetNextRuntime(ctx, cronWf)
	require.NoError(t, err)
	assert.LessOrEqual(t, next.Unix(), time.Now().Add(1*time.Minute).Unix())
	assert.Greater(t, next.Unix(), time.Now().Unix())
}

var cronMultipleSchedules = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  creationTimestamp: "2020-05-19T16:47:25Z"
  generation: 98
  name: wonderful-tiger
  namespace: argo
  resourceVersion: "465179"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/cronworkflows/wonderful-tiger
  uid: c4ea2e84-ec58-4638-bf1d-5d543e7cc86a
spec:
  schedules:
  - '* * * * *'
  - '*/2 * * * *'
  workflowSpec:
    entrypoint: whalesay
    templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        command: [/argosay]
`

func TestNextRuntimeWithMultipleSchedules(t *testing.T) {
	var cronWf = v1alpha1.MustUnmarshalCronWorkflow(cronMultipleSchedules)
	ctx := logging.TestContext(t.Context())
	next, err := GetNextRuntime(ctx, cronWf)
	require.NoError(t, err)
	assert.LessOrEqual(t, next.Unix(), time.Now().Add(1*time.Minute).Unix())
	assert.Greater(t, next.Unix(), time.Now().Unix())
}

var cronHashedSchedules = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: wonderful-tiger
  namespace: argo
spec:
  schedules:
  - 'H H * * *'
  - 'H(0-29)/10 * * * *'
  workflowSpec:
    entrypoint: whalesay
    templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        command: [/argosay]
`

func TestPrintCronWorkflowResolvedSchedules(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("Hashed", func(t *testing.T) {
		cronWf := v1alpha1.MustUnmarshalCronWorkflow(cronHashedSchedules)
		out := getCronWorkflowGet(ctx, cronWf)
		assert.Contains(t, out, "Schedules:                     H H * * *,H(0-29)/10 * * * *")
		assert.Contains(t, out, "ResolvedSchedules:             52 1 * * *,2-29/10 * * * *")
	})

	t.Run("NotHashed", func(t *testing.T) {
		cronWf := v1alpha1.MustUnmarshalCronWorkflow(cronMultipleSchedules)
		out := getCronWorkflowGet(ctx, cronWf)
		assert.Contains(t, out, "Schedules:                     * * * * *,*/2 * * * *")
		assert.NotContains(t, out, "ResolvedSchedules:")
	})
}
