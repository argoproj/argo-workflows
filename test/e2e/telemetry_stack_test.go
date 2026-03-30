//go:build telemetry

// Tests in this file require the full telemetry stack (otel-collector, tempo,
// prometheus, grafana) deployed via PROFILE=telemetry-stack, which is separate
// from the tracing build tag that only needs the otel-collector debug exporter.

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type TelemetryStackSuite struct {
	fixtures.E2ESuite
}

func TestTelemetryStack(t *testing.T) {
	suite.Run(t, new(TelemetryStackSuite))
}

func (s *TelemetryStackSuite) TestTracesAndMetrics() {
	s.Given().
		Workflow(`@../../examples/dag-diamond.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			// Get trace ID from workflow annotation
			traceID := metadata.Annotations[common.AnnotationKeyTraceID]
			require.NotEmpty(t, traceID, "workflow should have trace ID annotation")
			t.Logf("Trace ID: %s", traceID)

			ctx := t.Context()

			// Poll Tempo for the trace via k8s service proxy
			t.Run("TracesReachTempo", func(t *testing.T) {
				tempoURL := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/traces/%s",
					fixtures.Namespace, fixtures.TempoServiceName, fixtures.TempoServicePort, traceID)

				var traceFound bool
				require.Eventually(t, func() bool {
					body, statusCode, err := fixtures.ProxyGet(ctx, s.KubeClient, tempoURL)
					if err != nil {
						t.Logf("Tempo query error: %v", err)
						return false
					}
					if statusCode == http.StatusNotFound {
						t.Log("Trace not yet available in Tempo")
						return false
					}
					if statusCode != http.StatusOK {
						t.Logf("Tempo returned status %d: %s", statusCode, string(body))
						return false
					}
					var result fixtures.TempoTraceResponse
					if err := json.Unmarshal(body, &result); err != nil {
						t.Logf("Failed to parse Tempo response: %v", err)
						return false
					}
					if len(result.Batches) > 0 {
						traceFound = true
						t.Logf("Found trace with %d batches in Tempo", len(result.Batches))
						return true
					}
					t.Log("Tempo returned empty trace")
					return false
				}, 60*time.Second, 3*time.Second, "trace should be available in Tempo")
				assert.True(t, traceFound)
			})

			// Poll Prometheus for workflow controller metrics via k8s service proxy.
			// Metrics flow: controller -> OTLP HTTP -> collector -> prometheusremotewrite -> Prometheus.
			// The OTEL periodic reader flushes every 60s, so allow up to 120s for metrics to appear.
			t.Run("MetricsReachPrometheus", func(t *testing.T) {
				prometheusPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/v1/query",
					fixtures.Namespace, fixtures.PrometheusServiceName, fixtures.PrometheusServicePort)

				require.Eventually(t, func() bool {
					// target_info is emitted by the OTLP SDK for every resource and is the
					// first metric to appear after a successful remote-write handshake.
					body, statusCode, err := fixtures.ProxyGetWithParams(ctx, s.KubeClient, prometheusPath, map[string]string{
						"query": "target_info",
					})
					if err != nil {
						t.Logf("Prometheus query error: %v", err)
						return false
					}
					if statusCode != http.StatusOK {
						t.Logf("Prometheus returned status %d: %s", statusCode, string(body))
						return false
					}
					var result fixtures.PrometheusQueryResponse
					if err := json.Unmarshal(body, &result); err != nil {
						t.Logf("Failed to parse Prometheus response: %v", err)
						return false
					}
					if result.Status == "success" && len(result.Data.Result) > 0 {
						t.Logf("Found %d target_info series in Prometheus", len(result.Data.Result))
						return true
					}
					t.Log("No target_info metrics yet")
					return false
				}, 120*time.Second, 5*time.Second, "target_info metric should be available in Prometheus via OTLP remote write")
			})
		})
}
