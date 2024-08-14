package controller

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var space = regexp.MustCompile(`\s+`)

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
        image: alpine:latest
        command: [sh, -c]
        args: ["RAND_INT=$((1 + RANDOM % 10)); echo $RAND_INT; echo $RAND_INT > /tmp/rand_int.txt"]
`

func TestBasicMetric(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(basicMetric)
	ctx := context.Background()
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	// Schedule first pod and mark completed
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// Process first metrics
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	metricDesc := wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDesc))
	metric := controller.metrics.GetCustomMetric(metricDesc).(prometheus.Gauge)
	metricString, err := getMetricStringValue(metric)
	require.NoError(t, err)
	assert.Contains(t, metricString, `label:{name:"name" value:"random-int"} gauge:{value:`)
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
	cancel, controller := newController(wf)
	defer cancel()

	// Schedule first pod and mark completed
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	metricAddDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricAddDesc))
	metricSubDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[1].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricSubDesc))
	metricSetDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[2].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricSetDesc))
	metricDefaultDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[3].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDefaultDesc))

	metricAddGauge := controller.metrics.GetCustomMetric(metricAddDesc).(prometheus.Gauge)
	metricAddGaugeValue, err := getMetricStringValue(metricAddGauge)
	require.NoError(t, err)
	assert.Contains(t, metricAddGaugeValue, `label:{name:"name" value:"random-int"} gauge:{value:10}`)

	metricSubGauge := controller.metrics.GetCustomMetric(metricSubDesc).(prometheus.Gauge)
	metricSubGaugeValue, err := getMetricStringValue(metricSubGauge)
	require.NoError(t, err)
	assert.Contains(t, metricSubGaugeValue, `label:{name:"name" value:"random-int"} gauge:{value:-5}`)

	metricSetGauge := controller.metrics.GetCustomMetric(metricSetDesc).(prometheus.Gauge)
	metricSetGaugeValue, err := getMetricStringValue(metricSetGauge)
	require.NoError(t, err)
	assert.Contains(t, metricSetGaugeValue, `label:{name:"name" value:"random-int"} gauge:{value:50}`)

	metricDefaultGauge := controller.metrics.GetCustomMetric(metricDefaultDesc).(prometheus.Gauge)
	metricDefaultGaugeValue, err := getMetricStringValue(metricDefaultGauge)
	require.NoError(t, err)
	assert.Contains(t, metricDefaultGaugeValue, `label:{name:"name" value:"random-int"} gauge:{value:15}`)
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
	cancel, controller := newController(wf)
	defer cancel()

	// Schedule first pod and mark completed
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	metricTotalDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricTotalDesc))
	metricErrorDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[1].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	metricTotalCounter := controller.metrics.GetCustomMetric(metricTotalDesc).(prometheus.Counter)
	metricTotalCounterString, err := getMetricStringValue(metricTotalCounter)
	require.NoError(t, err)
	assert.Contains(t, metricTotalCounterString, `label:{name:"name" value:"flakey"} counter:{value:1`)

	metricErrorCounter, ok := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
	if ok {
		metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
		require.NoError(t, err)
		assert.Contains(t, metricErrorCounterString, `label:{name:"name" value:"flakey"} counter:{value:1`)
	}
}

func getMetricStringValue(metric prometheus.Metric) (string, error) {
	metricString := &dto.Metric{}
	err := metric.Write(metricString)
	if err != nil {
		return "", err
	}

	// Workaround for https://github.com/prometheus/client_model/issues/83
	normalizedString := space.ReplaceAllString(metricString.String(), " ")
	return normalizedString, nil
}

func getMetricGaugeValue(metric prometheus.Metric) (*float64, error) {
	metricString := &dto.Metric{}
	err := metric.Write(metricString)
	if err != nil {
		return nil, err
	}
	return metricString.Gauge.Value, nil
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
  -
    inputs: {}
    metadata: {}
    name: steps-1
    outputs: {}
    steps:
    - -
        name: hello2a
        template: steps-2
  -
    inputs: {}
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
    - -
        name: hello1
        template: whalesay
        withParam: mary had a little lamb
  -
    container:
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testMetricEmissionSameOperationCreationAndFailure)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	metricErrorDesc := wf.Spec.Templates[1].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	require.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `counter:{value:1 `)
}

var testRetryStrategyMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-template-whalesay-9pk8f
spec:

  entrypoint: whalesay
  templates:
  -
    inputs: {}
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
  -
    container:
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
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	ctx := context.Background()
	woc.operate(ctx)

	// Ensure no metrics have been emitted yet
	metricErrorDesc := wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.Nil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	metricErrorDesc = wf.Spec.Templates[1].Metrics.Prometheus[0].GetDesc()
	assert.Nil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	// Simulate pod succeeded
	podNode := woc.wf.Status.Nodes["workflow-template-whalesay-9pk8f-1966833540"]
	podNode.Phase = v1alpha1.NodeSucceeded
	woc.wf.Status.Nodes["workflow-template-whalesay-9pk8f-1966833540"] = podNode
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	metricErrorDesc = wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	if assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc)) {
		metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
		metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
		require.NoError(t, err)
		assert.Contains(t, metricErrorCounterString, `counter:{value:1 `)

		metricErrorDesc = wf.Spec.Templates[1].Metrics.Prometheus[0].GetDesc()
		assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
		metricErrorCounter = controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
		metricErrorCounterString, err = getMetricStringValue(metricErrorCounter)
		require.NoError(t, err)
		assert.Contains(t, metricErrorCounterString, `counter:{value:1 `)
	}
}

var dagTmplMetrics = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-nl9bj
spec:

  entrypoint: steps
  templates:
  -
    dag:
      tasks:
      -
        name: random-int-dag
        template: random-int
      -
        name: flakey-dag
        template: flakey

    name: steps
    outputs: {}
  -
    container:
      args:
      - RAND_INT=$((1 + RANDOM % 10)); echo $RAND_INT; echo $RAND_INT > /tmp/rand_int.txt
      command:
      - sh
      - -c
      image: alpine:latest
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
  -
    container:
      args:
      - import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.6
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(dagTmplMetrics)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	tmpl := woc.wf.GetTemplateByName("random-int")
	assert.NotNil(t, tmpl)
	metricDesc := tmpl.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDesc))
	metricHistogram := controller.metrics.GetCustomMetric(metricDesc).(prometheus.Histogram)
	metricHistogramString, err := getMetricStringValue(metricHistogram)
	require.NoError(t, err)
	assert.Contains(t, metricHistogramString, `histogram:{sample_count:1 sample_sum:5`)

	tmpl = woc.wf.GetTemplateByName("flakey")
	assert.NotNil(t, tmpl)
	metricDesc = tmpl.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDesc))
	metricCounter := controller.metrics.GetCustomMetric(metricDesc).(prometheus.Counter)
	metricCounterString, err := getMetricStringValue(metricCounter)
	require.NoError(t, err)
	assert.Contains(t, metricCounterString, `counter:{value:1 `)
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testRealtimeWorkflowMetric)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	metricErrorDesc := woc.wf.Spec.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	value, err := getMetricGaugeValue(controller.metrics.GetCustomMetric(metricErrorDesc))
	require.NoError(t, err)
	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	require.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `label:{name:"label" value:"foobar"} label:{name:"workflowName" value:"test-foobar"} gauge:{value:`)

	value1, err := getMetricGaugeValue(controller.metrics.GetCustomMetric(metricErrorDesc))
	require.NoError(t, err)
	assert.Greater(t, *value1, *value)
	woc.markWorkflowSuccess(ctx)
	controller.metrics.GetCustomMetric(metricErrorDesc)
	value2, err := getMetricGaugeValue(controller.metrics.GetCustomMetric(metricErrorDesc))
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	controller.metrics.GetCustomMetric(metricErrorDesc)
	value3, err := getMetricGaugeValue(controller.metrics.GetCustomMetric(metricErrorDesc))
	require.NoError(t, err)
	// Duration should be same after workflow complete
	assert.InEpsilon(t, *value2, *value3, 0.001)
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testRealtimeWorkflowMetricWithGlobalParameters)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	metricErrorDesc := woc.wf.Spec.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	require.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `label:{name:"label" value:"foobar"} label:{name:"workflowName" value:"test-foobar"} gauge:{value`)
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testProcessedRetryNode)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	metric := controller.metrics.GetCustomMetric("result_counter{work_unit=metrics-eg::A,workflow_result=Succeeded,}")
	assert.NotNil(t, metric)
	metricErrorCounterString, err := getMetricStringValue(metric)
	require.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `value:1`)
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(suspendWfWithMetrics)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	metricDesc := wf.Spec.Metrics.Prometheus[0].GetDesc()
	metric := controller.metrics.GetCustomMetric(metricDesc)
	assert.NotNil(t, metric)
	metricString, err := getMetricStringValue(metric)
	fmt.Println(metricString)
	require.NoError(t, err)
	assert.Contains(t, metricString, `model_a`)
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
      image: alpine:3.7
      command: [echo, "hello"]
`

func TestRuntimeMetrics(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(runtimeWfMetrics)
	ctx := context.Background()
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx) // create step node

	makePodsPhase(ctx, woc, apiv1.PodSucceeded) // pod is successful - manually workflow is succeeded
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx) // node status of previous context

	metricDesc := woc.wf.Spec.Metrics.Prometheus[0].GetDesc()
	metric := controller.metrics.GetCustomMetric(metricDesc)
	assert.NotNil(t, metric)
	metricString, err := getMetricStringValue(metric)
	require.NoError(t, err)
	assert.Contains(t, metricString, `Succeeded`)
}
