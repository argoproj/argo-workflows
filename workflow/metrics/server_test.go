//go:build !windows

package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisableMetricsServer(t *testing.T) {
	config := ServerConfig{
		Enabled: false,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.RunServer(ctx, false)
	time.Sleep(1 * time.Second) // to confirm that the server doesn't start, even if we wait
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	if resp != nil {
		defer resp.Body.Close()
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused") // expect that the metrics server not to start
}

func TestMetricsServer(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.RunServer(ctx, false)
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyString := string(bodyBytes)
	assert.NotEmpty(t, bodyString)
}

func TestDummyMetricsServer(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.RunServer(ctx, true)
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyString := string(bodyBytes)

	assert.Empty(t, bodyString) // expect the dummy metrics server to provide no metrics responses
}
