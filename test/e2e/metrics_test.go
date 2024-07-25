//go:build api

package e2e

import (
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

const baseUrlMetrics = "http://localhost:9090/metrics"

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type MetricsSuite struct {
	fixtures.E2ESuite
}

func (s *MetricsSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
}

func (s *MetricsSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseUrlMetrics,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(s.T(), true),
			},
			Client: httpClient,
		})
}

func (s *MetricsSuite) TestMetricsEndpoint() {
	s.Run("Metrics", func() {
		s.e(s.T()).GET("").
			Expect().
			Status(200).
			Body().
			Contains(`HELP argo_workflows_count`).
			Contains(`HELP argo_workflows_k8s_request_total`).
			Contains(`argo_workflows_k8s_request_total{kind="leases",status_code="200",verb="Get"}`).
			Contains(`argo_workflows_k8s_request_total{kind="workflowtemplates",status_code="200",verb="List"}`).
			Contains(`argo_workflows_k8s_request_total{kind="workflowtemplates",status_code="200",verb="Watch"}`).
			Contains(`HELP argo_workflows_pods_count`).
			Contains(`HELP argo_workflows_workers_busy`).
			Contains(`HELP argo_workflows_workflow_condition`).
			Contains(`HELP argo_workflows_workflows_processed_count`).
			Contains(`log_messages{level="info"}`).
			Contains(`log_messages{level="warning"}`).
			Contains(`log_messages{level="error"}`)
	})
}

func (s *MetricsSuite) TestRetryMetrics() {
	s.Given().
		Workflow(`@testdata/workflow-retry-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`runs_exit_status_counter{exit_code="1",status="Failed"} 3`)
		})
}

func (s *MetricsSuite) TestDAGMetrics() {
	s.Given().
		Workflow(`@testdata/workflow-dag-metrics.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`argo_workflows_result_counter{status="Succeeded"} 5`)
		})
}

func (s *MetricsSuite) TestFailedMetric() {
	s.Given().
		Workflow(`@testdata/template-status-failed-conditional-metric.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			s.e(s.T()).GET("").
				Expect().
				Status(200).
				Body().
				Contains(`argo_workflows_task_failure 1`)
		})
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
