package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

var basicMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: random-int
  templates:
    - name: random-int
      metrics:
        prometheus:
          - name: duration_gauge
            labels:
              - key: name
                value: random-int
            help: "Duration gauge by name"
            gauge:
              value: "{{duration}}"
      outputs:
        parameters:
          - name: rand-int-value
            globalName: rand-int-value
            valueFrom:
              path: /tmp/rand_int.txt
      container:
        image: alpine:3.23
        command: [sh, -c]
        args: ["RAND_INT=$((1 + RANDOM % 10)); echo $RAND_INT; echo $RAND_INT > /tmp/rand_int.txt"]
`

func TestBasicMetric(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(basicMetric)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Schedule first pod and mark completed
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// Process first metrics
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	metricName := wf.Spec.Templates[0].Metrics.Prometheus[0].Name
	assert.True(t, controller.metrics.CustomMetricExists(metricName))
	attribs := attribute.NewSet(attribute.String("name", "random-int"))
	_, err = testExporter.GetFloat64GaugeValue(ctx, metricName, &attribs)
	require.NoError(t, err)
}

var gaugeMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: gauge-metric
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      metrics:
        prometheus:
          - name: custom_gauge_add
            labels:
              - key: name
                value: random-int
            help: "A custom gauge"
            gauge:
              operation: Add
              value: "10"
          - name: custom_gauge_sub
            labels:
              - key: name
                value: random-int
            help: "A custom gauge"
            gauge:
              operation: Sub
              value: "5"
          - name: custom_gauge_set
            labels:
              - key: name
                value: random-int
            help: "A custom gauge"
            gauge:
              operation: Set
              value: "50"
          - name: custom_gauge_default
            labels:
              - key: name
                value: random-int
            help: "A custom gauge"
            gauge:
              value: "15"
      container:
        image: docker/whalesay:latest
        command: [cowsay]

`

func TestGaugeMetric(t *testing.T) {
	wf := v1alpha1.MustUnmarshalWorkflow(gaugeMetric)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Schedule first pod and mark completed
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("name", "random-int"))

	valAdd, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(10.0), valAdd, 0.001)

	valSub, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[1].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(-5.0), valSub, 0.001)

	valSet, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[2].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(50.0), valSet, 0.001)

	valDefault, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[3].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(15.0), valDefault, 0.001)
}

var counterMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: counter-metric
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      metrics:
        prometheus:
          - name: execution_counter
            help: "How many times a step has executed"
            labels:
              - key: name
                value: flakey
            counter:
              value: "1"
          - name: failure_counter
            help: "How many times a step has failed"
            labels:
              - key: name
                value: flakey
            when: "{{status}} == Failed"
            counter:
              value: "1"
      container:
        image: docker/whalesay:latest
        command: [cowsay]

`

func TestCounterMetric(t *testing.T) {
	wf := v1alpha1.MustUnmarshalWorkflow(counterMetric)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Schedule first pod and mark completed
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("name", "flakey"))

	valTotal, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valTotal, 0.001)

	valError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[1].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valError, 0.001)
}

var testMetricEmissionSameOperationCreationAndFailure = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-05-14T14:30:31Z"
  name: steps-s5rz4
spec:
  entrypoint: steps-1
  onExit: whalesay
  templates:
  - inputs: {}
    metadata: {}
    name: steps-1
    outputs: {}
    steps:
    - -
        name: hello2a
        template: steps-2
  - inputs: {}
    metadata: {}
    metrics:
      prometheus:
      - counter:
          value: "1"
        gauge: null
        help: Failure
        histogram: null
        labels: null
        name: failure
        when: '{{status}} == Failed'
    name: steps-2
    outputs: {}
    steps:
    - - name: hello1
        template: whalesay
        withParam: mary had a little lamb
  - container:
      args:
      - hello
      command:
      - cowsay
      image: docker/whalesay
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  phase: Running
  startedAt: "2020-05-14T14:30:31Z"
`

func TestMetricEmissionSameOperationCreationAndFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testMetricEmissionSameOperationCreationAndFailure)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet()

	valError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[1].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valError, 0.001)
}

var testRetryStrategyMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-template-whalesay-9pk8f
spec:
  entrypoint: whalesay
  templates:
  - inputs: {}
    metadata: {}
    metrics:
      prometheus:
      - counter:
          value: "1"
        help: number of times the outer workflow was invoked
        name: workflow_counter
    name: whalesay
    outputs: {}
    steps:
    - - arguments:
          parameters:
          - name: message
            value: hello world
        name: call-whalesay-template
        template: whalesay-template
  - container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    metrics:
      prometheus:
      - counter:
          value: "1"
        help: number of times the template was executed
        name: template_counter
    name: whalesay-template
    outputs: {}
    retryStrategy:
      limit: "2"
`

func TestRetryStrategyMetric(t *testing.T) {
	wf := v1alpha1.MustUnmarshalWorkflow(testRetryStrategyMetric)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Ensure no metrics have been emitted yet
	metricErrorDesc := wf.Spec.Templates[0].Metrics.Prometheus[0].GetKey()
	assert.Nil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	metricErrorDesc = wf.Spec.Templates[1].Metrics.Prometheus[0].GetKey()
	assert.Nil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	// Simulate pod succeeded
	podNode := woc.wf.Status.Nodes["workflow-template-whalesay-9pk8f-1966833540"]
	podNode.Phase = v1alpha1.NodeSucceeded
	woc.wf.Status.Nodes["workflow-template-whalesay-9pk8f-1966833540"] = podNode
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet()

	valWfError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1.0), valWfError, 0.001)

	valTplError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[1].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1.0), valTplError, 0.001)
}

var dagTmplMetrics = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-nl9bj
spec:
  entrypoint: steps
  templates:
  - dag:
      tasks:
      - name: random-int-dag
        template: random-int
      - name: flakey-dag
        template: flakey
    name: steps
    outputs: {}
  - container:
      args:
      - RAND_INT=$((1 + RANDOM % 10)); echo $RAND_INT; echo $RAND_INT > /tmp/rand_int.txt
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    metrics:
      prometheus:
      - help: Value of the int emitted by random-int at step level
        histogram:
          buckets:
          - 2.01
          - 4.01
          - 6.01
          - 8.01
          - 10.01
          value: 5
        name: random_int_step_histogram_dag
      - gauge:
          realtime: true
          value: '{{duration}}'
        help: Duration gauge by name
        labels:
        - key: name
          value: random-int
        name: duration_gauge_dag
    name: random-int
    outputs:
      parameters:
      - globalName: rand-int-value
        name: rand-int-value
        valueFrom:
          path: /tmp/rand_int.txt
  - container:
      args:
      - import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    metrics:
      prometheus:
      - counter:
          value: "1"
        help: Count of step execution by result status
        labels:
        - key: name
          value: flakey
        - key: status
          value: Failed
        name: result_counter_dag
    name: flakey
    outputs: {}
`

func TestDAGTmplMetrics(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(dagTmplMetrics)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	attribs := attribute.NewSet()
	tmpl := woc.wf.GetTemplateByName("random-int")
	assert.NotNil(t, tmpl)

	val, err := testExporter.GetFloat64HistogramData(ctx, tmpl.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(5.0), val.Sum, 0.001)
	assert.Equal(t, uint64(1), val.Count)

	attribs = attribute.NewSet(attribute.String("name", "flakey"), attribute.String("status", "Failed"))
	tmpl = woc.wf.GetTemplateByName("flakey")
	assert.NotNil(t, tmpl)
	valErrCount, err := testExporter.GetFloat64CounterValue(ctx, tmpl.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valErrCount, 0.001)
}

var testRealtimeWorkflowMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-foobar
  labels:
    testLabel: foobar
spec:
  entrypoint: whalesay
  metrics:
    prometheus:
      - name: intuit_data_persistplat_dppselfservice_workflow_test_duration
        help: Duration of workflow
        labels:
          - key: workflowName
            value: "{{workflow.name}}"
          - key: label
            value: "{{workflow.labels.testLabel}}"
        gauge:
          realtime: true
          value: "{{ workflow.duration }}"
  templates:
    - name: whalesay
      container:
        image: docker/whalesay
        command: [ cowsay ]
        args: [ "hello world" ]
`

func TestRealtimeWorkflowMetric(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testRealtimeWorkflowMetric)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("label", "foobar"), attribute.String("workflowName", "test-foobar"))
	value, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	value1, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	t.Logf("%v new %v old", value1, value)
	assert.Greater(t, value1, value)

	ctx = woc.markWorkflowSuccess(ctx)
	value2, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	value3, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	// Duration should be same after workflow complete
	assert.InEpsilon(t, value2, value3, 0.001)
}

var testRealtimeWorkflowMetricWithGlobalParameters = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-foobar
  labels:
    testLabel: foobar
spec:
  arguments:
    parameters:
      - name: testParam
        value: foo
  entrypoint: whalesay
  metrics:
    prometheus:
      - name: intuit_data_persistplat_dppselfservice_workflow_test_duration
        help: Duration of workflow
        labels:
          - key: workflowName
            value: "{{workflow.name}}"
          - key: label
            value: "{{workflow.labels.testLabel}}"
        gauge:
          realtime: true
          value: "{{workflow.duration}}"
  templates:
    - name: whalesay
      container:
        image: docker/whalesay
        command: [ cowsay ]
        args: [ "hello world" ]
`

func TestRealtimeWorkflowMetricWithGlobalParameters(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testRealtimeWorkflowMetricWithGlobalParameters)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("label", "foobar"), attribute.String("workflowName", "test-foobar"))
	_, err = testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
}

var testProcessedRetryNode = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: metrics-eg-lq4nj
spec:
  entrypoint: my-dag
  templates:
  - dag:
      tasks:
      - name: A
        template: A
    name: my-dag
  - container:
      args:
      - hello from A
      command:
      - cowsay
      image: docker/whalesay
    metrics:
      prometheus:
      - counter:
          value: "1"
        help: Number of argo workflows
        labels:
        - key: work_unit
          value: metrics-eg::A
        - key: workflow_result
          value: '{{status}}'
        name: result_counter
    name: A
    retryStrategy:
      backoff:
        duration: 2s
        factor: 1
        maxDuration: 6m
      limit: 2
      retryPolicy: Always
status:
  nodes:
    metrics-eg-lq4nj:
      children:
      - metrics-eg-lq4nj-4266717436
      displayName: metrics-eg-lq4nj
      finishedAt: "2021-01-13T16:14:03Z"
      id: metrics-eg-lq4nj
      name: metrics-eg-lq4nj
      outboundNodes:
      - metrics-eg-lq4nj-2568729143
      phase: Running
      startedAt: "2021-01-13T16:13:53Z"
      templateName: my-dag
      templateScope: local/metrics-eg-lq4nj
      type: DAG
    metrics-eg-lq4nj-2568729143:
      boundaryID: metrics-eg-lq4nj
      displayName: A(0)
      finishedAt: "2021-01-13T16:13:57Z"
      id: metrics-eg-lq4nj-2568729143
      name: metrics-eg-lq4nj.A(0)
      phase: Succeeded
      startedAt: "2021-01-13T16:13:53Z"
      templateName: A
      templateScope: local/metrics-eg-lq4nj
      type: Pod
      nodeFlag:
        retried: true
    metrics-eg-lq4nj-4266717436:
      boundaryID: metrics-eg-lq4nj
      children:
      - metrics-eg-lq4nj-2568729143
      displayName: A
      finishedAt: "2021-01-13T16:14:03Z"
      id: metrics-eg-lq4nj-4266717436
      name: metrics-eg-lq4nj.A
      phase: Running
      startedAt: "2021-01-13T16:13:53Z"
      templateName: A
      templateScope: local/metrics-eg-lq4nj
      type: Retry
  phase: Running
  startedAt: "2021-01-13T16:13:53Z"
`

func TestProcessedRetryNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testProcessedRetryNode)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("work_unit", "metrics-eg::A"), attribute.String("workflow_result", "Succeeded"))
	value, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[1].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), value, 0.001)
}

var suspendWfWithMetrics = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-template-qndm5
spec:
  entrypoint: suspend
  metrics:
    prometheus:
    - gauge:
        realtime: true
        value: '{{workflow.duration}}'
      help: Duration gauge by name
      labels:
      - key: name
        value: model_a
      name: exec_duration_gauge
  templates:
  - name: suspend
    steps:
    - - name: build
        template: whalesay
    - - name: approve
        template: approve
    - - name: delay
        template: delay
    - - name: release
        template: whalesay
  - name: approve
    suspend: {}
  - name: delay
    suspend:
      duration: "20"
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay
      name: ""
    name: whalesay
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  conditions:
  - status: "False"
    type: PodRunning
  finishedAt: null
  nodes:
    suspend-template-qndm5:
      children:
      - suspend-template-qndm5-343839516
      displayName: suspend-template-qndm5
      finishedAt: null
      id: suspend-template-qndm5
      name: suspend-template-qndm5
      phase: Running
      progress: 1/1
      startedAt: "2021-09-28T12:23:10Z"
      templateName: suspend
      templateScope: local/suspend-template-qndm5
      type: Steps
    suspend-template-qndm5-343839516:
      boundaryID: suspend-template-qndm5
      children:
      - suspend-template-qndm5-2823755246
      displayName: '[0]'
      finishedAt: "2021-09-28T12:23:20Z"
      id: suspend-template-qndm5-343839516
      name: suspend-template-qndm5[0]
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 6
        memory: 3
      startedAt: "2021-09-28T12:23:10Z"
      templateScope: local/suspend-template-qndm5
      type: StepGroup
    suspend-template-qndm5-2823755246:
      boundaryID: suspend-template-qndm5
      children:
      - suspend-template-qndm5-3632002577
      displayName: build
      finishedAt: "2021-09-28T12:23:16Z"
      hostNodeName: kind-control-plane
      id: suspend-template-qndm5-2823755246
      name: suspend-template-qndm5[0].build
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 6
        memory: 3
      startedAt: "2021-09-28T12:23:10Z"
      templateName: whalesay
      templateScope: local/suspend-template-qndm5
      type: Pod
    suspend-template-qndm5-3456849218:
      boundaryID: suspend-template-qndm5
      displayName: approve
      finishedAt: null
      id: suspend-template-qndm5-3456849218
      name: suspend-template-qndm5[1].approve
      phase: Running
      startedAt: "2021-09-28T12:23:20Z"
      templateName: approve
      templateScope: local/suspend-template-qndm5
      type: Suspend
    suspend-template-qndm5-3632002577:
      boundaryID: suspend-template-qndm5
      children:
      - suspend-template-qndm5-3456849218
      displayName: '[1]'
      finishedAt: null
      id: suspend-template-qndm5-3632002577
      name: suspend-template-qndm5[1]
      phase: Running
      startedAt: "2021-09-28T12:23:20Z"
      templateScope: local/suspend-template-qndm5
      type: StepGroup
  phase: Running
  progress: 1/1
  resourcesDuration:
    cpu: 6
    memory: 3
  startedAt: "2021-09-28T12:23:10Z"
`

func TestControllerRestartWithRunningWorkflow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(suspendWfWithMetrics)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	attribs := attribute.NewSet(attribute.String("name", "model_a"))
	_, err = testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
}

var runtimeWfMetrics = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-task-
spec:
  entrypoint: dag-task
  metrics: # Custom metric workflow level
    prometheus:
      - name: playground_workflow_new
        help: "Count of workflow execution by result status  - workflow level"
        labels:
          - key: "playground_id_workflow_counter"
            value: "test"
          - key: status
            value: "{{workflow.status}}"
        counter:
          value: "1"
  templates:
  - name: dag-task
    dag:
      tasks:
      - name: TEST-ONE
        template: echo

  - name: echo
    container:
      image: alpine:3.23
      command: [echo, "hello"]
`

func TestRuntimeMetrics(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(runtimeWfMetrics)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // create step node

	makePodsPhase(ctx, woc, apiv1.PodSucceeded) // pod is successful - manually workflow is succeeded
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // node status of previous context

	attribs := attribute.NewSet(attribute.String("playground_id_workflow_counter", "test"), attribute.String("status", "Succeeded"))
	value, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), value, 0.001)
}
