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
			Contains(`log_messages{level="info"}`).
			Contains(`log_messages{level="warning"}`).
			Contains(`log_messages{level="error"}`)
	})
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
