//go:build telemetry

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v4/util/kubeconfig"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"sigs.k8s.io/yaml"
)

// TestTelemetryStack verifies that traces reach Tempo and metrics reach Prometheus
// when using the full observability stack (otel-collector, tempo, prometheus, grafana).
// Requires PROFILE=telemetry-stack.
func TestTelemetryStack(t *testing.T) {
	restConfig, err := kubeconfig.DefaultRestConfig()
	require.NoError(t, err, "failed to get rest config")

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	require.NoError(t, err, "failed to create kube client")

	wfClient := versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().Workflows(fixtures.Namespace)

	// Load the dag-diamond workflow
	data, err := os.ReadFile("../../examples/dag-diamond.yaml")
	require.NoError(t, err, "failed to read dag-diamond.yaml")

	var wf wfv1.Workflow
	require.NoError(t, yaml.Unmarshal(data, &wf), "failed to unmarshal workflow")

	// Submit workflow
	ctx := t.Context()
	created, err := wfClient.Create(ctx, &wf, metav1.CreateOptions{})
	require.NoError(t, err, "failed to submit workflow")
	t.Logf("Submitted workflow %s", created.Name)

	// Wait for completion
	watcher, err := wfClient.Watch(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", created.Name).String(),
	})
	require.NoError(t, err, "failed to watch workflow")
	defer watcher.Stop()

	var completed *wfv1.Workflow
	timeout := time.After(120 * time.Second)
	for completed == nil {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for workflow to complete")
		case event := <-watcher.ResultChan():
			if event.Type == watch.Modified || event.Type == watch.Added {
				wf, ok := event.Object.(*wfv1.Workflow)
				if ok && wf.Status.Phase.Completed() {
					completed = wf
				}
			}
		}
	}
	require.Equal(t, wfv1.WorkflowSucceeded, completed.Status.Phase, "workflow should succeed")
	t.Logf("Workflow %s completed with phase %s", completed.Name, completed.Status.Phase)

	// Get trace ID from workflow annotation
	traceID := completed.Annotations[common.AnnotationKeyTraceID]
	require.NotEmpty(t, traceID, "workflow should have trace ID annotation")
	t.Logf("Trace ID: %s", traceID)

	// Poll Tempo for the trace via k8s service proxy
	t.Run("TracesReachTempo", func(t *testing.T) {
		tempoURL := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/traces/%s",
			fixtures.Namespace, "tempo", 3200, traceID)

		var traceFound bool
		require.Eventually(t, func() bool {
			body, statusCode, err := proxyGet(ctx, kubeClient, tempoURL)
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
			var result tempoTraceResponse
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
			fixtures.Namespace, "prometheus", 9090)

		require.Eventually(t, func() bool {
			// target_info is emitted by the OTLP SDK for every resource and is the
			// first metric to appear after a successful remote-write handshake.
			body, statusCode, err := proxyGetWithParams(ctx, kubeClient, prometheusPath, map[string]string{
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
			var result prometheusQueryResponse
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
}

// proxyGet performs a GET request through the k8s API server proxy.
func proxyGet(ctx context.Context, kubeClient kubernetes.Interface, path string) ([]byte, int, error) {
	return proxyGetWithParams(ctx, kubeClient, path, nil)
}

// proxyGetWithParams performs a GET request through the k8s API server proxy
// with query parameters. Params must be passed separately because AbsPath
// URL-encodes '?' to '%3F'.
func proxyGetWithParams(ctx context.Context, kubeClient kubernetes.Interface, path string, params map[string]string) ([]byte, int, error) {
	req := kubeClient.CoreV1().RESTClient().Get().AbsPath(path)
	for k, v := range params {
		req = req.Param(k, v)
	}
	result := req.Do(ctx)
	rawBody, err := result.Raw()
	if err != nil {
		var statusCode int
		result.StatusCode(&statusCode)
		if statusCode != 0 {
			return rawBody, statusCode, nil
		}
		return nil, 0, err
	}
	var statusCode int
	result.StatusCode(&statusCode)
	return rawBody, statusCode, nil
}

type tempoTraceResponse struct {
	Batches []json.RawMessage `json:"batches"`
}

type prometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []json.RawMessage `json:"result"`
	} `json:"data"`
}
