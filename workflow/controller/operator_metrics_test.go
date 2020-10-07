package controller

import (
	"fmt"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
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
        image: alpine:latest
        command: [sh, -c]
        args: ["RAND_INT=$((1 + RANDOM % 10)); echo $RAND_INT; echo $RAND_INT > /tmp/rand_int.txt"]
`

func TestBasicMetric(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(basicMetric)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	// Schedule first pod and mark completed
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate()
	makePodsPhase(woc, apiv1.PodSucceeded)

	// Process first metrics
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate()

	metricDesc := wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDesc))
	metric := controller.metrics.GetCustomMetric(metricDesc).(prometheus.Gauge)
	metricString, err := getMetricStringValue(metric)
	assert.NoError(t, err)
	assert.Contains(t, metricString, `label:<name:"name" value:"random-int" > gauge:<value:`)
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
	wf := unmarshalWF(counterMetric)
	cancel, controller := newController(wf)
	defer cancel()

	// Schedule first pod and mark completed
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	makePodsPhase(woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate()

	metricTotalDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricTotalDesc))
	metricErrorDesc := woc.wf.Spec.Templates[0].Metrics.Prometheus[1].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	metricTotalCounter := controller.metrics.GetCustomMetric(metricTotalDesc).(prometheus.Counter)
	metricTotalCounterString, err := getMetricStringValue(metricTotalCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricTotalCounterString, `label:<name:"name" value:"flakey" > counter:<value:1 >`)

	metricErrorCounter, ok := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
	if ok {
		metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
		assert.NoError(t, err)
		assert.Contains(t, metricErrorCounterString, `label:<name:"name" value:"flakey" > counter:<value:1 >`)
	}
}

func getMetricStringValue(metric prometheus.Metric) (string, error) {
	metricString := &dto.Metric{}
	err := metric.Write(metricString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", metricString), nil
}

var testMetricEmissionSameOperationCreationAndFailure = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-05-14T14:30:31Z"
  name: steps-s5rz4
spec:
  arguments: {}
  entrypoint: steps-1
  onExit: whalesay
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: steps-1
    outputs: {}
    steps:
    - - arguments: {}
        name: hello2a
        template: steps-2
  - arguments: {}
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
    - - arguments: {}
        name: hello1
        template: whalesay
        withParam: mary had a little lamb
  - arguments: {}
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
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(testMetricEmissionSameOperationCreationAndFailure)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()

	metricErrorDesc := wf.Spec.Templates[1].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `counter:<value:1 > `)
}

var testRetryStrategyMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-template-whalesay-9pk8f
spec:
  arguments: {}
  entrypoint: whalesay
  templates:
  - arguments: {}
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
  - arguments: {}
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
	wf := unmarshalWF(testRetryStrategyMetric)
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

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
	woc.operate()

	metricErrorDesc = wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	if assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc)) {
		metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
		metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
		assert.NoError(t, err)
		assert.Contains(t, metricErrorCounterString, `counter:<value:1 > `)

		metricErrorDesc = wf.Spec.Templates[1].Metrics.Prometheus[0].GetDesc()
		assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
		metricErrorCounter = controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
		metricErrorCounterString, err = getMetricStringValue(metricErrorCounter)
		assert.NoError(t, err)
		assert.Contains(t, metricErrorCounterString, `counter:<value:1 > `)
	}
}

var dagTmplMetrics = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-nl9bj
spec:
  arguments: {}
  entrypoint: steps
  templates:
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
        name: random-int-dag
        template: random-int
      - arguments: {}
        name: flakey-dag
        template: flakey

    name: steps
    outputs: {}
  - arguments: {}
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
  - arguments: {}
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
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(dagTmplMetrics)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	makePodsPhase(woc, apiv1.PodSucceeded)
	woc.operate()
	tmpl := woc.wf.GetTemplateByName("random-int")
	assert.NotNil(t, tmpl)
	metricDesc := tmpl.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDesc))
	metricHistogram := controller.metrics.GetCustomMetric(metricDesc).(prometheus.Histogram)
	metricHistogramString, err := getMetricStringValue(metricHistogram)
	assert.NoError(t, err)
	assert.Contains(t, metricHistogramString, `histogram:<sample_count:1 sample_sum:5`)

	tmpl = woc.wf.GetTemplateByName("flakey")
	assert.NotNil(t, tmpl)
	metricDesc = tmpl.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricDesc))
	metricCounter := controller.metrics.GetCustomMetric(metricDesc).(prometheus.Counter)
	metricCounterString, err := getMetricStringValue(metricCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricCounterString, `counter:<value:1 > `)
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
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(testRealtimeWorkflowMetricWithGlobalParameters)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()

	metricErrorDesc := woc.wf.Spec.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `label:<name:"workflowName" value:"test-foobar" > gauge:<value:`)
}
