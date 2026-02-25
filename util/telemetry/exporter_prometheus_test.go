//go:build !windows

package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// testScopeName is the name that the metrics running under test will have
const testScopeName string = "argo-workflows-test"

func TestDisablePrometheusServer(t *testing.T) {
	config := Config{
		Enabled: false,
		Path:    DefaultPrometheusServerPath,
		Port:    DefaultPrometheusServerPort,
	}
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	m, err := NewMetrics(ctx, testScopeName, testScopeName, &config)
	require.NoError(t, err)
	m.RunPrometheusServer(ctx, false)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d%s", DefaultPrometheusServerPort, DefaultPrometheusServerPath), nil)
	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	require.ErrorContains(t, err, "connection refused") // expect that the metrics server not to start
}

func TestPrometheusServer(t *testing.T) {
	var wg sync.WaitGroup
	config := Config{
		Enabled: true,
		Path:    DefaultPrometheusServerPath,
		Port:    DefaultPrometheusServerPort,
	}
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	m, err := NewMetrics(ctx, testScopeName, testScopeName, &config)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		m.RunPrometheusServer(ctx, false)
		wg.Done()
	}()
	time.Sleep(1 * time.Second)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d%s", DefaultPrometheusServerPort, DefaultPrometheusServerPath), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyString := string(bodyBytes)
	assert.NotEmpty(t, bodyString)

	cancel() // cancel and wait for server shutdown to prevent port conflicts with subsequent tests
	wg.Wait()
}

func TestDummyPrometheusServer(t *testing.T) {
	var wg sync.WaitGroup
	config := Config{
		Enabled: true,
		Path:    DefaultPrometheusServerPath,
		Port:    DefaultPrometheusServerPort,
		Secure:  false,
	}
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	m, err := NewMetrics(ctx, testScopeName, testScopeName, &config)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		m.RunPrometheusServer(ctx, true)
		wg.Done()
	}()
	time.Sleep(1 * time.Second)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d%s", DefaultPrometheusServerPort, DefaultPrometheusServerPath), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyString := string(bodyBytes)

	assert.Empty(t, bodyString) // expect the dummy metrics server to provide no metrics responses

	cancel() // cancel and wait for server shutdown to prevent port conflicts with subsequent tests
	wg.Wait()
}
