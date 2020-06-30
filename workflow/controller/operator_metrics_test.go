package controller

import (
	"fmt"
	"testing"

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
	woc.operate()
	makePodsPhaseAll(t, apiv1.PodSucceeded, controller.kubeclientset, wf.ObjectMeta.Namespace)

	// Process first metrics
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
  generateName: hello-world-
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
            when: "{{status}} == Error"
            counter:
              value: "1"
      container:
        image: docker/whalesay:latest
        command: [cowsay]
`

func TestCounterMetric(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(counterMetric)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	// Schedule first pod and mark completed
	woc.operate()
	makePodsPhaseAll(t, apiv1.PodFailed, controller.kubeclientset, wf.ObjectMeta.Namespace)

	// Process first metrics
	woc.operate()

	metricTotalDesc := wf.Spec.Templates[0].Metrics.Prometheus[0].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricTotalDesc))
	metricErrorDesc := wf.Spec.Templates[0].Metrics.Prometheus[1].GetDesc()
	assert.NotNil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	metricTotalCounter := controller.metrics.GetCustomMetric(metricTotalDesc).(prometheus.Counter)
	metricTotalCounterString, err := getMetricStringValue(metricTotalCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricTotalCounterString, `label:<name:"name" value:"flakey" > counter:<value:1 >`)

	metricErrorCounter := controller.metrics.GetCustomMetric(metricErrorDesc).(prometheus.Counter)
	metricErrorCounterString, err := getMetricStringValue(metricErrorCounter)
	assert.NoError(t, err)
	assert.Contains(t, metricErrorCounterString, `label:<name:"name" value:"flakey" > counter:<value:1 >`)
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
