//go:build telemetry

package fixtures

import (
	"context"
	"encoding/json"

	"k8s.io/client-go/kubernetes"
)

const (
	TempoServiceName      = "tempo"
	TempoServicePort      = 3200
	PrometheusServiceName = "prometheus"
	PrometheusServicePort = 9090
)

// ProxyGet performs a GET request through the k8s API server proxy.
func ProxyGet(ctx context.Context, kubeClient kubernetes.Interface, path string) ([]byte, int, error) {
	return ProxyGetWithParams(ctx, kubeClient, path, nil)
}

// ProxyGetWithParams performs a GET request through the k8s API server proxy
// with query parameters. Params must be passed separately because AbsPath
// URL-encodes '?' to '%3F'.
func ProxyGetWithParams(ctx context.Context, kubeClient kubernetes.Interface, path string, params map[string]string) ([]byte, int, error) {
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

// TempoTraceResponse represents the JSON response from Tempo's trace API.
type TempoTraceResponse struct {
	Batches []json.RawMessage `json:"batches"`
}

// PrometheusQueryResponse represents the JSON response from the Prometheus query API.
type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []json.RawMessage `json:"result"`
	} `json:"data"`
}
