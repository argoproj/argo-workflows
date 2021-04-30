package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	wf := v1alpha1.MustUnmarshalWorkflow(basicMetric)
	ctx := context.Background()
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

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
	assert.NoError(t, err)
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
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow(testRealtimeWorkflowMetricWithGlobalParameters)
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	metricErrorDesc := woc.wf.Spec.Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `label:<name:"workflowName" value:"test-foobar" > gauge:<value:`)
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
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	metric := controller.metrics.GetCustomMetric("result_counter{work_unit=metrics-eg::A,workflow_result=Succeeded,}")
	assert.NotNil(t, metric)
	metricErrorCounterString, err := getMetricStringValue(metric)
	assert.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `value:1`)
}
