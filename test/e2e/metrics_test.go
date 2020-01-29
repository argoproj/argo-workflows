package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/gavv/httpexpect.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

const metricsUrl = "http://localhost:9090"

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type MetricsSuite struct {
	fixtures.E2ESuite
}

func (s *MetricsSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  metricsUrl,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(s.Diagnostics, true),
			},
		})
}

func (s *MetricsSuite) TestEndpoint() {
	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/metrics").
			Expect().
			Status(200)
	})
}

func (s *MetricsSuite) TestBasic() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		Expect(func(t *testing.T, _ *metav1.ObjectMeta, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
			assert.NotEmpty(t, wf.Nodes)
		})

	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/metrics").
			Expect().
			Status(200).
			Body().
			Contains(`argo_workflow_status_phase{entrypoint="run-workflow",name="basic",namespace="argo",phase="Succeeded"} 1`).
			Contains(`argo_workflow_step_status_phase{name="basic",namespace="argo",phase="Succeeded",step_name="basic"} 1`)
	})
}

func (s *MetricsSuite) TestCustomMetrics() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: custom-metric
spec:
  entrypoint: custom-metric-example
  templates:
    - name: custom-metric-example
      steps:
        - - name: generate
            template: gen-random-int
    - name: gen-random-int
      outputs:
        parameters:
          - name: script_result
            valueFrom:
              path: "/tmp/metric.txt"
            emitMetric:
              metricSuffix: "number_generated"
              metricTags:
                - name: "generator_id"
                  value: "A"
      script:
        image: debian:9.4
        command: [bash]
        source: |
          echo "2746" > /tmp/metric.txt`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(40 * time.Second).
		Then().
		Expect(func(t *testing.T, _ *metav1.ObjectMeta, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
			assert.NotEmpty(t, wf.Nodes)
		})

	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/metrics").
			Expect().
			Status(200).
			Body().
			Contains(`argo_workflow_status_phase{entrypoint="custom-metric-example",name="custom-metric",namespace="argo",phase="Succeeded"} 1`).
			Contains(`argo_workflow_number_generated{generator_id="A",name="custom-metric",namespace="argo",step_name="custom-metric[0].generate"} 2746`)
	})
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
