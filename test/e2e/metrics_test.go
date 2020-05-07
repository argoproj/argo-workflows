package e2e

import (
	"runtime"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/suite"
)

type MetricsSuite struct {
	suite.Suite
}

func (s *MetricsSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  "http://localhost:9090",
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(&httpLogger{}, true),
			},
			Client: httpClient,
		})
}

func (s *MetricsSuite) TestMetrics() {
	s.e(s.T()).GET("/metrics").
		Expect().
		Status(200)
}

func (s *MetricsSuite) TestTelemetry() {
	expect := s.e(s.T()).GET("/telemetry").
		Expect()
	if runtime.GOOS == "darwin" {
		// "process metrics not supported on this platform"
		expect.
			Status(500)
	} else {
		expect.
			Status(200)
	}
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}
