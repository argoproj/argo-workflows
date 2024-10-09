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
)

// testScopeName is the name that the metrics running under test will have
const testScopeName string = "argo-workflows-test"

func TestDisablePrometheusServer(t *testing.T) {
	config := Config{
		Enabled: false,
		Path:    DefaultPrometheusServerPath,
		Port:    DefaultPrometheusServerPort,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := NewMetrics(ctx, testScopeName, testScopeName, &config)
	require.NoError(t, err)
	go m.RunPrometheusServer(ctx, false, &sync.WaitGroup{})
	time.Sleep(1 * time.Second) // to confirm that the server doesn't start, even if we wait
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultPrometheusServerPort, DefaultPrometheusServerPath))
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := NewMetrics(ctx, testScopeName, testScopeName, &config)
	require.NoError(t, err)
	go m.RunPrometheusServer(ctx, false, &wg)
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultPrometheusServerPort, DefaultPrometheusServerPath))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyString := string(bodyBytes)
	assert.NotEmpty(t, bodyString)

	wg.Wait() // wait for server shutdown to prevent port conflicts with subsequent tests
}

func TestDummyPrometheusServer(t *testing.T) {
	var wg sync.WaitGroup
	config := Config{
		Enabled: true,
		Path:    DefaultPrometheusServerPath,
		Port:    DefaultPrometheusServerPort,
		Secure:  false,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := NewMetrics(ctx, testScopeName, testScopeName, &config)
	require.NoError(t, err)
	go m.RunPrometheusServer(ctx, true, &wg)
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultPrometheusServerPort, DefaultPrometheusServerPath))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyString := string(bodyBytes)

	assert.Empty(t, bodyString) // expect the dummy metrics server to provide no metrics responses

	wg.Wait() // wait for server shutdown to prevent port conflicts with subsequent tests
}
