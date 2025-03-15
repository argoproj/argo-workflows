package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

var sharedMetrics *Metrics = nil
var sharedTE *telemetry.TestMetricsExporter = nil

// getSharedMetrics returns a singleton metrics with test exporter
// This is necessary because only the first call to workqueue.SetProvider
// takes effect within a single binary
// This can be fixed when we update to client-go 0.27 or later and we can
// create workqueues with https://godocs.io/k8s.io/client-go/util/workqueue#NewRateLimitingQueueWithConfig
func getSharedMetrics() (*Metrics, *telemetry.TestMetricsExporter, error) {
	if sharedMetrics == nil {
		config := telemetry.Config{
			Enabled: true,
			TTL:     1 * time.Second,
		}
		var err error
		sharedMetrics, sharedTE, err = createTestMetrics(&config, Callbacks{})
		if err != nil {
			return nil, nil, err
		}

		workqueue.SetProvider(sharedMetrics)
	}
	return sharedMetrics, sharedTE, nil
}

// CreateDefaultTestMetrics creates a boring testExporter enabled
// metrics, suitable for many tests
func CreateDefaultTestMetrics() (*Metrics, *telemetry.TestMetricsExporter, error) {
	config := telemetry.Config{
		Enabled: true,
	}
	return createTestMetrics(&config, Callbacks{})
}

func createTestMetrics(config *telemetry.Config, callbacks Callbacks) (*Metrics, *telemetry.TestMetricsExporter, error) {
	ctx /* with cancel*/ := context.Background()
	te := telemetry.NewTestMetricsExporter()

	m, err := New(ctx, telemetry.TestScopeName, telemetry.TestScopeName, config, callbacks, metric.WithReader(te))
	return m, te, err
}
