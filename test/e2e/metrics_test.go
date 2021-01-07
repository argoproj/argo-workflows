// +build e2e

package e2e

import (
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo/test/e2e/fixtures"
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
				httpexpect.NewDebugPrinter(&httpLogger{}, true),
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

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
